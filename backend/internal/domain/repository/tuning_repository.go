package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// TuningRepository is the read-only access the tune-scan report needs. Every
// method is a SELECT; the report never writes.
type TuningRepository interface {
	// HasEmbeddings reports whether any Card has an embedding of model.
	HasEmbeddings(ctx context.Context, model string) (bool, error)
	// NameGroups returns the normalized names that at least two Cards with an
	// embedding of model share, largest group first, ties by name.
	NameGroups(ctx context.Context, model string) ([]NameGroup, error)
	// GroupCards returns the Cards with an embedding of model whose normalized
	// name is name, ordered by set code, then local id.
	GroupCards(ctx context.Context, model, name string) ([]GroupCard, error)
	// PairDistances returns each pair of the given Cards that both have an
	// embedding of model, once, with the nearest distance between their
	// embeddings: the rule ReprintsOf applies. A Card with no embedding of
	// model forms no pair.
	PairDistances(ctx context.Context, model string, cardIDs []uuid.UUID) ([]CardPair, error)
}

// NameGroup is a normalized Card name and how many embedded Cards share it.
type NameGroup struct {
	Name string
	Size int
}

// GroupCard identifies one Card of a name group by its Expansion Set code and
// its local id, the two parts an operator reads a printing by.
type GroupCard struct {
	ID      uuid.UUID
	SetCode string
	LocalID string
}

// CardPair is the nearest distance between the embeddings of two Cards, A
// before B.
type CardPair struct {
	A        uuid.UUID
	B        uuid.UUID
	Distance float64
}

// tuningRepository embeds a crud.Repository only for GetGormInstance.
type tuningRepository struct {
	crud.Repository[entity.Card]
}

// NewTuningRepository builds a TuningRepository over base's database.
func NewTuningRepository(base crud.Repository[entity.Card]) TuningRepository {
	return &tuningRepository{Repository: base}
}

func (r *tuningRepository) HasEmbeddings(ctx context.Context, model string) (bool, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return false, err
	}

	var exists bool
	err = db.Raw("SELECT EXISTS (SELECT 1 FROM card_embeddings WHERE model = ?)", model).
		Scan(&exists).
		Error
	if err != nil {
		return false, ungerr.Wrap(err, "checking for embeddings")
	}
	return exists, nil
}

func (r *tuningRepository) NameGroups(ctx context.Context, model string) ([]NameGroup, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var groups []NameGroup
	err = db.Table("cards").
		Select(normalizedName("cards.name")+" AS name, COUNT(DISTINCT cards.id) AS size").
		Joins("JOIN card_embeddings ce ON ce.card_id = cards.id AND ce.model = ?", model).
		Group(normalizedName("cards.name")).
		Having("COUNT(DISTINCT cards.id) >= 2").
		Order("size DESC, name ASC").
		Scan(&groups).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "grouping cards by name")
	}
	return groups, nil
}

func (r *tuningRepository) GroupCards(ctx context.Context, model, name string) ([]GroupCard, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var cards []GroupCard
	err = db.Table("cards").
		Select("cards.id AS id, expansion_sets.code AS set_code, cards.local_id AS local_id").
		Joins("JOIN expansion_sets ON expansion_sets.id = cards.expansion_set_id").
		Where(normalizedName("cards.name")+" = ?", name).
		Where("EXISTS (SELECT 1 FROM card_embeddings ce WHERE ce.card_id = cards.id AND ce.model = ?)", model).
		Order("expansion_sets.code, cards.local_id, cards.id").
		Scan(&cards).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "listing cards of a name group")
	}
	return cards, nil
}

func (r *tuningRepository) PairDistances(ctx context.Context, model string, cardIDs []uuid.UUID) ([]CardPair, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	// The ordering join keeps each unordered pair once. MIN over the pair's
	// rows is the nearest of their embeddings, as ReprintsOf compares them.
	var pairs []CardPair
	err = db.Table("card_embeddings AS ea").
		Select("ea.card_id AS a, eb.card_id AS b, MIN(ea.embedding <=> eb.embedding) AS distance").
		Joins("JOIN card_embeddings eb ON eb.model = ea.model AND eb.card_id > ea.card_id").
		Where("ea.model = ? AND ea.card_id IN ? AND eb.card_id IN ?", model, cardIDs, cardIDs).
		Group("ea.card_id, eb.card_id").
		Scan(&pairs).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "measuring distances within a name group")
	}
	return pairs, nil
}
