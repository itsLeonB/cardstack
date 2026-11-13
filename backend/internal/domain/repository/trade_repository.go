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
	// OfferChanged records that the offer moved: it bumps the Trade's
	// updated_at — the revision an accept compares against — and clears
	// both seats' acceptances, because an edit after an acceptance means
	// the deal they agreed to no longer stands. Item edits call it in the
	// same transaction while the caller holds the Trade's row lock.
	OfferChanged(ctx context.Context, trade entity.Trade) error
	// Accept stamps one seat's acceptance on the locked Trade row and, when
	// the other seat already accepted, moves it to locked. The caller holds
	// the row lock (FindBySlugForUpdate) and has already checked the rule.
	Accept(ctx context.Context, trade entity.Trade, side int) error
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

// AddItem inserts the Trade Item and bumps the Trade's offer revision in
// the same transaction, so an accept that read the offer before this edit
// refuses as stale. The caller holds the Trade's row lock.
func (r *tradeRepository) AddItem(ctx context.Context, item entity.TradeItem) error {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}
	if err := db.Create(&item).Error; err != nil {
		return ungerr.Wrap(err, "adding trade item")
	}
	trade := entity.Trade{BaseEntity: crud.BaseEntity{ID: item.TradeID}}
	return r.OfferChanged(ctx, trade)
}

// Accept stamps one seat's acceptance and, when the other seat already
// accepted, locks the Trade: the row the caller's transaction already holds
// FOR UPDATE serializes any racing accept, so the entity checks here are
// the whole rule — never a conditional UPDATE whose outcome RowsAffected
// would have to infer.
func (r *tradeRepository) Accept(ctx context.Context, trade entity.Trade, side int) error {
	now := time.Now()
	switch side {
	case entity.SideOwner:
		trade.OwnerAcceptedAt = &now
	case entity.SideParticipant:
		trade.ParticipantAcceptedAt = &now
	}
	if trade.OwnerAcceptedAt != nil && trade.ParticipantAcceptedAt != nil &&
		trade.Status == entity.TradeStatusNegotiating {
		trade.Status = entity.TradeStatusLocked
	}
	trade.UpdatedAt = now

	_, err := r.Update(ctx, trade)
	return err
}

// OfferChanged records that the offer moved: it bumps the Trade's
// updated_at — the revision an accept compares against — and clears both
// seats' acceptances, because an edit after an acceptance means the deal
// they agreed to no longer stands. Item edits call it in the same
// transaction while the caller holds the Trade's row lock.
// OfferChanged records that the offer moved: it bumps the Trade's
// updated_at — the revision an accept compares against — and clears both
// seats' acceptances, because an edit after an acceptance means the deal
// they agreed to no longer stands. It writes only those columns, so the
// caller need not pass the whole row. Item edits call it in the same
// transaction while the caller holds the Trade's row lock.
func (r *tradeRepository) OfferChanged(ctx context.Context, trade entity.Trade) error {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}
	result := db.Model(&entity.Trade{}).
		Where("id = ?", trade.ID).
		Updates(map[string]any{
			"owner_accepted_at":       nil,
			"participant_accepted_at": nil,
			"updated_at":              time.Now(),
		})
	if result.Error != nil {
		return ungerr.Wrap(result.Error, "recording trade offer change")
	}
	return nil
}

// RemoveItem deletes the Trade Item and bumps the Trade's offer revision
// in the same transaction. The caller holds the Trade's row lock and has
// already checked the item belongs to the caller's side.
func (r *tradeRepository) RemoveItem(ctx context.Context, tradeID, itemID uuid.UUID, side int) (bool, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return false, err
	}
	result := db.Where("trade_id = ? AND id = ? AND side = ?", tradeID, itemID, side).Delete(&entity.TradeItem{})
	if result.Error != nil {
		return false, ungerr.Wrap(result.Error, "removing trade item")
	}
	if result.RowsAffected != 1 {
		return false, nil
	}
	return true, r.OfferChanged(ctx, entity.Trade{BaseEntity: crud.BaseEntity{ID: tradeID}})
}
