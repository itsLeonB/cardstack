package repository

import (
	"cmp"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/db/postgres/migrations"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testDB opens a connection to a real local Postgres instance (per
// docs/adr/0005: repository tests run against real Postgres, not mocks) and
// migrates it. It reads the same DB_* env vars backend-ci.yml's test job
// sets; locally, run a matching Postgres 18 container, e.g.:
//
//	docker run -d -p 5432:5432 -e POSTGRES_USER=cardstack \
//	  -e POSTGRES_PASSWORD=cardstack -e POSTGRES_DB=cardstack postgres:18
//
// It deliberately does not truncate tables: this database is shared with
// other packages' tests (e.g. internal/adapters/http/routes), which `go
// test ./...` runs concurrently in separate processes, so a truncate here
// could wipe rows another package's test is mid-assertion on. Tests instead
// use uniqueEmail/uuid.NewString() for one-of-a-kind data, so they never
// collide with each other regardless of run order or table contents left
// over from previous runs.
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
		t.Fatalf("connecting to test postgres (see testDB's doc comment for how to start one locally): %v", err)
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

// uniqueEmail returns an email address unique to this test run, so tests
// never collide over a shared row regardless of execution order or leftover
// data from previous runs (see testDB's doc comment).
func uniqueEmail(t *testing.T) string {
	t.Helper()
	return uuid.NewString() + "@example.com"
}

// uniqueHash returns a token_hash value unique to this test run — refresh
// tokens' token_hash column is uniquely indexed, so a fixed literal would
// collide with a previous run's leftover row (see testDB's doc comment).
func uniqueHash(t *testing.T) string {
	t.Helper()
	return uuid.NewString()
}
