package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ungerr"
	"github.com/pgvector/pgvector-go"
)

// EmbeddingService embeds the hosted image of every card for one embedding
// model, and is safe to rerun: cards already embedded by that model are
// skipped, and a card that fails is logged and left for the next run.
type EmbeddingService interface {
	EmbedCatalog(ctx context.Context, req dto.EmbedCatalogRequest) (dto.EmbedCatalogSummary, error)
}

type embeddingService struct {
	repo      repository.EmbeddingRepository
	embedder  embedding.Embedder
	images    embedding.ImageFetcher
	imageHost mapper.ImageHost
	model     string
}

// NewEmbeddingService builds the batch service for model. Changing model
// makes every card pending again, because a card's stored embedding only
// counts for the model that made it.
func NewEmbeddingService(
	repo repository.EmbeddingRepository,
	embedder embedding.Embedder,
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

func (s *embeddingService) EmbedCatalog(ctx context.Context, req dto.EmbedCatalogRequest) (dto.EmbedCatalogSummary, error) {
	started := time.Now()

	states, err := s.repo.CountEmbeddingStates(ctx, s.model, req.Set)
	if err != nil {
		return dto.EmbedCatalogSummary{}, err
	}
	pending, err := s.repo.ListPendingCards(ctx, s.model, req.Set)
	if err != nil {
		return dto.EmbedCatalogSummary{}, err
	}

	summary := dto.EmbedCatalogSummary{
		Total:           states.Total,
		NoImage:         states.NoImage,
		AlreadyEmbedded: states.Embedded,
	}
	logger.Infof(
		"embedding %d pending card(s) with %s, skipping %d with no hosted image and %d already embedded",
		len(pending), s.model, states.NoImage, states.Embedded,
	)

	for i, card := range pending {
		if err := ctx.Err(); err != nil {
			return summary, ungerr.Wrap(err, "embedding interrupted")
		}

		position := fmt.Sprintf("[%d/%d]", i+1, len(pending))
		err := s.embedCard(ctx, card)
		switch {
		case errors.Is(err, embedding.ErrDailyQuotaExhausted):
			summary.Stopped = true
			summary.Elapsed = time.Since(started)
			logger.Warnf("%s stopped: the daily embedding quota is spent after %d embedded; rerun to resume, embedded cards are skipped", position, summary.Embedded)
			return summary, nil
		case err != nil:
			summary.Failed++
			logger.Errorf("%s card %s failed: %v", position, card.ID, err)
		default:
			summary.Embedded++
			logger.Infof("%s card %s embedded", position, card.ID)
		}
	}

	summary.Elapsed = time.Since(started)
	logger.Infof(
		"done in %s: %d embedded, %d skipped (%d no hosted image, %d already embedded), %d failed",
		summary.Elapsed, summary.Embedded, summary.NoImage+summary.AlreadyEmbedded, summary.NoImage, summary.AlreadyEmbedded, summary.Failed,
	)
	return summary, nil
}

// embedCard fetches, embeds and stores one card. Its errors are the callee's
// own wrapped errors, returned unchanged so the quota sentinel stays matchable.
func (s *embeddingService) embedCard(ctx context.Context, card entity.Card) error {
	img, err := s.images.Fetch(ctx, s.imageHost.URL(card.ImageKey))
	if err != nil {
		return err
	}
	vector, err := s.embedder.Embed(ctx, img)
	if err != nil {
		return err
	}
	return s.repo.Upsert(ctx, entity.CardEmbedding{
		CardID:    card.ID,
		Model:     s.model,
		Embedding: pgvector.NewVector(vector),
	})
}
