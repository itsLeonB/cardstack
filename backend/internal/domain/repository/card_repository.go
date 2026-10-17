package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// CardRepository is crud.Repository[entity.Card] plus the per-set aggregate
// go-crud has no equivalent for.
type CardRepository interface {
	crud.Repository[entity.Card]
	// CountsByExpansionSet returns the Card count and the count without a
	// hosted image for each Expansion Set of the Game that has Cards. A set
	// with no Cards is absent from the map.
	CountsByExpansionSet(ctx context.Context, gameID uuid.UUID) (map[uuid.UUID]CardCounts, error)
}

// CardCounts is one Expansion Set's Card totals.
type CardCounts struct {
	Total        int
	WithoutImage int
}

type cardRepository struct {
	crud.Repository[entity.Card]
}

func NewCardRepository(base crud.Repository[entity.Card]) CardRepository {
	return &cardRepository{Repository: base}
}

func (r *cardRepository) CountsByExpansionSet(ctx context.Context, gameID uuid.UUID) (map[uuid.UUID]CardCounts, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var rows []struct {
		ExpansionSetID uuid.UUID
		Total          int
		WithoutImage   int
	}
	// The join scopes the Cards to the Game in the query itself.
	err = db.Table("cards AS c").
		Select("c.expansion_set_id, COUNT(*) AS total, COUNT(*) FILTER (WHERE c.image_key = '') AS without_image").
		Joins("JOIN expansion_sets AS es ON es.id = c.expansion_set_id").
		Where("es.game_id = ?", gameID).
		Group("c.expansion_set_id").
		Scan(&rows).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "counting cards by expansion set")
	}

	counts := make(map[uuid.UUID]CardCounts, len(rows))
	for _, row := range rows {
		counts[row.ExpansionSetID] = CardCounts{Total: row.Total, WithoutImage: row.WithoutImage}
	}

	return counts, nil
}
