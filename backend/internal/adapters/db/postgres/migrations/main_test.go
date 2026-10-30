package migrations

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// The tests in this package get a database of their own. They run a
// migration's Down and Up in a rolled-back transaction, and that DDL holds
// ACCESS EXCLUSIVE locks on tables such as cards until the rollback. `go test
// ./...` runs other packages' tests against the shared test database at the
// same time, and a query that locks the same tables in the opposite order
// deadlocks with the DDL (SQLSTATE 40P01), failing whichever side Postgres
// picks. A private database has no other client, so there is nothing to wait on.
var private struct {
	once  sync.Once
	admin *gorm.DB // connected to the shared database, which owns CREATE and DROP
	name  string
	err   error
}

// TestMain drops the private database once the tests are done. It is created
// on first use, so listing tests or running none needs no Postgres.
func TestMain(m *testing.M) {
	code := m.Run()
	if private.admin != nil {
		// FORCE drops it even if a test left a connection open.
		if err := private.admin.Exec(`DROP DATABASE IF EXISTS ` + private.name + ` WITH (FORCE)`).Error; err != nil {
			fmt.Fprintf(os.Stderr, "migrations tests: drop %s: %v\n", private.name, err)
		}
	}
	os.Exit(code)
}

// privateDB returns the name of this package's database, creating it on the
// first call.
func privateDB(t *testing.T) string {
	t.Helper()
	private.once.Do(func() {
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			envOr("DB_HOST", "localhost"), envOr("DB_PORT", "5432"), envOr("DB_USER", "cardstack"),
			envOr("DB_PASSWORD", "cardstack"), envOr("DB_NAME", "cardstack"))
		admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
		if err != nil {
			private.err = fmt.Errorf("connect to the test database: %w", err)
			return
		}
		// The suffix keeps concurrent runs and the leftover of an interrupted one apart.
		suffix := make([]byte, 4)
		if _, err := rand.Read(suffix); err != nil {
			private.err = err
			return
		}
		name := fmt.Sprintf("migrations_test_%d_%s", os.Getpid(), hex.EncodeToString(suffix))
		if err := admin.Exec(`CREATE DATABASE ` + name).Error; err != nil {
			private.err = fmt.Errorf("create %s: %w", name, err)
			return
		}
		private.admin, private.name = admin, name
	})
	require.NoError(t, private.err)
	return private.name
}
