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

	require.NoError(t, repo.Upsert(ctx, []entity.CardEmbedding{catalogEmbedding(far, model, unitVector(1))}))
	require.NoError(t, repo.Upsert(ctx, []entity.CardEmbedding{catalogEmbedding(exact, model, unitVector(0))}))
	require.NoError(t, repo.Upsert(ctx, []entity.CardEmbedding{catalogEmbedding(near, model, mixedVector(0, 2))}))
	require.NoError(t, repo.Upsert(ctx, []entity.CardEmbedding{catalogEmbedding(otherModel, "other-"+model, unitVector(0))}))

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

func TestEmbeddingRepository_UpsertReplacesTheRowForTheSameModelAndSource(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	repo := NewEmbeddingRepository(crud.NewRepository[entity.CardEmbedding](db))
	ctx := context.Background()
	card := newHostedCard(t, fixture, db, set.ID, "cards/replace")

	require.NoError(t, repo.Upsert(ctx, []entity.CardEmbedding{catalogEmbedding(card, "model", unitVector(0))}))
	require.NoError(t, repo.Upsert(ctx, []entity.CardEmbedding{catalogEmbedding(card, "model", unitVector(1))}))
	require.NoError(t, repo.Upsert(ctx, nil))

	var rows []entity.CardEmbedding
	require.NoError(t, db.Where("card_id = ?", card.ID).Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, unitVector(1), rows[0].Embedding.Slice())
}

func TestEmbeddingRepository_AModelChangeAddsARowInsteadOfReplacingOne(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	repo := NewEmbeddingRepository(crud.NewRepository[entity.CardEmbedding](db))
	ctx := context.Background()
	oldModel := "old-" + uuid.NewString()
	card := newHostedCard(t, fixture, db, set.ID, "cards/two-models")

	require.NoError(t, repo.Upsert(ctx, []entity.CardEmbedding{catalogEmbedding(card, oldModel, unitVector(0))}))
	require.NoError(t, repo.Upsert(ctx, []entity.CardEmbedding{catalogEmbedding(card, "new-"+oldModel, unitVector(1))}))

	var rows int64
	require.NoError(t, db.Model(&entity.CardEmbedding{}).Where("card_id = ?", card.ID).Count(&rows).Error)
	assert.Equal(t, int64(2), rows)

	old, err := repo.NearestCards(ctx, oldModel, unitVector(0), 10)
	require.NoError(t, err)
	require.Len(t, old, 1, "the old model's row still answers for its model")
	assert.Equal(t, card.ID, old[0].CardID)
}

func TestEmbeddingRepository_NearestCardsReturnsEveryRowOfACard(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	repo := NewEmbeddingRepository(crud.NewRepository[entity.CardEmbedding](db))
	ctx := context.Background()
	model := "rows-model-" + uuid.NewString()
	card := newHostedCard(t, fixture, db, set.ID, "cards/two-sources")

	require.NoError(t, repo.Upsert(ctx, []entity.CardEmbedding{
		catalogEmbedding(card, model, unitVector(0)),
		{CardID: card.ID, Model: model, Source: "crop", Embedding: pgvector.NewVector(mixedVector(0, 2))},
	}))

	got, err := repo.NearestCards(ctx, model, unitVector(0), 5)
	require.NoError(t, err)
	require.Len(t, got, 2, "a card with two rows is returned twice; the matcher picks the best row")
	assert.Equal(t, []uuid.UUID{card.ID, card.ID}, []uuid.UUID{got[0].CardID, got[1].CardID})
	assert.InDelta(t, 0, got[0].Distance, 1e-6)
	assert.InDelta(t, 0.4, got[1].Distance, 1e-6)
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

	require.NoError(t, repo.Upsert(ctx, []entity.CardEmbedding{catalogEmbedding(embedded, model, unitVector(0))}))
	require.NoError(t, repo.Upsert(ctx, []entity.CardEmbedding{catalogEmbedding(staleModel, "previous-"+model, unitVector(0))}))

	states, err := repo.CountEmbeddingStates(ctx, model, "catalog", set.Code)
	require.NoError(t, err)
	assert.Equal(t, int64(4), states.Total)
	assert.Equal(t, int64(1), states.NoImage)
	assert.Equal(t, int64(1), states.Embedded)
	assert.Zero(t, states.InFlight)

	pendingCards, err := repo.ListPendingCards(ctx, model, "catalog", set.Code)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{pending.ID, staleModel.ID}, cardIDs(pendingCards), "no image and already embedded by this model are skipped; another model's embedding is pending")
	for _, c := range pendingCards {
		assert.NotEmpty(t, c.ImageKey)
	}

	everySet, err := repo.ListPendingCards(ctx, model, "catalog", "")
	require.NoError(t, err)
	assert.Contains(t, cardIDs(everySet), outside.ID, "an empty set code scopes to every set")
	assert.NotContains(t, cardIDs(everySet), noImage.ID, "a card with no hosted image is never pending")

	scoped, err := repo.ListPendingCards(ctx, model, "catalog", otherSet.Code)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{outside.ID}, cardIDs(scoped))

	otherSource, err := repo.ListPendingCards(ctx, model, "crop", set.Code)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{pending.ID, staleModel.ID, embedded.ID}, cardIDs(otherSource), "an embedding of another source does not count")
}

