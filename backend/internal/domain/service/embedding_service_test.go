package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testEmbeddingModel = "gemini-embedding-2"

// embeddingFixture wires the service to mocks of its three seams.
type embeddingFixture struct {
	repo     *mocks.MockEmbeddingRepository
	embedder *mocks.MockBatchEmbedder
	images   *mocks.MockImageFetcher
	service  EmbeddingService
}

func newEmbeddingFixture(t *testing.T) embeddingFixture {
	t.Helper()
	f := embeddingFixture{
		repo:     mocks.NewMockEmbeddingRepository(t),
		embedder: mocks.NewMockBatchEmbedder(t),
		images:   mocks.NewMockImageFetcher(t),
	}
	f.service = NewEmbeddingService(f.repo, f.embedder, f.images, mapper.NewImageHost("https://img.test"), testEmbeddingModel)
	return f
}

// hostedCard is a card with a hosted image.
func hostedCard(key string) entity.Card {
	return entity.Card{BaseEntity: crud.BaseEntity{ID: uuid.New()}, ImageKey: key}
}

// stateCounts is the scope the repository reports before a run.
func stateCounts(total, noImage, embedded, inFlight int64) repository.EmbeddingStates {
	return repository.EmbeddingStates{Total: total, NoImage: noImage, Embedded: embedded, InFlight: inFlight}
}

func TestEmbeddingService_SubmitCatalog_SubmitsThePendingCardsAsOneBatch(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()
	first, second := hostedCard("cards/one"), hostedCard("cards/two")

	f.repo.EXPECT().CountEmbeddingStates(ctx, testEmbeddingModel, "catalog", "MA6").Return(stateCounts(4, 1, 1, 0), nil)
	f.repo.EXPECT().ListPendingCards(ctx, testEmbeddingModel, "catalog", "MA6").Return([]entity.Card{first, second}, nil)
	f.images.EXPECT().Fetch(ctx, "https://img.test/cards/one").Return("image/png", []byte("one"), nil)
	f.images.EXPECT().Fetch(ctx, "https://img.test/cards/two").Return("image/png", []byte("two"), nil)
	f.embedder.EXPECT().Submit(ctx, []embedding.Image{
		{Key: first.ID.String(), MIMEType: "image/png", Data: []byte("one")},
		{Key: second.ID.String(), MIMEType: "image/png", Data: []byte("two")},
	}).Return("batches/job-1", nil)
	f.repo.EXPECT().CreateBatch(ctx, mock.MatchedBy(func(b *entity.EmbeddingBatch) bool {
		return b.JobName == "batches/job-1" && b.Model == testEmbeddingModel && b.Source == "catalog" && b.State == entity.EmbeddingBatchSubmitted
	}), []uuid.UUID{first.ID, second.ID}).Return(nil)

	summary, err := f.service.SubmitCatalog(ctx, dto.SubmitCatalogRequest{Set: "MA6"})
	require.NoError(t, err)
	assert.Equal(t, dto.SubmitCatalogSummary{Total: 4, NoImage: 1, AlreadyEmbedded: 1, Submitted: 2, Batches: 1}, withoutElapsed(summary))
}

func TestEmbeddingService_SubmitCatalog_SplitsTheCardsIntoBatchesOfTheBatchSize(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()
	cards := make([]entity.Card, embeddingBatchSize+1)
	for i := range cards {
		cards[i] = hostedCard("cards/" + uuid.NewString())
	}

	f.repo.EXPECT().CountEmbeddingStates(ctx, testEmbeddingModel, "catalog", "").Return(stateCounts(int64(len(cards)), 0, 0, 0), nil)
	f.repo.EXPECT().ListPendingCards(ctx, testEmbeddingModel, "catalog", "").Return(cards, nil)
	f.images.EXPECT().Fetch(mock.Anything, mock.Anything).Return("image/png", []byte("img"), nil).Times(len(cards))
	f.embedder.EXPECT().Submit(ctx, mock.MatchedBy(func(images []embedding.Image) bool { return len(images) == embeddingBatchSize })).Return("batches/full", nil)
	f.embedder.EXPECT().Submit(ctx, mock.MatchedBy(func(images []embedding.Image) bool { return len(images) == 1 })).Return("batches/rest", nil)
	f.repo.EXPECT().CreateBatch(ctx, mock.Anything, mock.Anything).Return(nil).Times(2)

	summary, err := f.service.SubmitCatalog(ctx, dto.SubmitCatalogRequest{})
	require.NoError(t, err)
	assert.Equal(t, len(cards), summary.Submitted)
	assert.Equal(t, 2, summary.Batches)
}

