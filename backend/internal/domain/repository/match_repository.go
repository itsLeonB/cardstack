package repository

import (
	"context"

	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
)

// MatchRepository is the persistence access the scan match endpoint needs.
type MatchRepository interface {
	// RandomCards returns up to limit Cards picked at random from the whole
	// catalog, with the joins a CardSummary needs.
	RandomCards(ctx context.Context, limit int) ([]CardResult, error)
}

// matchRepository embeds a crud.Repository only for GetGormInstance.
type matchRepository struct {
	crud.Repository[entity.Card]
}

// NewMatchRepository builds a MatchRepository over base's database.
func NewMatchRepository(base crud.Repository[entity.Card]) MatchRepository {
	return &matchRepository{Repository: base}
}

// RandomCards orders by random(), which sorts the whole table; fine for the
// stub's catalog size.
func (r *matchRepository) RandomCards(ctx context.Context, limit int) ([]CardResult, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var results []CardResult
	err = db.Table("cards").
		Joins("JOIN rarities ON rarities.id = cards.rarity_id").
		Joins("JOIN expansion_sets ON expansion_sets.id = cards.expansion_set_id").
		Select(cardResultColumns).
		Order("random()").
		Limit(limit).
		Find(&results).
		Error

	return results, err
}
