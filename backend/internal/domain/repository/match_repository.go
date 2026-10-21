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
	// ReprintsOf returns the Cards of the Reprint rule for source, leaving out
	// exclude: same normalized name, an embedding of model within maxDistance
	// of source's, in a published Expansion Set. Each Card is returned once,
	// with the nearest distance of its embeddings, ordered by distance, then
	// newest Expansion Set first. A source without an embedding returns nothing.
	ReprintsOf(ctx context.Context, model string, source uuid.UUID, exclude []uuid.UUID, maxDistance float64) ([]CardDistance, error)
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

func (r *matchRepository) ReprintsOf(ctx context.Context, model string, source uuid.UUID, exclude []uuid.UUID, maxDistance float64) ([]CardDistance, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	// The art test compares every pair of the two Cards' embeddings for model
	// and keeps the nearest, so a Card with several rows is judged by its
	// nearest one.
	var reprints []CardDistance
	err = db.Table("cards").
		Select("cards.id AS card_id, MIN(ce.embedding <=> se.embedding) AS distance").
		Joins("JOIN expansion_sets ON expansion_sets.id = cards.expansion_set_id AND expansion_sets.published = true").
		Joins("JOIN card_embeddings ce ON ce.card_id = cards.id AND ce.model = ?", model).
		Joins("JOIN cards src ON src.id = ?", source).
		Joins("JOIN card_embeddings se ON se.card_id = src.id AND se.model = ?", model).
		Where(normalizedName("cards.name")+" = "+normalizedName("src.name")).
		Where("cards.id NOT IN ?", exclude).
		Group("cards.id, expansion_sets.release_date").
		Having("MIN(ce.embedding <=> se.embedding) <= ?", maxDistance).
		Order("distance, expansion_sets.release_date DESC NULLS LAST, cards.id").
		Scan(&reprints).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "searching reprints")
	}
	return reprints, nil
}

// normalizedName is the Card name the Reprint rule compares: case and runs of
// whitespace folded. It must match idx_cards_normalized_name, or the lookup
// scans the table.
func normalizedName(column string) string {
	return "btrim(regexp_replace(lower(" + column + "), '\\s+', ' ', 'g'))"
}