func TestEmbeddingService_SubmitCatalog_LeavesCardsInAnOutstandingBatchOut(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()

	f.repo.EXPECT().CountEmbeddingStates(ctx, testEmbeddingModel, "catalog", "").Return(stateCounts(3, 0, 0, 2), nil)
	f.repo.EXPECT().ListPendingCards(ctx, testEmbeddingModel, "catalog", "").Return(nil, nil)

	summary, err := f.service.SubmitCatalog(ctx, dto.SubmitCatalogRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), summary.InFlight, "the cards in an outstanding batch are reported as skipped")
	assert.Zero(t, summary.Submitted)
}

func TestEmbeddingService_SubmitCatalog_CountsACardWhoseImageCannotBeFetched(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()
	broken, fine := hostedCard("cards/broken"), hostedCard("cards/fine")

	f.repo.EXPECT().CountEmbeddingStates(ctx, testEmbeddingModel, "catalog", "").Return(stateCounts(2, 0, 0, 0), nil)
	f.repo.EXPECT().ListPendingCards(ctx, testEmbeddingModel, "catalog", "").Return([]entity.Card{broken, fine}, nil)
	f.images.EXPECT().Fetch(ctx, "https://img.test/cards/broken").Return("", nil, errors.New("HTTP 404"))
	f.images.EXPECT().Fetch(ctx, "https://img.test/cards/fine").Return("image/png", []byte("ok"), nil)
	f.embedder.EXPECT().Submit(ctx, []embedding.Image{{Key: fine.ID.String(), MIMEType: "image/png", Data: []byte("ok")}}).Return("batches/job-2", nil)
	f.repo.EXPECT().CreateBatch(ctx, mock.Anything, []uuid.UUID{fine.ID}).Return(nil)

	summary, err := f.service.SubmitCatalog(ctx, dto.SubmitCatalogRequest{})
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Failed)
	assert.Equal(t, 1, summary.Submitted)
}

func TestEmbeddingService_SubmitCatalog_CountsAFailedSubmitAndLeavesItsCardsPending(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()
	card := hostedCard("cards/refused")

	f.repo.EXPECT().CountEmbeddingStates(ctx, testEmbeddingModel, "catalog", "").Return(stateCounts(1, 0, 0, 0), nil)
	f.repo.EXPECT().ListPendingCards(ctx, testEmbeddingModel, "catalog", "").Return([]entity.Card{card}, nil)
	f.images.EXPECT().Fetch(ctx, mock.Anything).Return("image/png", []byte("img"), nil)
	f.embedder.EXPECT().Submit(ctx, mock.Anything).Return("", errors.New("quota"))

	summary, err := f.service.SubmitCatalog(ctx, dto.SubmitCatalogRequest{})
	require.NoError(t, err, "a refused job does not stop the run")
	assert.Equal(t, 1, summary.Failed)
	assert.Zero(t, summary.Batches, "no batch is recorded, so the card stays pending")
}

func TestEmbeddingService_SubmitCatalog_StopsWhenAJobCannotBeRecorded(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()
	card := hostedCard("cards/unrecorded")
	connLost := errors.New("connection lost")

	f.repo.EXPECT().CountEmbeddingStates(ctx, testEmbeddingModel, "catalog", "").Return(stateCounts(1, 0, 0, 0), nil)
	f.repo.EXPECT().ListPendingCards(ctx, testEmbeddingModel, "catalog", "").Return([]entity.Card{card}, nil)
	f.images.EXPECT().Fetch(ctx, mock.Anything).Return("image/png", []byte("img"), nil)
	f.embedder.EXPECT().Submit(ctx, mock.Anything).Return("batches/orphan", nil)
	f.repo.EXPECT().CreateBatch(ctx, mock.Anything, mock.Anything).Return(connLost)

	_, err := f.service.SubmitCatalog(ctx, dto.SubmitCatalogRequest{})
	assert.Same(t, connLost, err, "the repository error comes back unchanged; the orphaned job is logged")
}

