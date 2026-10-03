package migrations

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const hostedImagesMigration = "20261004000000_hosted_images.sql"

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// TestHostedImagesMigration_BackfillsSourceAndLeavesKeyEmpty runs the
// migration's Down then Up inside one rolled-back transaction, so the shared
// test database is never left changed. Down restores the pre-migration
// image_url column, which is the state a real upgrade starts from.
func TestHostedImagesMigration_BackfillsSourceAndLeavesKeyEmpty(t *testing.T) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		envOr("DB_HOST", "localhost"), envOr("DB_PORT", "5432"), envOr("DB_USER", "cardstack"),
		envOr("DB_PASSWORD", "cardstack"), envOr("DB_NAME", "cardstack"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)

	// Down below needs the migration applied; goose.Up is a no-op once it is.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	goose.SetBaseFS(Migrations)
	require.NoError(t, goose.SetDialect("postgres"))
	require.NoError(t, goose.Up(sqlDB, "."))

	raw, err := Migrations.ReadFile(hostedImagesMigration)
	require.NoError(t, err)
	_, afterUp, _ := strings.Cut(string(raw), "-- +goose Up")
	up, down, found := strings.Cut(afterUp, "-- +goose Down")
	require.True(t, found)

	require.NoError(t, runMigrationInRollback(t, db, func(tx *gorm.DB) {
		require.NoError(t, tx.Exec(down).Error)

		s := uuid.NewString()
		var gameID, localeID, rarityID, setID string
		require.NoError(t, tx.Raw(`INSERT INTO games (slug, name) VALUES (?, ?) RETURNING id`, "mig-"+s, "Mig").Scan(&gameID).Error)
		require.NoError(t, tx.Raw(`INSERT INTO locales (code) VALUES (?) RETURNING id`, "mig-"+s).Scan(&localeID).Error)
		require.NoError(t, tx.Raw(`INSERT INTO rarities (game_id, code, name) VALUES (?, 'C', 'Common') RETURNING id`, gameID).Scan(&rarityID).Error)
		require.NoError(t, tx.Raw(`INSERT INTO expansion_sets (game_id, code, name, locale_id, image_url) VALUES (?, ?, 'Set', ?, 'https://source.test/set.png') RETURNING id`, gameID, "mig-"+s, localeID).Scan(&setID).Error)
		require.NoError(t, tx.Exec(`INSERT INTO cards (expansion_set_id, local_id, name, category, tags, rarity_id, attributes, image_url) VALUES (?, '001', 'Card', 'Pokémon', '[]', ?, '{}', 'https://source.test/card.png')`, setID, rarityID).Error)

		require.NoError(t, tx.Exec(up).Error)

		var card struct{ SourceImageURL, ImageKey string }
		require.NoError(t, tx.Raw(`SELECT source_image_url, image_key FROM cards WHERE expansion_set_id = ?`, setID).Scan(&card).Error)
		assert.Equal(t, "https://source.test/card.png", card.SourceImageURL)
		assert.Empty(t, card.ImageKey)

		var set struct{ SourceImageURL, ImageKey string }
		require.NoError(t, tx.Raw(`SELECT source_image_url, image_key FROM expansion_sets WHERE id = ?`, setID).Scan(&set).Error)
		assert.Equal(t, "https://source.test/set.png", set.SourceImageURL)
		assert.Empty(t, set.ImageKey)

		var cols int
		require.NoError(t, tx.Raw(`SELECT count(*) FROM information_schema.columns WHERE table_name IN ('cards','expansion_sets') AND column_name = 'image_url'`).Scan(&cols).Error)
		assert.Zero(t, cols, "the old address column is gone")
	}))
}

// runMigrationInRollback runs fn in a transaction that is always rolled back.
func runMigrationInRollback(t *testing.T, db *gorm.DB, fn func(tx *gorm.DB)) error {
	t.Helper()
	errRollback := errors.New("rollback")
	err := db.Transaction(func(tx *gorm.DB) error {
		fn(tx)
		return errRollback
	})
	if errors.Is(err, errRollback) {
		return nil
	}
	return err
}
