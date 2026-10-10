// Command collect-embeddings stores the vectors of the batch jobs that
// `make embed-catalog` submitted, once each job has finished. A job still
// running is left for the next run. A job that failed, was cancelled or expired
// is marked failed, so its cards are submitted again by the next embed-catalog.
// A collected batch is never read again. Manually triggered only, never on
// Railway.
package main

import (
	"context"
	"os"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	crud "github.com/itsLeonB/go-crud"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	logger.Init("CollectEmbeddings")

	if err := config.Load(); err != nil {
		logger.Fatal(err)
	}
	if config.Global.APIKey == "" {
		logger.Fatal("GEMINI_API_KEY is not set")
	}

	providers, cleanup, err := provider.InitializeProviders()
	if err != nil {
		logger.Fatal(err)
	}
	defer cleanup()

	ctx := context.Background()
	embedder, err := embedding.NewGeminiBatchEmbedder(ctx, config.Global.APIKey, config.Global.Model)
	if err != nil {
		logger.Fatal(err)
	}
	job := service.NewEmbeddingService(
		repository.NewEmbeddingRepository(crud.NewRepository[entity.CardEmbedding](providers.Gorm)),
		embedder,
		embedding.NewHTTPImageFetcher(),
		mapper.NewImageHost(config.Global.BaseURL),
		config.Global.Model,
	)

	summary, err := job.CollectBatches(ctx)
	if err != nil {
		logger.Error(err)
		cleanup()
		os.Exit(1)
	}

	if summary.Failed > 0 || summary.FailedBatches > 0 {
		logger.Warnf("%d batch(es) failed at the provider and %d card(s) did not embed - run embed-catalog to submit the cards again", summary.FailedBatches, summary.Failed)
		// Non-zero exit tells an operator a rerun is needed, as the ingester does.
		cleanup()
		os.Exit(1)
	}
}
