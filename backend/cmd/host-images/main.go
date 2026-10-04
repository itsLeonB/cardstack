// Command host-images copies the image of every Expansion Set and card that
// has a scraped source address but no hosted key into the R2 bucket
// (docs/adr/0016), straight from the database and without crawling the source
// site. It is the backfill for rows ingested before image hosting existed;
// `ingest-pokemon-asia` still hosts images for rows it ingests. Re-running
// only copies what is still missing. Manually triggered, never on Railway.
package main

import (
	"context"
	"flag"
	"os"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/ingestion/pokemonasia"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	set := flag.String("set", "", "limit hosting to one Expansion Set by its site code (e.g. \"MA6\"); default hosts every row that is missing its image")
	flag.Parse()

	logger.Init("HostImages")

	if err := config.Load(); err != nil {
		logger.Fatal(err)
	}
	if !config.Global.R2.Configured() {
		logger.Fatal("R2 is not configured: set R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY and R2_BUCKET")
	}

	providers, cleanup, err := provider.InitializeProviders()
	if err != nil {
		logger.Fatal(err)
	}
	defer cleanup()

	ingester := pokemonasia.NewIngester(providers.Gorm, objectstore.NewR2Store(config.Global.R2))
	summary, err := ingester.HostMissingImages(context.Background(), *set)
	if err != nil {
		logger.Error(err)
		cleanup()
		os.Exit(1)
	}

	if len(summary.Failures) > 0 {
		logger.Warnf("%d image(s) failed - re-run to retry just those:", len(summary.Failures))
		for _, f := range summary.Failures {
			logger.Warn(f.Error())
		}
		// Non-zero exit tells an operator a re-run is needed, as the ingester does.
		cleanup()
		os.Exit(1)
	}
}
