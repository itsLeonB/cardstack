// Command ingest-tcgcollector compares TCG Collector's Indonesian sets list
// with the catalog and prints what is missing. It is a dry run unless -apply
// is set: then it creates the missing sets unpublished, adds the missing Cards
// to the listed sets, and fills empty set logos; nothing else is written. It
// prints its report to stdout, logs go to
// stderr. Manually triggered only. Without -dir it starts a visible Chrome
// itself and fetches through it, so solve TCG Collector's challenge in that
// window once; later runs reuse the profile and read fetched pages from the
// page cache. With -dir it reads pages saved by hand instead
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
	apply := flag.Bool("apply", false, "create the missing sets unpublished and fill empty set logos; without it the command only reports")
	refresh := flag.Bool("refresh", false, "refetch the pages this run reads instead of using the page cache; with -set, only that set's pages and the sets page")
	flag.Parse()

	// The report is the command's output, so the logs go to stderr. The shared
	// logger takes os.Stdout when it is initialised, so it is initialised while
	// stdout points at stderr.
	stdout := os.Stdout
	os.Stdout = os.Stderr
	logger.Init("IngestTCGCollector")
	os.Stdout = stdout

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
	report, err := ingester.Run(context.Background(), tcgcollector.Options{SetCode: *set, Apply: *apply, Refresh: *refresh})
	if err != nil {
		logger.Fatal(err)
	}

	if err := report.Render(stdout); err != nil {
		logger.Fatal(err)
	}

	if sets, cards := report.Failures(); sets+cards > 0 {
		logger.Fatalf("%d sets failed to write and %d cards failed: see the report", sets, cards)
	}
}

// newPageSource reads saved pages from dir when it is set. Otherwise it opens
// the operator's visible Chrome, whose profile lives in the user cache dir, and
// keeps every page it fetches in the page cache beside it.
func newPageSource(dir string) (tcgcollector.PageSource, func()) {
	if dir != "" {
		return tcgcollector.NewDirSource(dir), func() {}
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		logger.Fatal(err)
	}

	browser, closeBrowser := tcgcollector.NewBrowserSource(filepath.Join(cacheDir, "cardstack", "tcgcollector-chrome"))
	cache := tcgcollector.NewCacheSource(filepath.Join(cacheDir, "cardstack", "tcgcollector-pages"), browser)
	return cache, closeBrowser
}