func TestEmbeddingService_CollectBatches_StoresSucceededVectorsAndMarksTheBatchCollected(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()
	card := uuid.New()
	batch := submittedBatch("batches/job-1")
	vector := unitVector(0)

	f.repo.EXPECT().ListSubmittedBatches(ctx).Return([]entity.EmbeddingBatch{batch}, nil)
	f.embedder.EXPECT().Collect(ctx, "batches/job-1").Return(embedding.Outcome{
		Status:  embedding.JobSucceeded,
		Vectors: map[string][]float32{card.String(): vector},
	}, nil)
	f.repo.EXPECT().ListBatchCardIDs(ctx, batch.ID).Return([]uuid.UUID{card}, nil)
	f.repo.EXPECT().Upsert(ctx, []entity.CardEmbedding{{
		CardID:    card,
		Model:     testEmbeddingModel,
		Source:    "catalog",
		Embedding: pgvector.NewVector(vector),
	}}).Return(nil)
	f.repo.EXPECT().SetBatchState(ctx, batch.ID, entity.EmbeddingBatchCollected).Return(nil)

	summary, err := f.service.CollectBatches(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Collected)
	assert.Equal(t, 1, summary.Embedded)
	assert.Zero(t, summary.Failed)
}

func TestEmbeddingService_CollectBatches_LeavesARunningBatchSubmitted(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()
	batch := submittedBatch("batches/job-running")

	f.repo.EXPECT().ListSubmittedBatches(ctx).Return([]entity.EmbeddingBatch{batch}, nil)
	f.embedder.EXPECT().Collect(ctx, "batches/job-running").Return(embedding.Outcome{Status: embedding.JobRunning}, nil)

	summary, err := f.service.CollectBatches(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Running)
	assert.Zero(t, summary.Collected)
}

func TestEmbeddingService_CollectBatches_MarksAFailedBatchFailedSoItsCardsAreAgainPending(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()
	batch := submittedBatch("batches/job-cancelled")

	f.repo.EXPECT().ListSubmittedBatches(ctx).Return([]entity.EmbeddingBatch{batch}, nil)
	f.embedder.EXPECT().Collect(ctx, "batches/job-cancelled").Return(embedding.Outcome{Status: embedding.JobFailed, Reason: "JOB_STATE_CANCELLED"}, nil)
	f.repo.EXPECT().SetBatchState(ctx, batch.ID, entity.EmbeddingBatchFailed).Return(nil)

	summary, err := f.service.CollectBatches(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.FailedBatches)
}

func TestEmbeddingService_CollectBatches_StoresTheRestOfABatchWithPerItemFailures(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()
	good, bad := uuid.New(), uuid.New()
	batch := submittedBatch("batches/job-partial")

	f.repo.EXPECT().ListSubmittedBatches(ctx).Return([]entity.EmbeddingBatch{batch}, nil)
	f.embedder.EXPECT().Collect(ctx, "batches/job-partial").Return(embedding.Outcome{
		Status:   embedding.JobSucceeded,
		Vectors:  map[string][]float32{good.String(): unitVector(1)},
		Failures: map[string]string{bad.String(): "image unreadable"},
	}, nil)
	f.repo.EXPECT().ListBatchCardIDs(ctx, batch.ID).Return([]uuid.UUID{good, bad}, nil)
	f.repo.EXPECT().Upsert(ctx, mock.MatchedBy(func(rows []entity.CardEmbedding) bool {
		return len(rows) == 1 && rows[0].CardID == good
	})).Return(nil)
	f.repo.EXPECT().SetBatchState(ctx, batch.ID, entity.EmbeddingBatchCollected).Return(nil)

	summary, err := f.service.CollectBatches(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Embedded)
	assert.Equal(t, 1, summary.Failed, "the failed card is counted and left pending, not stored")
}

func TestEmbeddingService_CollectBatches_KeepsOnlyTheCardsOfTheBatch(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()
	card, deleted := uuid.New(), uuid.New()
	batch := submittedBatch("batches/job-foreign-key")

	f.repo.EXPECT().ListSubmittedBatches(ctx).Return([]entity.EmbeddingBatch{batch}, nil)
	f.embedder.EXPECT().Collect(ctx, "batches/job-foreign-key").Return(embedding.Outcome{
		Status: embedding.JobSucceeded,
		Vectors: map[string][]float32{
			card.String():    unitVector(0),
			deleted.String(): unitVector(0),
		},
	}, nil)
	f.repo.EXPECT().ListBatchCardIDs(ctx, batch.ID).Return([]uuid.UUID{card}, nil)
	f.repo.EXPECT().Upsert(ctx, mock.MatchedBy(func(rows []entity.CardEmbedding) bool {
		return len(rows) == 1 && rows[0].CardID == card
	})).Return(nil)
	f.repo.EXPECT().SetBatchState(ctx, batch.ID, entity.EmbeddingBatchCollected).Return(nil)

	summary, err := f.service.CollectBatches(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Embedded)
	assert.Equal(t, 1, summary.Failed, "a card deleted since the submit is counted, and its row is not written")
}

