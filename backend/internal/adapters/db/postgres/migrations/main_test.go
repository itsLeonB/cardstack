package migrations

import (
	"fmt"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// TestMain gives this package a database of its own. Its tests run a
// migration's Down and Up in a rolled-back transaction, and that DDL holds
// ACCESS EXCLUSIVE locks on tables such as cards until the rollback. `go test
// ./...` runs other packages' tests against the shared test database at the
// same time, and a query that locks the same tables in the opposite order
// deadlocks with the DDL (SQLSTATE 40P01), failing whichever side Postgres
// picks. A private database has no other client, so there is nothing to wait on.
func TestMain(m *testing.M) {
	os.Exit(runInPrivateDatabase(m))
}

func runInPrivateDatabase(m *testing.M) int {
	dsn := func(dbname string) string {
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			envOr("DB_HOST", "localhost"), envOr("DB_PORT", "5432"), envOr("DB_USER", "cardstack"),
			envOr("DB_PASSWORD", "cardstack"), dbname)
	}
	admin, err := gorm.Open(postgres.Open(dsn(envOr("DB_NAME", "cardstack"))), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrations tests: connect to the test database: %v\n", err)
		return 1
	}
	name := fmt.Sprintf("migrations_test_%d", os.Getpid())
	if err := admin.Exec(`CREATE DATABASE ` + name).Error; err != nil {
		fmt.Fprintf(os.Stderr, "migrations tests: create %s: %v\n", name, err)
		return 1
	}
	defer func() {
		// FORCE drops it even if a test left a connection open.
		if err := admin.Exec(`DROP DATABASE IF EXISTS ` + name + ` WITH (FORCE)`).Error; err != nil {
			fmt.Fprintf(os.Stderr, "migrations tests: drop %s: %v\n", name, err)
		}
	}()
	// The tests read DB_NAME when they open their connection.
	if err := os.Setenv("DB_NAME", name); err != nil {
		fmt.Fprintf(os.Stderr, "migrations tests: %v\n", err)
		return 1
	}
	return m.Run()
}
