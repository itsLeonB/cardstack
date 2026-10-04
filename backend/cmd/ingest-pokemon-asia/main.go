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
	"os"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/ingestion/pokemonasia"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	corestore "github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/core/otel"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	series := flag.String("series", "", "debug: limit ingestion to one Series (site's own name, e.g. \"Evolusi Mega\"); default ingests all four")
	set := flag.String("set", "", "debug: limit ingestion to one Expansion Set by its site code (e.g. \"MA6\"), independent of -series; default ingests every set")
	cleanupStaleRarities := flag.Bool(
		"cleanup-stale-rarities",
		false,
		"one-time cleanup: delete rarities rows left over from the pre-fix Regulation-Mark-as-rarity bug (ticket 11) that have zero referencing cards; rows still referenced are reported, not deleted. Off by default - normal ingestion runs no deletion.",
	)
	syncExpansionSets := flag.Bool(
		"sync-expansion-sets",
		false,
		"lightweight opt-in: enumerate listings and upsert only Series/Expansion Sets (e.g. to backfill Expansion Set covers into R2), skipping every card/rarity crawl. Honors -series/-set. Ignored if -cleanup-stale-rarities is also passed.",
	)
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

	// A nil interface (not a nil *R2Store) is what tells the ingester that
	// image hosting is off.
	var store corestore.ObjectStore
	if config.Global.Configured() {
		store = objectstore.NewR2Store(config.Global.R2)
	}
	ingester := pokemonasia.NewIngester(providers.Gorm, store)

	if *cleanupStaleRarities {
		results, err := ingester.CleanupStaleRarities(ctx)
		if err != nil {
			logger.Fatal(err)
		}

		blocked := 0
		for _, r := range results {
			if r.Deleted {
				logger.Infof("deleted stale rarity %q", r.Code)
				continue
			}
			blocked++
			logger.Warnf("rarity %q still referenced by %d card(s) - not deleted, re-run ingestion first", r.Code, r.CardCount)
		}
		if blocked > 0 {
			// Non-zero exit tells an operator this cleanup pass didn't fully
			// complete, the same way Run's own Summary.Failures does below.
			os.Exit(1)
		}
		return
	}

	run := ingester.Run
	if *syncExpansionSets {
		run = ingester.SyncExpansionSets
	}
	summary, err := run(ctx, *series, *set)
	if err != nil {
		logger.Fatal(err)
	}

	if len(summary.Failures) > 0 {
		logger.Warnf(
			"%d item(s) failed after exhausting retries - retry these independently, no need to rerun the whole ingestion:",
			len(summary.Failures),
		)
		for _, f := range summary.Failures {
			logger.Warn(f.Error())
		}
		// Non-zero exit lets an operator/script notice via $? that some
		// items need a follow-up run, even though ingestion itself
		// completed and every other row was written successfully.
		os.Exit(1)
	}
}
