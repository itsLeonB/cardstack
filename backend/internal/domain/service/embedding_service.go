package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ungerr"
	"github.com/pgvector/pgvector-go"
)

// embeddingBatchSize is how many cards one provider batch job embeds. The job
// reads a file, so the limit is the upload's, not the 20 MB inline one; the
// images of a batch are held in memory while it is built.
const embeddingBatchSize = 500

// catalogSource names the image that catalog embeddings come from.
const catalogSource = "catalog"

// EmbeddingService submits the catalog's card images for batch embedding and
// collects the batches that have finished.
type EmbeddingService interface {
	SubmitCatalog(ctx context.Context, req dto.SubmitCatalogRequest) (dto.SubmitCatalogSummary, error)
	CollectBatches(ctx context.Context) (dto.CollectBatchesSummary, error)
}

type embeddingService struct {
	repo      repository.EmbeddingRepository
	embedder  embedding.BatchEmbedder
	images    embedding.ImageFetcher
	imageHost mapper.ImageHost
	model     string
}

func NewEmbeddingService(
	repo repository.EmbeddingRepository,
	embedder embedding.BatchEmbedder,
	images embedding.ImageFetcher,
	imageHost mapper.ImageHost,
	model string,
) EmbeddingService {
	return &embeddingService{
		repo:      repo,
		embedder:  embedder,
		images:    images,
		imageHost: imageHost,
		model:     model,
	}
}

func (s *embeddingService) SubmitCatalog(ctx context.Context, req dto.SubmitCatalogRequest) (dto.SubmitCatalogSummary, error) {
	started := time.Now()

	states, err := s.repo.CountEmbeddingStates(ctx, s.model, catalogSource, req.Set)
	if err != nil {
		return dto.SubmitCatalogSummary{}, err
	}
	pending, err := s.repo.ListPendingCards(ctx, s.model, catalogSource, req.Set)
	if err != nil {
		return dto.SubmitCatalogSummary{}, err
	}

	summary := dto.SubmitCatalogSummary{
		Total:           states.Total,
		NoImage:         states.NoImage,
		AlreadyEmbedded: states.Embedded,
		InFlight:        states.InFlight,
	}
	batches := (len(pending) + embeddingBatchSize - 1) / embeddingBatchSize
	logger.Infof(
		"submitting %d pending card(s) with %s in %d batch(es) of up to %d, skipping %d with no hosted image, %d already embedded and %d in an outstanding batch",
		len(pending), s.model, batches, embeddingBatchSize, states.NoImage, states.Embedded, states.InFlight,
	)

	for start := 0; start < len(pending); start += embeddingBatchSize {
		end := min(start+embeddingBatchSize, len(pending))
		batch := batchProgress{
			first:   start + 1,
			total:   len(pending),
			number:  start/embeddingBatchSize + 1,
			batches: batches,
		}
		if err := s.submitBatch(ctx, pending[start:end], batch, &summary); err != nil {
			return s.finishSubmit(summary, started), err
		}
	}
	return s.finishSubmit(summary, started), nil
}

// batchProgress places one batch in its run for the log lines: the position of
// its first card out of total, and its number out of batches.
type batchProgress struct {
	first   int
	total   int
	number  int
	batches int
}

// submitBatch prepares the images of cards and submits them as one job, then
// records the job. A card whose image cannot be prepared, and a job the
// provider refuses, count as failed and leave their cards pending. A job the
// provider accepted but the database did not record is an error that stops the
// run: its cards will be submitted again, which is rework rather than a duplicate.
func (s *embeddingService) submitBatch(ctx context.Context, cards []entity.Card, batch batchProgress, summary *dto.SubmitCatalogSummary) error {
	images := make([]embedding.Image, 0, len(cards))
	cardIDs := make([]uuid.UUID, 0, len(cards))
	for i, card := range cards {
		if err := ctx.Err(); err != nil {
			return ungerr.Wrap(err, "submitting interrupted")
		}
		position := fmt.Sprintf("[%d/%d]", batch.first+i, batch.total)
		mimeType, data, err := s.images.Fetch(ctx, s.imageHost.URL(card.ImageKey))
		if err != nil {
			summary.Failed++
			logger.Errorf("%s card %s failed: %v", position, card.ID, err)
			continue
		}
		images = append(images, embedding.Image{Key: card.ID.String(), MIMEType: mimeType, Data: data})
		cardIDs = append(cardIDs, card.ID)
		logger.Infof("%s card %s prepared", position, card.ID)
	}
	if len(images) == 0 {
		return nil
	}

	label := fmt.Sprintf("[%d/%d]", batch.number, batch.batches)
	job, err := s.embedder.Submit(ctx, images)
	if err != nil {
		summary.Failed += len(images)
		logger.Errorf("batch %s of %d card(s) failed to submit: %v", label, len(images), err)
		return nil
	}

	record := &entity.EmbeddingBatch{JobName: job, Model: s.model, Source: catalogSource, State: entity.EmbeddingBatchSubmitted}
	if err := s.repo.CreateBatch(ctx, record, cardIDs); err != nil {
		return ungerr.Wrapf(err, "recording batch %s as job %s: its cards will be submitted again", label, job)
	}
	summary.Batches++
	summary.Submitted += len(images)
	logger.Infof("batch %s submitted as %s with %d card(s)", label, job, len(images))
	return nil
}

