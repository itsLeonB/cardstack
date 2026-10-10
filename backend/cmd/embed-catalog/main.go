// Command embed-catalog embeds the hosted image of every card that has one
// with Gemini (EMBEDDING_MODEL, default gemini-embedding-2) and stores the
// vectors in card_embeddings. Manually triggered only, never on Railway.
// Rerunning is safe: cards already embedded by the configured model are
// skipped, failed cards are retried by a rerun, and a spent daily quota stops
// the run cleanly so the next run resumes it.
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
	set := flag.String("set", "", "limit embedding to one Expansion Set by its site code (e.g. \"MA6\"); default embeds every card with a hosted image")
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

	job := service.NewEmbeddingService(
		repository.NewEmbeddingRepository(crud.NewRepository[entity.CardEmbedding](providers.Gorm)),
		embedding.NewGeminiEmbedder(config.Global.APIKey, config.Global.Model),
		embedding.NewHTTPImageFetcher(),
		mapper.NewImageHost(config.Global.BaseURL),
		config.Global.Model,
	)

	summary, err := job.EmbedCatalog(context.Background(), dto.EmbedCatalogRequest{Set: *set})
	if err != nil {
		logger.Error(err)
		cleanup()
		os.Exit(1)
	}

	if summary.Failed > 0 {
		logger.Warnf("%d card(s) failed - rerun to retry them; embedded cards are skipped", summary.Failed)
		// Non-zero exit tells an operator a rerun is needed, as the ingester does.
		cleanup()
		os.Exit(1)
	}
}
