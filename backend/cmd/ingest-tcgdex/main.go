// Command ingest-tcgdex ingests Pokémon TCG (SV) catalog data from the
// public TCGDex API (https://api.tcgdex.net/v2) into the catalog schema
// (games/expansion_sets/cards/card_variants). Manually triggered only — no
// scheduler or cron job invokes it; run it directly (`make ingest-tcgdex`)
// whenever a fresh pull is needed.
package main

import (
	"context"
	"flag"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/ingestion/tcgdex"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/core/otel"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	locale := flag.String("locale", "id", "TCGDex locale to ingest")
	series := flag.String("series", "sv", "TCGDex series to ingest")
	flag.Parse()

	logger.Init("IngestTCGDex")

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

	ingester := tcgdex.NewIngester(providers.Gorm)

	summary, err := ingester.Run(ctx, *locale, *series)
	if err != nil {
		logger.Fatal(err)
	}

	for _, mismatch := range summary.Mismatches {
		logger.Warnf("card count mismatch: %s", mismatch)
	}

	logger.Infof("ingestion complete: %d sets, %d cards, %d variants", summary.Sets, summary.Cards, summary.Variants)
}
