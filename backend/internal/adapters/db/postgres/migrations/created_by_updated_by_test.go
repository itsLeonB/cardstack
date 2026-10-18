package migrations

import (
	"fmt"
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

const createdByUpdatedByMigration = "20261017000001_created_by_updated_by.sql"

// TestCreatedByUpdatedByMigration_CuratedRowsBecomeAdminAndTheRestPokemonAsia
// runs the migration's Down then Up inside one rolled-back transaction. Down
// restores the curated column, which is the state a real upgrade starts from.
func TestCreatedByUpdatedByMigration_CuratedRowsBecomeAdminAndTheRestPokemonAsia(t *testing.T) {
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

	raw, err := Migrations.ReadFile(createdByUpdatedByMigration)
	require.NoError(t, err)
	_, afterUp, _ := strings.Cut(string(raw), "-- +goose Up")
	up, down, found := strings.Cut(afterUp, "-- +goose Down")
	require.True(t, found)

	require.NoError(t, runMigrationInRollback(t, db, func(tx *gorm.DB) {
		require.NoError(t, tx.Exec(down).Error)

		s := uuid.NewString()
		var gameID, localeID, rarityID, curatedSetID, asiaSetID string
		require.NoError(t, tx.Raw(`INSERT INTO games (slug, name) VALUES (?, ?) RETURNING id`, "mig-"+s, "Mig").Scan(&gameID).Error)
		require.NoError(t, tx.Raw(`INSERT INTO locales (code) VALUES (?) RETURNING id`, "mig-"+s).Scan(&localeID).Error)
		require.NoError(t, tx.Raw(`INSERT INTO rarities (game_id, code, name) VALUES (?, 'C', 'Common') RETURNING id`, gameID).Scan(&rarityID).Error)
		require.NoError(t, tx.Raw(`INSERT INTO expansion_sets (game_id, code, name, locale_id, curated) VALUES (?, ?, 'Curated', ?, true) RETURNING id`, gameID, "mig-c-"+s, localeID).Scan(&curatedSetID).Error)
		require.NoError(t, tx.Raw(`INSERT INTO expansion_sets (game_id, code, name, locale_id, curated) VALUES (?, ?, 'Asia', ?, false) RETURNING id`, gameID, "mig-a-"+s, localeID).Scan(&asiaSetID).Error)

		var curatedCardID, asiaCardID string
		require.NoError(t, tx.Raw(`INSERT INTO cards (expansion_set_id, local_id, name, category, tags, rarity_id, attributes, curated) VALUES (?, '001', 'Hand', 'Pokémon', '[]', ?, '{}', true) RETURNING id`, curatedSetID, rarityID).Scan(&curatedCardID).Error)
		require.NoError(t, tx.Raw(`INSERT INTO cards (expansion_set_id, local_id, name, category, tags, rarity_id, attributes, curated) VALUES (?, '001', 'Site', 'Pokémon', '[]', ?, '{}', false) RETURNING id`, asiaSetID, rarityID).Scan(&asiaCardID).Error)

		require.NoError(t, tx.Exec(up).Error)

		var curatedCard struct{ CreatedBy, UpdatedBy string }
		require.NoError(t, tx.Raw(`SELECT created_by, updated_by FROM cards WHERE id = ?`, curatedCardID).Scan(&curatedCard).Error)
		assert.Equal(t, "admin", curatedCard.CreatedBy)
		assert.Equal(t, "admin", curatedCard.UpdatedBy)

		var asiaCard struct{ CreatedBy, UpdatedBy string }
		require.NoError(t, tx.Raw(`SELECT created_by, updated_by FROM cards WHERE id = ?`, asiaCardID).Scan(&asiaCard).Error)
		assert.Equal(t, "pokemon-asia", asiaCard.CreatedBy)
		assert.Equal(t, "pokemon-asia", asiaCard.UpdatedBy)

		var curatedSet struct{ CreatedBy, UpdatedBy string }
		require.NoError(t, tx.Raw(`SELECT created_by, updated_by FROM expansion_sets WHERE id = ?`, curatedSetID).Scan(&curatedSet).Error)
		assert.Equal(t, "admin", curatedSet.CreatedBy)
		assert.Equal(t, "admin", curatedSet.UpdatedBy)

		var asiaSet struct{ CreatedBy, UpdatedBy string }
		require.NoError(t, tx.Raw(`SELECT created_by, updated_by FROM expansion_sets WHERE id = ?`, asiaSetID).Scan(&asiaSet).Error)
		assert.Equal(t, "pokemon-asia", asiaSet.CreatedBy)
		assert.Equal(t, "pokemon-asia", asiaSet.UpdatedBy)

		var cols int
		require.NoError(t, tx.Raw(`SELECT count(*) FROM information_schema.columns WHERE table_name IN ('cards','expansion_sets') AND column_name = 'curated'`).Scan(&cols).Error)
		assert.Zero(t, cols, "the curated column is gone")
	}))
}
