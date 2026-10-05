// Command host-series-images uploads the Series logos under assets/series
// (built by scripts/build-series-logos.sh) to the R2 bucket as
// series/<code>.<hash8>.webp and stores the key on the matching Series row
// (docs/adr/0017). Re-running with unchanged files changes nothing. It exits 1
// if an upload failed or a file's code matched no Series row. Manually
// triggered, never on Railway.
package main

import (
	"context"
	"flag"
	"os"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	crud "github.com/itsLeonB/go-crud"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	dir := flag.String("dir", "assets/series", "directory of <series code>.webp logo files")
	flag.Parse()

	logger.Init("HostSeriesImages")

	if err := config.Load(); err != nil {
		logger.Fatal(err)
	}
	if !config.Global.Configured() {
		logger.Fatal("R2 is not configured: set R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY and R2_BUCKET")
	}

	providers, cleanup, err := provider.InitializeProviders()
	if err != nil {
		logger.Fatal(err)
	}
	defer cleanup()

	sum, err := hostLogos(context.Background(), *dir, crud.NewRepository[entity.Series](providers.Gorm), objectstore.NewR2Store(config.Global.R2))
	logger.Infof("matched %d series row(s), uploaded %d logo(s)", sum.Matched, sum.Uploaded)
	if err != nil {
		// Non-zero exit tells an operator a re-run is needed.
		cleanup()
		os.Exit(1)
	}
}
