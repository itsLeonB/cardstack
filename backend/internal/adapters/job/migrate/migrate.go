package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/db/postgres/migrations"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	"github.com/pressly/goose/v3"
)

type Migrate struct {
	db *sql.DB
}

func Setup(providers *provider.Providers) (*Migrate, error) {
	return &Migrate{providers.SQL}, nil
}

func (m *Migrate) Run() error {
	if err := migrations.Up(context.Background(), m.db, goose.WithLogger(logger.Global), goose.WithVerbose(true)); err != nil {
		return fmt.Errorf("error running migrations: %w", err)
	}
	return nil
}
