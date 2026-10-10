// Command embed-catalog submits the hosted image of every card that has one
// and no embedding yet from EMBEDDING_MODEL (default gemini-embedding-2) to
// Gemini's batch embedding API. It only submits: a job can take up to 24 hours,
// so `make collect-embeddings` stores the vectors once the jobs finish. Cards
// already embedded by the model, and cards in an outstanding batch, are
// skipped, so rerunning is safe. Manually triggered only, never on Railway.
package main

import (
	"context"
	"flag"
	"os"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	crud "github.com/itsLeonB/go-crud"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	set := flag.String("set", "", "limit submitting to one Expansion Set by its site code (e.g. \"MA6\"); default submits every card with a hosted image")
	flag.Parse()

	logger.Init("EmbedCatalog")

	if err := config.Load(); err != nil {
		logger.Fatal(err)
	}
	if config.Global.APIKey == "" {
		logger.Fatal("GEMINI_API_KEY is not set")
	}
	if config.Global.BaseURL == "" {
		logger.Fatal("IMAGE_BASE_URL is not set: card images are fetched from it")
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

	summary, err := job.SubmitCatalog(ctx, dto.SubmitCatalogRequest{Set: *set})
	if err != nil {
		logger.Error(err)
		cleanup()
		os.Exit(1)
	}

	if summary.Failed > 0 {
		logger.Warnf("%d card(s) failed to submit - rerun to submit them again; submitted and skipped cards are left alone", summary.Failed)
		// Non-zero exit tells an operator a rerun is needed, as the ingester does.
		cleanup()
		os.Exit(1)
	}
}
