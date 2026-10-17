// Command ingest-tcgcollector compares TCG Collector's Indonesian sets list
// with the catalog and prints what is missing. It is a dry run: it writes
// nothing and prints its report to stdout, logs go to stderr. Manually
// triggered only. Without -dir it starts a visible Chrome itself and fetches
// through it, so solve TCG Collector's challenge in that window once; later
// runs reuse the profile. With -dir it reads pages saved by hand instead
// (`make ingest-tcgcollector ARGS="-dir ./saved"`).
package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"

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
	if err := config.Load(); err != nil {
		logger.Fatal(err)
	}

	providers, cleanup, err := provider.InitializeProviders()
	if err != nil {
		logger.Fatal(err)
	}
	defer cleanup()

	pages, closePages := newPageSource(*dir)
	defer closePages()

	ingester := tcgcollector.NewIngester(providers.Gorm, pages)
	report, err := ingester.Run(context.Background(), tcgcollector.Options{SetCode: *set})
	if err != nil {
		logger.Fatal(err)
	}

	if err := report.Render(os.Stdout); err != nil {
		logger.Fatal(err)
	}
}

// newPageSource reads saved pages from dir when it is set. Otherwise it opens
// the operator's visible Chrome, whose profile lives in the user cache dir.
func newPageSource(dir string) (tcgcollector.PageSource, func()) {
	if dir != "" {
		return tcgcollector.NewDirSource(dir), func() {}
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		logger.Fatal(err)
	}

	return tcgcollector.NewBrowserSource(filepath.Join(cacheDir, "cardstack", "tcgcollector-chrome"))
}
