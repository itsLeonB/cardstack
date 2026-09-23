// Command ingest-pokemon-asia scrapes Pokémon TCG catalog data from
// https://asia.pokemon-card.com/id (plain server-rendered HTML, no public
// API) into the catalog schema (games/series/expansion_sets/rarities/
// cards), covering all four in-scope Series. Manually triggered only — no
// scheduler or cron job invokes it; run it directly
// (`make ingest-pokemon-asia`) whenever a fresh pull is needed.
package main

import (
	"context"
	"flag"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/ingestion/pokemonasia"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/core/otel"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	series := flag.String("series", "", "debug: limit ingestion to one Series (site's own name, e.g. \"Evolusi Mega\"); default ingests all four")
	flag.Parse()

	logger.Init("IngestPokemonAsia")

	if err := config.Load(); err != nil {
		logger.Fatal(err)
	}

	ctx := context.Background()
	otelShutdown, err := otel.InitSDK(ctx, config.Global.OTel)
	if err != nil {
		logger.Fatalf("failed to initialize OTel SDK: %v", err)
	}
	defer func() {
		if err := otelShutdown(ctx); err != nil {
			logger.Errorf("error shutting down OTel SDK: %v", err)
		}
	}()

	providers, cleanup, err := provider.InitializeProviders()
	if err != nil {
		logger.Fatal(err)
	}
	defer cleanup()

	ingester := pokemonasia.NewIngester(providers.Gorm)

	summary, err := ingester.Run(ctx, *series)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Infof(
		"ingestion complete: %d series, %d sets, %d rarities, %d cards",
		summary.Series, summary.Sets, summary.Rarities, summary.Cards,
	)
}
