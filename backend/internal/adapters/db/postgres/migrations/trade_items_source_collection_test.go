package migrations

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const tradeItemsSourceCollectionMigration = "20261202000000_trade_items_source_collection_set_null.sql"

// TestTradeItemsSourceCollectionMigration_DownFailsOnDetachedItem runs the
// migration's Down while a Trade Item's source Collection is already deleted,
// inside one rolled-back transaction. Down must fail and keep every item:
// SET NOT NULL cannot hold with NULL sources, and a rollback must never
// silently delete agreed deal items.
func TestTradeItemsSourceCollectionMigration_DownFailsOnDetachedItem(t *testing.T) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		envOr("DB_HOST", "localhost"), envOr("DB_PORT", "5432"), envOr("DB_USER", "cardstack"),
		envOr("DB_PASSWORD", "cardstack"), privateDB(t))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, Up(t.Context(), sqlDB))

	raw, err := Migrations.ReadFile(tradeItemsSourceCollectionMigration)
	require.NoError(t, err)
	_, afterUp, _ := strings.Cut(string(raw), "-- +goose Up")
	up, down, found := strings.Cut(afterUp, "-- +goose Down")
	require.True(t, found)

	// First Down must succeed with no detached item: it is the state a real
	// downgrade starts from.
	require.NoError(t, runMigrationInRollback(t, db, func(tx *gorm.DB) {
		require.NoError(t, tx.Exec(down).Error)
		require.NoError(t, tx.Exec(up).Error)
	}))

	tradeID := seedTradeWithDetachedItem(t, db)

	require.NoError(t, runMigrationInRollback(t, db, func(tx *gorm.DB) {
		require.NoError(t, tx.Exec(`SAVEPOINT before_down`).Error)

		err := tx.Exec(down).Error
		require.Error(t, err, "Down cannot restore NOT NULL while a source is NULL")

		// The failed statement aborts the transaction; roll back to the
		// savepoint to check what survived.
		require.NoError(t, tx.Exec(`ROLLBACK TO SAVEPOINT before_down`).Error)

		var count int64
		require.NoError(t, tx.Raw(`SELECT count(*) FROM trade_items WHERE trade_id = ?`, tradeID).Scan(&count).Error)
		assert.EqualValues(t, 2, count, "Down deletes nothing: both agreed items stay")

		var nullable string
		require.NoError(t, tx.Raw(`SELECT is_nullable FROM information_schema.columns WHERE table_name = 'trade_items' AND column_name = 'source_collection_id'`).Scan(&nullable).Error)
		assert.Equal(t, "YES", nullable, "Up's schema is untouched by the failed Down")

		var fkCount int64
		require.NoError(t, tx.Raw(`SELECT count(*) FROM information_schema.table_constraints WHERE table_name = 'trade_items' AND constraint_name = 'trade_items_source_collection_id_fkey'`).Scan(&fkCount).Error)
		assert.EqualValues(t, 1, fkCount, "the failed Down's DROP CONSTRAINT rolled back too")

		require.NoError(t, tx.Exec(up).Error)
	}))
}

// seedTradeWithDetachedItem creates a Trade with two Trade Items on the owner
// side, then deletes one item's source Collection so SET NULL detaches it,
// and returns the Trade id. Both items stay in trade_items.
func seedTradeWithDetachedItem(t *testing.T, db *gorm.DB) string {
	t.Helper()

	s := uuid.NewString()
	var tradeID string
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var userID, profileID, gameID, localeID, rarityID, setID, card1ID, card2ID string
		require.NoError(t, tx.Raw(`INSERT INTO users (auth_provider, auth_subject, email) VALUES ('clerk', ?, ?) RETURNING id`, "mig-"+s, s+"@example.com").Scan(&userID).Error)
		require.NoError(t, tx.Raw(`INSERT INTO user_profiles (user_id, name, slug) VALUES (?, ?, ?) RETURNING id`, userID, "Mig", "mig-"+s).Scan(&profileID).Error)

		require.NoError(t, tx.Raw(`INSERT INTO games (slug, name) VALUES (?, ?) RETURNING id`, "mig-"+s, "Mig").Scan(&gameID).Error)
		require.NoError(t, tx.Raw(`INSERT INTO locales (code) VALUES (?) RETURNING id`, "mig-"+s).Scan(&localeID).Error)
		require.NoError(t, tx.Raw(`INSERT INTO rarities (game_id, code, name) VALUES (?, 'C', 'Common') RETURNING id`, gameID).Scan(&rarityID).Error)
		require.NoError(t, tx.Raw(`INSERT INTO expansion_sets (game_id, code, name, locale_id, created_by, updated_by) VALUES (?, ?, 'Set', ?, 'pokemon-asia', 'pokemon-asia') RETURNING id`, gameID, "mig-"+s, localeID).Scan(&setID).Error)
		require.NoError(t, tx.Raw(`INSERT INTO cards (expansion_set_id, local_id, name, category, tags, rarity_id, attributes, created_by, updated_by) VALUES (?, '001', 'Card', 'Pokémon', '[]', ?, '{}', 'pokemon-asia', 'pokemon-asia') RETURNING id`, setID, rarityID).Scan(&card1ID).Error)
		require.NoError(t, tx.Raw(`INSERT INTO cards (expansion_set_id, local_id, name, category, tags, rarity_id, attributes, created_by, updated_by) VALUES (?, '002', 'Card Two', 'Pokémon', '[]', ?, '{}', 'pokemon-asia', 'pokemon-asia') RETURNING id`, setID, rarityID).Scan(&card2ID).Error)

		var collectionID string
		require.NoError(t, tx.Raw(`INSERT INTO collections (profile_id, title) VALUES (?, 'Box') RETURNING id`, profileID).Scan(&collectionID).Error)

		require.NoError(t, tx.Raw(`INSERT INTO trades (slug, owner_profile_id, status) VALUES (?, ?, 'open') RETURNING id`, "trade-"+s, profileID).Scan(&tradeID).Error)
		require.NoError(t, tx.Exec(`INSERT INTO trade_items (trade_id, side, card_id, source_collection_id, quantity) VALUES (?, 0, ?, ?, 1)`, tradeID, card1ID, collectionID).Error)
		require.NoError(t, tx.Exec(`INSERT INTO trade_items (trade_id, side, card_id, source_collection_id, quantity) VALUES (?, 0, ?, ?, 1)`, tradeID, card2ID, collectionID).Error)

		// Deleting the source Collection detaches the items via ON DELETE
		// SET NULL — the agreed deal outlives its source.
		require.NoError(t, tx.Exec(`DELETE FROM collections WHERE id = ?`, collectionID).Error)
		return nil
	}))
	return tradeID
}
