package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// insertTuningEmbedding stores v as cardID's embedding of model.
func insertTuningEmbedding(t *testing.T, db *gorm.DB, cardID uuid.UUID, model string, v []float32) {
	t.Helper()
	require.NoError(t, db.Create(&entity.CardEmbedding{CardID: cardID, Model: model, Source: "catalog", Embedding: pgvector.NewVector(v)}).Error)
}

// pairDistance finds the distance of the pair x and y in either order.
func pairDistance(t *testing.T, pairs []CardPair, x, y uuid.UUID) float64 {
	t.Helper()
	for _, p := range pairs {
		if (p.A == x && p.B == y) || (p.A == y && p.B == x) {
			return p.Distance
		}
	}
	t.Fatalf("no pair for %s and %s in %v", x, y, pairs)
	return 0
}

func TestTuningRepository_HasEmbeddingsIsScopedToTheModel(t *testing.T) {
	db := testDB(t)
	repo := NewTuningRepository(crud.NewRepository[entity.Card](db))
	f := newCatalogFixture(t, db)
	card := f.newCard(t, db, f.newExpansionSet(t, db, nil, nil).ID, nil)
	model := "tuning-" + uuid.NewString()
	ctx := context.Background()

	before, err := repo.HasEmbeddings(ctx, model)
	require.NoError(t, err)
	assert.False(t, before)

	insertTuningEmbedding(t, db, card.ID, model, unitVector(0))

	after, err := repo.HasEmbeddings(ctx, model)
	require.NoError(t, err)
	assert.True(t, after)
	other, err := repo.HasEmbeddings(ctx, "other-"+model)
	require.NoError(t, err)
	assert.False(t, other, "an embedding of another model does not count")
}

func TestTuningRepository_NameGroupsLargestFirstAndOnlyEmbeddedCards(t *testing.T) {
	db := testDB(t)
	repo := NewTuningRepository(crud.NewRepository[entity.Card](db))
	f := newCatalogFixture(t, db)
	set := f.newExpansionSet(t, db, nil, nil)
	model := "tuning-" + uuid.NewString()
	suffix := uuid.NewString()
	ctx := context.Background()

	// newNamedCard inserts a Card with the given name, embedded in model when embedded is set.
	newNamedCard := func(name string, embedded bool) {
		card := f.newCard(t, db, set.ID, func(c *entity.Card) { c.Name = name; c.LocalID = uuid.NewString() })
		if embedded {
			insertTuningEmbedding(t, db, card.ID, model, unitVector(0))
		}
	}
	// One name in three spellings is one group of three.
	newNamedCard("Tune Pikachu "+suffix, true)
	newNamedCard("tune  pikachu "+suffix, true)
	newNamedCard("TUNE PIKACHU "+suffix, true)
	newNamedCard("Tune Pikachu "+suffix, false)
	// Two groups of two tie on size, so the name decides the order.
	newNamedCard("Tune Energy "+suffix, true)
	newNamedCard("Tune Energy "+suffix, true)
	newNamedCard("Tune Able "+suffix, true)
	newNamedCard("Tune Able "+suffix, true)
	// A group of one, and a group of two with only one embedded Card, are left out.
	newNamedCard("Tune Solo "+suffix, true)
	newNamedCard("Tune Twin "+suffix, true)
	newNamedCard("Tune Twin "+suffix, false)

	// Cards in an unpublished set are not Reprints, so they are not counted: two of them make no group.
	hidden := f.newExpansionSet(t, db, nil, nil)
	require.NoError(t, db.Model(&hidden).Update("published", false).Error)
	for range 2 {
		card := f.newCard(t, db, hidden.ID, func(c *entity.Card) { c.Name = "Tune Hidden " + suffix; c.LocalID = uuid.NewString() })
		insertTuningEmbedding(t, db, card.ID, model, unitVector(0))
	}

	groups, err := repo.NameGroups(ctx, model)

	require.NoError(t, err)
	assert.Equal(t, []NameGroup{
		{Name: "tune pikachu " + suffix, Size: 3},
		{Name: "tune able " + suffix, Size: 2},
		{Name: "tune energy " + suffix, Size: 2},
	}, groups)
}

