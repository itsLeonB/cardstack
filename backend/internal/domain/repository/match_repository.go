package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// MatchRepository is the persistence access the scan match endpoint needs.
type MatchRepository interface {
	// CardsByIDs returns the Cards with the given ids, in no particular order,
	// with the joins a CardSummary needs. An unknown id is simply absent.
	CardsByIDs(ctx context.Context, ids []uuid.UUID) ([]CardResult, error)
}

// matchRepository embeds a crud.Repository only for GetGormInstance.
type matchRepository struct {
	crud.Repository[entity.Card]
}

// NewMatchRepository builds a MatchRepository over base's database.
func NewMatchRepository(base crud.Repository[entity.Card]) MatchRepository {
	return &matchRepository{Repository: base}
}

func (r *matchRepository) CardsByIDs(ctx context.Context, ids []uuid.UUID) ([]CardResult, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var results []CardResult
	err = withCardResultJoins(db.Table("cards")).
		Select(cardResultColumns).
		Where("cards.id IN ?", ids).
		Find(&results).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "loading matched cards")
	}
	return results, nil
}