func (s *embeddingService) finishSubmit(summary dto.SubmitCatalogSummary, started time.Time) dto.SubmitCatalogSummary {
	summary.Elapsed = time.Since(started)
	logger.Infof(
		"submitted in %s: %d card(s) in %d batch(es), %d failed; skipped %d (%d no hosted image, %d already embedded, %d in an outstanding batch)",
		summary.Elapsed, summary.Submitted, summary.Batches, summary.Failed,
		summary.NoImage+summary.AlreadyEmbedded+summary.InFlight, summary.NoImage, summary.AlreadyEmbedded, summary.InFlight,
	)
	return summary
}

func (s *embeddingService) CollectBatches(ctx context.Context) (dto.CollectBatchesSummary, error) {
	started := time.Now()

	batches, err := s.repo.ListSubmittedBatches(ctx)
	if err != nil {
		return dto.CollectBatchesSummary{}, err
	}
	summary := dto.CollectBatchesSummary{Batches: len(batches)}
	logger.Infof("collecting %d submitted batch(es)", len(batches))

	for i, batch := range batches {
		if err := ctx.Err(); err != nil {
			return s.finishCollect(summary, started), ungerr.Wrap(err, "collecting interrupted")
		}
		position := fmt.Sprintf("[%d/%d]", i+1, len(batches))
		if err := s.collectBatch(ctx, batch, position, &summary); err != nil {
			return s.finishCollect(summary, started), err
		}
	}
	return s.finishCollect(summary, started), nil
}

// collectBatch asks the provider for one batch. A job that has finished is
// stored (succeeded) or marked failed, and one still running is left submitted
// for a later run. A check that errors is also left submitted, and counts as
// failed so the run exits non-zero. Only database errors stop the run.
func (s *embeddingService) collectBatch(ctx context.Context, batch entity.EmbeddingBatch, position string, summary *dto.CollectBatchesSummary) error {
	outcome, err := s.embedder.Collect(ctx, batch.JobName)
	if err != nil {
		summary.Failed++
		logger.Errorf("%s batch %s could not be checked, left submitted: %v", position, batch.JobName, err)
		return nil
	}

	switch outcome.Status {
	case embedding.JobRunning:
		summary.Running++
		logger.Infof("%s batch %s still running", position, batch.JobName)
		return nil
	case embedding.JobFailed:
		summary.FailedBatches++
		logger.Warnf("%s batch %s failed (%s): its cards are pending again", position, batch.JobName, outcome.Reason)
		return s.repo.SetBatchState(ctx, batch.ID, entity.EmbeddingBatchFailed)
	default:
		return s.storeBatch(ctx, batch, outcome, position, summary)
	}
}

// storeBatch upserts the vectors of a succeeded job, then marks the batch
// collected. A crash between the two repeats the upsert on the next run, which
// replaces the same rows and so creates no duplicates.
func (s *embeddingService) storeBatch(ctx context.Context, batch entity.EmbeddingBatch, outcome embedding.Outcome, position string, summary *dto.CollectBatchesSummary) error {
	embeddings := make([]entity.CardEmbedding, 0, len(outcome.Vectors))
	for key, vector := range outcome.Vectors {
		cardID, err := uuid.Parse(key)
		if err != nil {
			summary.Failed++
			logger.Errorf("%s batch %s has a result for %q, which is not a card ID", position, batch.JobName, key)
			continue
		}
		embeddings = append(embeddings, entity.CardEmbedding{
			CardID:    cardID,
			Model:     batch.Model,
			Source:    batch.Source,
			Embedding: pgvector.NewVector(vector),
		})
	}
	for key, reason := range outcome.Failures {
		summary.Failed++
		logger.Errorf("%s batch %s card %s did not embed, it is pending again: %s", position, batch.JobName, key, reason)
	}

	if err := s.repo.Upsert(ctx, embeddings); err != nil {
		return err
	}
	if err := s.repo.SetBatchState(ctx, batch.ID, entity.EmbeddingBatchCollected); err != nil {
		return err
	}
	summary.Collected++
	summary.Embedded += len(embeddings)
	logger.Infof("%s batch %s collected: %d embedded, %d did not", position, batch.JobName, len(embeddings), len(outcome.Failures))
	return nil
}

func (s *embeddingService) finishCollect(summary dto.CollectBatchesSummary, started time.Time) dto.CollectBatchesSummary {
	summary.Elapsed = time.Since(started)
	logger.Infof(
		"collected in %s: %d of %d batch(es) collected, %d still running, %d failed at the provider, %d embedded, %d failed",
		summary.Elapsed, summary.Collected, summary.Batches, summary.Running, summary.FailedBatches, summary.Embedded, summary.Failed,
	)
	return summary
}