func TestTuningRepository_GroupCardsOfOneNameWithTheirSetAndNumber(t *testing.T) {
	db := testDB(t)
	repo := NewTuningRepository(crud.NewRepository[entity.Card](db))
	f := newCatalogFixture(t, db)
	set := f.newExpansionSet(t, db, nil, nil)
	model := "tuning-" + uuid.NewString()
	name := "tune group " + uuid.NewString()
	ctx := context.Background()

	second := f.newCard(t, db, set.ID, func(c *entity.Card) { c.Name = name; c.LocalID = "002" })
	first := f.newCard(t, db, set.ID, func(c *entity.Card) { c.Name = name; c.LocalID = "001" })
	insertTuningEmbedding(t, db, second.ID, model, unitVector(0))
	insertTuningEmbedding(t, db, first.ID, model, unitVector(1))
	// Not embedded, or another name: neither is in the group.
	f.newCard(t, db, set.ID, func(c *entity.Card) { c.Name = name; c.LocalID = "003" })
	other := f.newCard(t, db, set.ID, func(c *entity.Card) { c.Name = name + " ex"; c.LocalID = "004" })
	insertTuningEmbedding(t, db, other.ID, model, unitVector(2))

	// A Card in an unpublished set is not a Reprint, so it is not listed.
	hidden := f.newExpansionSet(t, db, nil, nil)
	require.NoError(t, db.Model(&hidden).Update("published", false).Error)
	unpublished := f.newCard(t, db, hidden.ID, func(c *entity.Card) { c.Name = name; c.LocalID = "005" })
	insertTuningEmbedding(t, db, unpublished.ID, model, unitVector(3))

	got, err := repo.GroupCards(ctx, model, name)

	require.NoError(t, err)
	assert.Equal(t, []GroupCard{
		{ID: first.ID, SetCode: set.Code, LocalID: "001"},
		{ID: second.ID, SetCode: set.Code, LocalID: "002"},
	}, got)
}

func TestTuningRepository_PairDistancesUseTheNearestPairOfEmbeddings(t *testing.T) {
	db := testDB(t)
	repo := NewTuningRepository(crud.NewRepository[entity.Card](db))
	f := newCatalogFixture(t, db)
	set := f.newExpansionSet(t, db, nil, nil)
	model := "tuning-" + uuid.NewString()
	ctx := context.Background()

	a := newHostedCard(t, f, db, set.ID, "cards/tuning")
	b := newHostedCard(t, f, db, set.ID, "cards/tuning")
	c := newHostedCard(t, f, db, set.ID, "cards/tuning")
	outside := newHostedCard(t, f, db, set.ID, "cards/tuning")
	// A holds two rows: the pair with B takes the nearer one (0.4), not the farther (1.0).
	insertTuningEmbedding(t, db, a.ID, model, unitVector(0))
	require.NoError(t, db.Create(&entity.CardEmbedding{CardID: a.ID, Model: model, Source: "crop", Embedding: pgvector.NewVector(unitVector(2))}).Error)
	insertTuningEmbedding(t, db, b.ID, model, mixedVector(0, 1))
	insertTuningEmbedding(t, db, c.ID, model, unitVector(2))
	insertTuningEmbedding(t, db, outside.ID, model, unitVector(0))

	pairs, err := repo.PairDistances(ctx, model, []uuid.UUID{a.ID, b.ID, c.ID})

	require.NoError(t, err)
	assert.Len(t, pairs, 3, "each pair of the given Cards once, and none with the Card left out")
	assert.InDelta(t, 0.4, pairDistance(t, pairs, a.ID, b.ID), 1e-4)
	assert.InDelta(t, 0.0, pairDistance(t, pairs, a.ID, c.ID), 1e-4)
	assert.InDelta(t, 1.0, pairDistance(t, pairs, b.ID, c.ID), 1e-4)
}