func TestEmbeddingService_CollectBatches_CountsACardWithNoLineInTheResultAsPending(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()
	answered, silent := uuid.New(), uuid.New()
	batch := submittedBatch("batches/job-missing-line")

	f.repo.EXPECT().ListSubmittedBatches(ctx).Return([]entity.EmbeddingBatch{batch}, nil)
	f.embedder.EXPECT().Collect(ctx, "batches/job-missing-line").Return(embedding.Outcome{
		Status:  embedding.JobSucceeded,
		Vectors: map[string][]float32{answered.String(): unitVector(0)},
	}, nil)
	f.repo.EXPECT().ListBatchCardIDs(ctx, batch.ID).Return([]uuid.UUID{answered, silent}, nil)
	f.repo.EXPECT().Upsert(ctx, mock.MatchedBy(func(rows []entity.CardEmbedding) bool {
		return len(rows) == 1 && rows[0].CardID == answered
	})).Return(nil)
	f.repo.EXPECT().SetBatchState(ctx, batch.ID, entity.EmbeddingBatchCollected).Return(nil)

	summary, err := f.service.CollectBatches(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Failed, "a card the result never mentions is counted, and stays pending")
	assert.Equal(t, 1, summary.Embedded)
}

func TestEmbeddingService_CollectBatches_LeavesABatchItCannotCheckAndGoesOn(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()
	unreachable, finished := submittedBatch("batches/unreachable"), submittedBatch("batches/finished")

	f.repo.EXPECT().ListSubmittedBatches(ctx).Return([]entity.EmbeddingBatch{unreachable, finished}, nil)
	f.embedder.EXPECT().Collect(ctx, "batches/unreachable").Return(embedding.Outcome{}, errors.New("HTTP 503"))
	f.embedder.EXPECT().Collect(ctx, "batches/finished").Return(embedding.Outcome{Status: embedding.JobRunning}, nil)

	summary, err := f.service.CollectBatches(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Failed)
	assert.Equal(t, 1, summary.Running)
}

func TestEmbeddingService_CollectBatches_StopsWhenTheBatchCannotBeMarkedCollected(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()
	batch := submittedBatch("batches/job-retry")
	vector := unitVector(0)
	card := uuid.New()

	f.repo.EXPECT().ListSubmittedBatches(ctx).Return([]entity.EmbeddingBatch{batch}, nil)
	f.embedder.EXPECT().Collect(ctx, "batches/job-retry").Return(embedding.Outcome{
		Status:  embedding.JobSucceeded,
		Vectors: map[string][]float32{card.String(): vector},
	}, nil)
	f.repo.EXPECT().ListBatchCardIDs(ctx, batch.ID).Return([]uuid.UUID{card}, nil)
	f.repo.EXPECT().Upsert(ctx, mock.Anything).Return(nil)
	f.repo.EXPECT().SetBatchState(ctx, batch.ID, entity.EmbeddingBatchCollected).Return(errors.New("connection lost"))

	_, err := f.service.CollectBatches(ctx)
	require.Error(t, err, "the batch stays submitted, so the next run repeats the upsert and marks it")
}

func TestEmbeddingService_CollectBatches_DoesNothingWhenNoBatchIsSubmitted(t *testing.T) {
	f := newEmbeddingFixture(t)
	ctx := context.Background()

	f.repo.EXPECT().ListSubmittedBatches(ctx).Return(nil, nil)

	summary, err := f.service.CollectBatches(ctx)
	require.NoError(t, err)
	assert.Zero(t, summary.Batches)
}

// submittedBatch is a batch the repository reports as awaiting collection.
func submittedBatch(job string) entity.EmbeddingBatch {
	return entity.EmbeddingBatch{
		BaseEntity: crud.BaseEntity{ID: uuid.New()},
		JobName:    job,
		Model:      testEmbeddingModel,
		Source:     "catalog",
		State:      entity.EmbeddingBatchSubmitted,
	}
}

// unitVector is a vector of the configured dimension with 1 on one axis.
func unitVector(axis int) []float32 {
	v := make([]float32, embedding.Dimensions)
	v[axis] = 1
	return v
}

// withoutElapsed clears the timing, which differs from run to run.
func withoutElapsed(s dto.SubmitCatalogSummary) dto.SubmitCatalogSummary {
	s.Elapsed = 0
	return s
}
