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

const testEmbeddingDims = 1536

// newHostedCard inserts a card with a unique local ID, so one set can hold
// many of them; the fixture's default local ID is shared.
func newHostedCard(t *testing.T, fixture catalogFixture, db *gorm.DB, setID uuid.UUID, imageKey string) entity.Card {
	t.Helper()
	return fixture.newCard(t, db, setID, func(c *entity.Card) {
		c.LocalID = uuid.NewString()
		c.ImageKey = imageKey
	})
}

// unitVector is a 1536-dimensional vector with a single 1 at axis.
func unitVector(axis int) []float32 {
	v := make([]float32, testEmbeddingDims)
	v[axis] = 1
	return v
}

// mixedVector is 0.6 along axis a and 0.8 along axis b: its cosine with
// unitVector(a) is 0.6, so its cosine distance from it is 0.4.
func mixedVector(a, b int) []float32 {
	v := make([]float32, testEmbeddingDims)
	v[a] = 0.6
	v[b] = 0.8
	return v
}

func TestEmbeddingRepository_NearestCardsOrdersByCosineWithinOneModel(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	repo := NewEmbeddingRepository(crud.NewRepository[entity.CardEmbedding](db))
	ctx := context.Background()
	model := "test-model-" + uuid.NewString()

	exact := newHostedCard(t, fixture, db, set.ID, "cards/exact")
	near := newHostedCard(t, fixture, db, set.ID, "cards/near")
	far := newHostedCard(t, fixture, db, set.ID, "cards/far")
	otherModel := newHostedCard(t, fixture, db, set.ID, "cards/other")

	require.NoError(t, repo.Upsert(ctx, entity.CardEmbedding{CardID: far.ID, Model: model, Embedding: pgvector.NewVector(unitVector(1))}))
	require.NoError(t, repo.Upsert(ctx, entity.CardEmbedding{CardID: exact.ID, Model: model, Embedding: pgvector.NewVector(unitVector(0))}))
	require.NoError(t, repo.Upsert(ctx, entity.CardEmbedding{CardID: near.ID, Model: model, Embedding: pgvector.NewVector(mixedVector(0, 2))}))
	require.NoError(t, repo.Upsert(ctx, entity.CardEmbedding{CardID: otherModel.ID, Model: "other-" + model, Embedding: pgvector.NewVector(unitVector(0))}))

	got, err := repo.NearestCards(ctx, model, unitVector(0), 3)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, exact.ID, got[0].CardID)
	assert.InDelta(t, 0, got[0].Distance, 1e-6)
	assert.Equal(t, near.ID, got[1].CardID)
	assert.InDelta(t, 0.4, got[1].Distance, 1e-6)
	assert.Equal(t, far.ID, got[2].CardID)
	assert.InDelta(t, 1, got[2].Distance, 1e-6)

	limited, err := repo.NearestCards(ctx, model, unitVector(0), 1)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{exact.ID}, []uuid.UUID{limited[0].CardID})
}

func TestEmbeddingRepository_UpsertReplacesTheCardsEmbedding(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	repo := NewEmbeddingRepository(crud.NewRepository[entity.CardEmbedding](db))
	ctx := context.Background()
	card := newHostedCard(t, fixture, db, set.ID, "cards/replace")

	require.NoError(t, repo.Upsert(ctx, entity.CardEmbedding{CardID: card.ID, Model: "old", Embedding: pgvector.NewVector(unitVector(0))}))
	require.NoError(t, repo.Upsert(ctx, entity.CardEmbedding{CardID: card.ID, Model: "new", Embedding: pgvector.NewVector(unitVector(1))}))

	var rows []entity.CardEmbedding
	require.NoError(t, db.Where("card_id = ?", card.ID).Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, "new", rows[0].Model)

	old, err := repo.NearestCards(ctx, "old", unitVector(0), 10)
	require.NoError(t, err)
	for _, n := range old {
		assert.NotEqual(t, card.ID, n.CardID, "the replaced row no longer answers for its old model")
	}
}

func TestEmbeddingRepository_CountAndListPendingCardsByModel(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	otherSet := fixture.newExpansionSet(t, db, nil, nil)
	repo := NewEmbeddingRepository(crud.NewRepository[entity.CardEmbedding](db))
	ctx := context.Background()
	model := "count-model-" + uuid.NewString()

	noImage := newHostedCard(t, fixture, db, set.ID, "")
	embedded := newHostedCard(t, fixture, db, set.ID, "cards/embedded")
	pending := newHostedCard(t, fixture, db, set.ID, "cards/pending")
	staleModel := newHostedCard(t, fixture, db, set.ID, "cards/stale")
	outside := newHostedCard(t, fixture, db, otherSet.ID, "cards/outside")

	require.NoError(t, repo.Upsert(ctx, entity.CardEmbedding{CardID: embedded.ID, Model: model, Embedding: pgvector.NewVector(unitVector(0))}))
	require.NoError(t, repo.Upsert(ctx, entity.CardEmbedding{CardID: staleModel.ID, Model: "previous-" + model, Embedding: pgvector.NewVector(unitVector(0))}))

	states, err := repo.CountEmbeddingStates(ctx, model, set.Code)
	require.NoError(t, err)
	assert.Equal(t, int64(4), states.Total)
	assert.Equal(t, int64(1), states.NoImage)
	assert.Equal(t, int64(1), states.Embedded)

	pendingCards, err := repo.ListPendingCards(ctx, model, set.Code)
	require.NoError(t, err)
	ids := make([]uuid.UUID, len(pendingCards))
	for i, c := range pendingCards {
		ids[i] = c.ID
	}
	assert.ElementsMatch(t, []uuid.UUID{pending.ID, staleModel.ID}, ids, "no image and already embedded by this model are skipped; another model's embedding is pending")
	for _, c := range pendingCards {
		assert.NotEmpty(t, c.ImageKey)
	}

	everySet, err := repo.ListPendingCards(ctx, model, "")
	require.NoError(t, err)
	assert.Contains(t, cardIDs(everySet), outside.ID, "an empty set code scopes to every set")
	assert.NotContains(t, cardIDs(everySet), noImage.ID, "a card with no hosted image is never pending")

	scoped, err := repo.ListPendingCards(ctx, model, otherSet.Code)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{outside.ID}, cardIDs(scoped))
}

func cardIDs(cards []entity.Card) []uuid.UUID {
	out := make([]uuid.UUID, len(cards))
	for i, c := range cards {
		out[i] = c.ID
	}
	return out
}
