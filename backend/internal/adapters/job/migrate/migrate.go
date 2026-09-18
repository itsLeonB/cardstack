package migrate

import (
	"database/sql"
	"fmt"

	appembed "github.com/itsLeonB/cardstack/backend"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	"github.com/pressly/goose/v3"
)

type Migrate struct {
	db *sql.DB
}

func Setup(providers *provider.Providers) (*Migrate, error) {
	goose.SetBaseFS(appembed.Migrations)
	goose.SetLogger(logger.Global)

	if err := goose.SetDialect("postgres"); err != nil {
		return nil, fmt.Errorf("error setting migrator dialect to postgres: %w", err)
	}

	return &Migrate{providers.SQL}, nil
}

func (m *Migrate) Run() error {
	if err := goose.Up(m.db, "internal/adapters/db/postgres/migrations"); err != nil {
		return fmt.Errorf("error running migrations: %w", err)
	}
	return nil
}
