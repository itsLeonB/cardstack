// Command ingest-tcgcollector compares TCG Collector's Indonesian sets list
// with the catalog and prints what is missing. It is a dry run: it writes
// nothing and prints its report to stdout, logs go to stderr. Manually
// triggered only; run it with -dir pointing at a saved copy of the sets page
// (`make ingest-tcgcollector ARGS="-dir ./saved"`), since the browser fetcher
// is not built yet.
package main

import (
	"context"
	"flag"
	"os"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/ingestion/tcgcollector"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	dir := flag.String("dir", "", "directory of saved TCG Collector pages; the sets page is sets-id.html in it")
	set := flag.String("set", "", "limit the report to one set by its code (e.g. \"SMP\" or \"SM-P\")")
	apply := flag.Bool("apply", false, "create and update rows; not implemented yet, so the command only reports")
	refresh := flag.Bool("refresh", false, "refetch cached pages; accepted but has no effect until the page cache lands")
	flag.Parse()

	logger.Init("IngestTCGCollector")

	if *apply {
		logger.Fatal("-apply is not implemented yet: this command only reports")
	}
	if *refresh {
		logger.Warn("-refresh has no effect yet: there is no page cache")
	}
	if *dir == "" {
		logger.Fatal("-dir is required: the browser fetcher is not built yet")
	}

	if err := config.Load(); err != nil {
		logger.Fatal(err)
	}

	providers, cleanup, err := provider.InitializeProviders()
	if err != nil {
		logger.Fatal(err)
	}
	defer cleanup()

	ingester := tcgcollector.NewIngester(providers.Gorm, tcgcollector.NewDirSource(*dir))
	report, err := ingester.Run(context.Background(), tcgcollector.Options{SetCode: *set})
	if err != nil {
		logger.Fatal(err)
	}

	if err := report.Render(os.Stdout); err != nil {
		logger.Fatal(err)
	}
}
