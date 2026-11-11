package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// TradeRepository is crud.Repository[entity.Trade] plus the Trade's lookups:
// by slug, and the conditional join that claims the second seat.
type TradeRepository interface {
	crud.Repository[entity.Trade]
	// FindBySlug returns the Trade with this slug, preloading its Trade Items
	// with their Cards (Rarity and Expansion Set) and source Collections, or
	// the zero value with a nil error when there is none.
	FindBySlug(ctx context.Context, slug string) (entity.Trade, error)
	// Join claims the second seat for profileID when it is empty, moving the
	// Trade to negotiating. It reports false, changing nothing, when the seat
	// is taken or the Trade is no longer open — one conditional UPDATE, so two
	// joins at once leave exactly one winner.
	Join(ctx context.Context, id, profileID uuid.UUID) (bool, error)
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
