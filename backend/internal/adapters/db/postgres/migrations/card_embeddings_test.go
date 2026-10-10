package migrations

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const cardEmbeddingsMigration = "20261010000000_card_embeddings.sql"

// TestCardEmbeddingsMigration_DownThenUpRestoresTheTable runs the migration's
// Down then Up inside one rolled-back transaction, so the shared test
// database is never left changed. It needs pgvector on the server.
func TestCardEmbeddingsMigration_DownThenUpRestoresTheTables(t *testing.T) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		envOr("DB_HOST", "localhost"), envOr("DB_PORT", "5432"), envOr("DB_USER", "cardstack"),
		envOr("DB_PASSWORD", "cardstack"), envOr("DB_NAME", "cardstack"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	goose.SetBaseFS(Migrations)
	require.NoError(t, goose.SetDialect("postgres"))
	require.NoError(t, goose.Up(sqlDB, "."))

	raw, err := Migrations.ReadFile(cardEmbeddingsMigration)
	require.NoError(t, err)
	_, afterUp, _ := strings.Cut(string(raw), "-- +goose Up")
	up, down, found := strings.Cut(afterUp, "-- +goose Down")
	require.True(t, found)

	require.NoError(t, runMigrationInRollback(t, db, func(tx *gorm.DB) {
		require.NoError(t, tx.Exec(down).Error)

		var tables int
		require.NoError(t, tx.Raw(`SELECT count(*) FROM information_schema.tables WHERE table_name IN ('card_embeddings', 'embedding_batches', 'embedding_batch_cards')`).Scan(&tables).Error)
		assert.Zero(t, tables, "Down removes all three tables")

		require.NoError(t, tx.Exec(up).Error)

		var columnType string
		require.NoError(t, tx.Raw(`SELECT format_type(atttypid, atttypmod) FROM pg_attribute WHERE attrelid = 'card_embeddings'::regclass AND attname = 'embedding'`).Scan(&columnType).Error)
		assert.Equal(t, "vector(1536)", columnType)

		var hnsw int
		require.NoError(t, tx.Raw(`SELECT count(*) FROM pg_indexes WHERE tablename = 'card_embeddings' AND indexdef LIKE '%hnsw%vector_cosine_ops%'`).Scan(&hnsw).Error)
		assert.Equal(t, 1, hnsw, "the HNSW cosine index is recreated")

		var unique int
		require.NoError(t, tx.Raw(`SELECT count(*) FROM pg_indexes WHERE tablename = 'card_embeddings' AND indexdef LIKE 'CREATE UNIQUE INDEX%(card_id, model, source)%'`).Scan(&unique).Error)
		assert.Equal(t, 1, unique, "one row per card, model and source")

		var batchTables int
		require.NoError(t, tx.Raw(`SELECT count(*) FROM information_schema.tables WHERE table_name IN ('embedding_batches', 'embedding_batch_cards')`).Scan(&batchTables).Error)
		assert.Equal(t, 2, batchTables, "the batch tables are recreated")
	}))
}
