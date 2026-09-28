package collection

import (
	"cmp"
	"os"
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/db/postgres/migrations"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testDB opens a connection to a real local Postgres instance and migrates
// it (per docs/adr/0005: repository tests run against real Postgres, not
// mocks) - mirrors internal/domain/repository/repository_test_helper_test.go's
// testDB, duplicated here since CollectionRepository lives in its own
// package per docs/adr/0011 rather than internal/domain/repository. See
// that file's doc comment for how to start a matching Postgres locally, and
// docs/agents/testing.md for a cloud/remote agent environment.
//
// It deliberately does not truncate tables: this database is shared with
// other packages' tests, which `go test ./...` runs concurrently in
// separate processes.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "host=" + envOr("DB_HOST", "localhost") +
		" port=" + envOr("DB_PORT", "5432") +
		" user=" + envOr("DB_USER", "cardstack") +
		" password=" + envOr("DB_PASSWORD", "cardstack") +
		" dbname=" + envOr("DB_NAME", "cardstack") +
		" sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connecting to test postgres (see repository_test_helper_test.go's testDB doc comment for how to start one locally): %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("getting sql.DB: %v", err)
	}

	goose.SetBaseFS(migrations.Migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("setting goose dialect: %v", err)
	}
	if err := goose.Up(sqlDB, "."); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	return db
}

func envOr(key, fallback string) string {
	return cmp.Or(os.Getenv(key), fallback)
}