func TestEmbeddingRepository_OutstandingBatchCardsAreNotPending(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	repo := NewEmbeddingRepository(crud.NewRepository[entity.CardEmbedding](db))
	ctx := context.Background()
	model := "batch-model-" + uuid.NewString()

	outstanding := newHostedCard(t, fixture, db, set.ID, "cards/outstanding")
	failedBatch := newHostedCard(t, fixture, db, set.ID, "cards/failed-batch")
	otherModelBatch := newHostedCard(t, fixture, db, set.ID, "cards/other-model-batch")

	submitted := entity.EmbeddingBatch{JobName: "batches/" + uuid.NewString(), Model: model, Source: "catalog", State: entity.EmbeddingBatchSubmitted}
	require.NoError(t, repo.CreateBatch(ctx, &submitted, []uuid.UUID{outstanding.ID}))

	failed := entity.EmbeddingBatch{JobName: "batches/" + uuid.NewString(), Model: model, Source: "catalog", State: entity.EmbeddingBatchSubmitted}
	require.NoError(t, repo.CreateBatch(ctx, &failed, []uuid.UUID{failedBatch.ID}))
	require.NoError(t, repo.SetBatchState(ctx, failed.ID, entity.EmbeddingBatchFailed))

	other := entity.EmbeddingBatch{JobName: "batches/" + uuid.NewString(), Model: "other-" + model, Source: "catalog", State: entity.EmbeddingBatchSubmitted}
	require.NoError(t, repo.CreateBatch(ctx, &other, []uuid.UUID{otherModelBatch.ID}))

	pending, err := repo.ListPendingCards(ctx, model, "catalog", set.Code)
	require.NoError(t, err)
	ids := cardIDs(pending)
	assert.NotContains(t, ids, outstanding.ID, "a card in a submitted batch for this model is not pending")
	assert.Contains(t, ids, failedBatch.ID, "a failed batch's cards are pending again")
	assert.Contains(t, ids, otherModelBatch.ID, "a batch for another model does not hold this model's cards")

	states, err := repo.CountEmbeddingStates(ctx, model, "catalog", set.Code)
	require.NoError(t, err)
	assert.Equal(t, int64(1), states.InFlight)
}

func TestEmbeddingRepository_BatchLifecycleStampsCollectedAt(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	repo := NewEmbeddingRepository(crud.NewRepository[entity.CardEmbedding](db))
	ctx := context.Background()
	card := newHostedCard(t, fixture, db, set.ID, "cards/lifecycle")

	batch := entity.EmbeddingBatch{JobName: "batches/" + uuid.NewString(), Model: "lifecycle-" + uuid.NewString(), Source: "catalog", State: entity.EmbeddingBatchSubmitted}
	require.NoError(t, repo.CreateBatch(ctx, &batch, []uuid.UUID{card.ID}))

	submitted, err := repo.ListSubmittedBatches(ctx)
	require.NoError(t, err)
	assert.Contains(t, batchIDs(submitted), batch.ID)

	require.NoError(t, repo.SetBatchState(ctx, batch.ID, entity.EmbeddingBatchCollected))

	submitted, err = repo.ListSubmittedBatches(ctx)
	require.NoError(t, err)
	assert.NotContains(t, batchIDs(submitted), batch.ID, "a collected batch is not awaiting collection")

	var stored entity.EmbeddingBatch
	require.NoError(t, db.First(&stored, "id = ?", batch.ID).Error)
	assert.Equal(t, entity.EmbeddingBatchCollected, stored.State)
	require.NotNil(t, stored.CollectedAt)
}

// catalogEmbedding is the embedding of the card's catalog image from model.
func catalogEmbedding(card entity.Card, model string, v []float32) entity.CardEmbedding {
	return entity.CardEmbedding{CardID: card.ID, Model: model, Source: "catalog", Embedding: pgvector.NewVector(v)}
}

func batchIDs(batches []entity.EmbeddingBatch) []uuid.UUID {
	out := make([]uuid.UUID, len(batches))
	for i, b := range batches {
		out[i] = b.ID
	}
	return out
}

func cardIDs(cards []entity.Card) []uuid.UUID {
	out := make([]uuid.UUID, len(cards))
	for i, c := range cards {
		out[i] = c.ID
	}
	return out
}
