// Package migrations holds the go:embed FS for goose migrations, consumed
// by internal/adapters/job/migrate. cmd/api never imports this package, so
// embedding the migration SQL here doesn't bloat the API binary — only
// cmd/job (which does import it, via the migrate package) pays for it.
package migrations

import "embed"

//go:embed *.sql
var Migrations embed.FS
