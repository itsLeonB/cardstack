// Command presize-images resizes the hosted original of every Expansion Set
// that has one (image_key) but no cover (cover_key) to 128px on its longest
// side as WebP with alpha, uploads it to the R2 bucket as
// expansion-sets/<id>.<hash8>.webp and stores the key in cover_key
// (docs/adr/0017). It reads the originals from R2, never from pokemonasia, and
// leaves them and image_key untouched. It shells out to vips and exits
// non-zero before touching anything if vips is missing. Re-running only
// handles sets still missing a cover; it exits 1 if any set failed.
// Manually triggered, never on Railway.
package main

import (
	"context"
	"flag"
	"os"
	"os/exec"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	corestore "github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	set := flag.String("set", "", "limit the run to one Expansion Set by its site code (e.g. \"MA6\"); default sizes every set missing a cover")
	flag.Parse()

	logger.Init("PresizeImages")

	os.Exit(run(context.Background(), *set, exec.LookPath, vipsResize, openDeps))
}

// deps is what the run needs from the outside world, built only after vips is
// known to be present.
type deps struct {
	sets  repository.ExpansionSetRepository
	store corestore.ObjectStore
}

// openDeps loads the configuration and opens the database and the R2 store. The
// returned cleanup closes the database.
func openDeps() (deps, func(), error) {
	if err := config.Load(); err != nil {
		return deps{}, nil, err
	}
	if !config.Global.Configured() {
		return deps{}, nil, ungerr.Unknown("R2 is not configured: set R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY and R2_BUCKET")
	}
	providers, cleanup, err := provider.InitializeProviders()
	if err != nil {
		return deps{}, nil, err
	}
	return deps{
		sets:  repository.NewExpansionSetRepository(crud.NewRepository[entity.ExpansionSet](providers.Gorm)),
		store: objectstore.NewR2Store(config.Global.R2),
	}, cleanup, nil
}

// run returns the process exit code. It checks for vips first, before the
// configuration, the database or the bucket is touched.
func run(ctx context.Context, setCode string, lookPath func(string) (string, error), resize resizeFunc, open func() (deps, func(), error)) int {
	if err := requireVips(lookPath); err != nil {
		logger.Error(err)
		return 1
	}

	d, cleanup, err := open()
	if err != nil {
		logger.Error(err)
		return 1
	}
	defer cleanup()

	sum, err := presizeCovers(ctx, d.sets, d.store, resize, setCode)
	logger.Infof("%d set(s) needed a cover: %d sized, %d failed", sum.Selected, sum.Resized, sum.Failed)
	if err != nil {
		// Non-zero exit tells an operator a re-run is needed.
		return 1
	}
	return 0
}
