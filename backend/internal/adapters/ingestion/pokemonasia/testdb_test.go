package pokemonasia

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/db/postgres/migrations"
	"github.com/kelseyhightower/envconfig"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// testDBConfig holds this file's Postgres DSN pieces, loaded via envconfig
// with the same bare DB_* env var names internal/adapters/repository's
// testDB helper reads manually.
type testDBConfig struct {
	Host     string `envconfig:"DB_HOST" default:"localhost"`
	Port     string `envconfig:"DB_PORT" default:"5432"`
	User     string `envconfig:"DB_USER" default:"cardstack"`
	Password string `envconfig:"DB_PASSWORD" default:"cardstack"`
	Name     string `envconfig:"DB_NAME" default:"cardstack"`
}

// testDB takes the same real-Postgres testing approach as
// internal/adapters/repository's testDB helper (see its doc comment for how
// to start a matching local Postgres, per docs/adr/0005) — same setup
// ticket 03's tcgdex ingester used. It deliberately does not truncate
// tables — this DB is shared with other packages' tests — so tests use
// uniqueCode for one-of-a-kind natural keys.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	var cfg testDBConfig
	if err := envconfig.Process("", &cfg); err != nil {
		t.Fatalf("loading test DB config: %v", err)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name,
	)

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

// uniqueCode returns a set/series/rarity code unique to this test run, so
// tests never collide with each other or leftover rows regardless of run
// order (mirrors internal/adapters/repository's uniqueEmail).
func uniqueCode(t *testing.T) string {
	t.Helper()
	return uuid.NewString()
}
