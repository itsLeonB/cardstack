// Package migrations holds the go:embed FS for goose migrations, consumed
// by internal/adapters/job/migrate. cmd/api never imports this package, so
// embedding the migration SQL here doesn't bloat the API binary — only
// cmd/job (which does import it, via the migrate package) pays for it.
package migrations

import (
	"context"
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed *.sql
var Migrations embed.FS

// Up applies every pending migration while holding a Postgres advisory lock,
// so callers migrating the same database at once wait for each other instead
// of racing: `go test ./...` runs each package's tests in its own process, and
// on a fresh database each package's test helper would otherwise apply the
// same migrations concurrently.
func Up(ctx context.Context, db *sql.DB, opts ...goose.ProviderOption) error {
	// Poll every second, for up to five minutes.
	locker, err := lock.NewPostgresSessionLocker(lock.WithLockTimeout(1, 300))
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, Migrations, append(opts, goose.WithSessionLocker(locker))...)
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}
