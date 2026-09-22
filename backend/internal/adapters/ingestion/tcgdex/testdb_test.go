package tcgdex

import (
	"cmp"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/db/postgres/migrations"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// testDB mirrors internal/adapters/repository's testDB helper (see its doc
// comment for how to start a matching local Postgres): repository-layer
// tests run against a real Postgres per docs/adr/0005, not mocks. It
// deliberately does not truncate tables — this DB is shared with other
// packages' tests — so tests use uniqueCode for one-of-a-kind natural keys.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "host=" + envOr("DB_HOST", "localhost") +
		" port=" + envOr("DB_PORT", "5432") +
		" user=" + envOr("DB_USER", "cardstack") +
		" password=" + envOr("DB_PASSWORD", "cardstack") +
		" dbname=" + envOr("DB_NAME", "cardstack") +
		" sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("connecting to test postgres (see internal/adapters/repository's testDB doc comment for how to start one locally): %v", err)
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

// uniqueCode returns a set/card code unique to this test run, so tests
// never collide with each other or leftover rows regardless of run order
// (mirrors internal/adapters/repository's uniqueEmail).
func uniqueCode(t *testing.T) string {
	t.Helper()
	return uuid.NewString()
}
