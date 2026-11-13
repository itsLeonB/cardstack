package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// TradeRepository is crud.Repository[entity.Trade] plus the Trade's lookups:
// by slug, and the conditional join that claims the second seat.
type TradeRepository interface {
	crud.Repository[entity.Trade]
	FindBySlug(ctx context.Context, slug string) (entity.Trade, error)
	FindBySlugForUpdate(ctx context.Context, slug string) (entity.Trade, error)
	Join(ctx context.Context, id, profileID uuid.UUID) (bool, error)
	AddItem(ctx context.Context, item entity.TradeItem) error
	RemoveItem(ctx context.Context, tradeID, itemID uuid.UUID, side int) (bool, error)
	Accept(ctx context.Context, id uuid.UUID, side int) (bool, error)
}

type tradeRepository struct {
	crud.Repository[entity.Trade]
}

func NewTradeRepository(base crud.Repository[entity.Trade]) TradeRepository {
	return &tradeRepository{Repository: base}
}

// tradeItemPreloads load each Trade Item's Card (with Rarity and Expansion
// Set) and its source Collection title.
var tradeItemPreloads = []string{
	"TradeItems.Card.Rarity",
	"TradeItems.Card.ExpansionSet",
	"TradeItems.SourceCollection",
}

func (r *tradeRepository) FindBySlug(ctx context.Context, slug string) (entity.Trade, error) {
	if slug == "" {
		return entity.Trade{}, nil
	}

	return r.FindFirst(ctx, crud.Specification[entity.Trade]{
		Model:            entity.Trade{Slug: slug},
		PreloadRelations: tradeItemPreloads,
	})
}

func (r *tradeRepository) FindBySlugForUpdate(ctx context.Context, slug string) (entity.Trade, error) {
	if slug == "" {
		return entity.Trade{}, nil
	}
	return r.FindFirst(ctx, crud.Specification[entity.Trade]{
		Model:            entity.Trade{Slug: slug},
		ForUpdate:        true,
		PreloadRelations: tradeItemPreloads,
	})
}

// Join is one conditional UPDATE, so two joins racing on the same Trade
// serialize on the row: the second finds participant_profile_id no longer
// NULL and updates nothing. RowsAffected decides the winner, never inferred
// state.
func (r *tradeRepository) Join(ctx context.Context, id, profileID uuid.UUID) (bool, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return false, err
	}

	result := db.Model(&entity.Trade{}).
		Where("id = ? AND participant_profile_id IS NULL AND status = ?", id, entity.TradeStatusOpen).
		Updates(map[string]any{
			"participant_profile_id": profileID,
			"status":                 entity.TradeStatusNegotiating,
		})
	if result.Error != nil {
		return false, ungerr.Wrap(result.Error, "joining trade")
	}

	return result.RowsAffected == 1, nil
}

func (r *tradeRepository) AddItem(ctx context.Context, item entity.TradeItem) error {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}
	if err := db.Create(&item).Error; err != nil {
		return ungerr.Wrap(err, "adding trade item")
	}
	return nil
}

// Accept stamps one seat's acceptance: one conditional UPDATE that lands
// only while the Trade negotiates and that seat has not accepted. A repeat
// accept matches nothing and the caller treats it as idempotent; RowsAffected
// decides whether this call changed anything. When the other seat already
// accepted, the same call also moves the Trade to locked.
func (r *tradeRepository) Accept(ctx context.Context, id uuid.UUID, side int) (bool, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return false, err
	}

	column := "participant_accepted_at"
	other := "owner_accepted_at"
	if side == entity.SideOwner {
		column, other = "owner_accepted_at", "participant_accepted_at"
	}
	now := time.Now()
	result := db.Model(&entity.Trade{}).
		Where("id = ? AND status = ? AND "+column+" IS NULL", id, entity.TradeStatusNegotiating).
		Updates(map[string]any{column: now, "updated_at": now})
	if result.Error != nil {
		return false, ungerr.Wrap(result.Error, "accepting trade")
	}
	if result.RowsAffected != 1 {
		return false, nil
	}

	// The second acceptance locks the Trade: the row lock the caller's
	// transaction already holds serializes any racing accept.
	lock := db.Model(&entity.Trade{}).
		Where("id = ? AND status = ? AND "+other+" IS NOT NULL", id, entity.TradeStatusNegotiating).
		Update("status", entity.TradeStatusLocked)
	if lock.Error != nil {
		return false, ungerr.Wrap(lock.Error, "locking trade")
	}

	return true, nil
}

func (r *tradeRepository) RemoveItem(ctx context.Context, tradeID, itemID uuid.UUID, side int) (bool, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return false, err
	}
	result := db.Where("trade_id = ? AND id = ? AND side = ?", tradeID, itemID, side).Delete(&entity.TradeItem{})
	if result.Error != nil {
		return false, ungerr.Wrap(result.Error, "removing trade item")
	}
	return result.RowsAffected == 1, nil
}
