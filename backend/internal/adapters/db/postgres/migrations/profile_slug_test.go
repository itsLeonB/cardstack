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

const profileSlugMigration = "20261029000000_profile_slug.sql"

// TestProfileSlugMigration_BackfillsExistingProfiles runs the migration's Down
// then Up inside one rolled-back transaction. Down restores the column-less
// table, which is the state a real upgrade starts from.
func TestProfileSlugMigration_BackfillsExistingProfiles(t *testing.T) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		envOr("DB_HOST", "localhost"), envOr("DB_PORT", "5432"), envOr("DB_USER", "cardstack"),
		envOr("DB_PASSWORD", "cardstack"), envOr("DB_NAME", "cardstack"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, Up(t.Context(), sqlDB))

	raw, err := Migrations.ReadFile(profileSlugMigration)
	require.NoError(t, err)
	_, afterUp, _ := strings.Cut(string(raw), "-- +goose Up")
	up, down, found := strings.Cut(afterUp, "-- +goose Down")
	require.True(t, found)

	require.NoError(t, runMigrationInRollback(t, db, func(tx *gorm.DB) {
		require.NoError(t, tx.Exec(down).Error)

		insertProfile := func(name string) (id string) {
			s := uuid.NewString()
			var userID string
			require.NoError(t, tx.Raw(`INSERT INTO users (auth_provider, auth_subject, email) VALUES ('clerk', ?, ?) RETURNING id`, "mig-"+s, s+"@example.com").Scan(&userID).Error)
			require.NoError(t, tx.Raw(`INSERT INTO user_profiles (user_id, name) VALUES (?, ?) RETURNING id`, userID, name).Scan(&id).Error)
			return id
		}
		ada := insertProfile("Ada  Lovelace")
		adaTwin := insertProfile("Ada  Lovelace")
		emoji := insertProfile("🔥🔥")

		require.NoError(t, tx.Exec(up).Error)

		slugOf := func(id string) (slug string) {
			require.NoError(t, tx.Raw(`SELECT slug FROM user_profiles WHERE id = ?`, id).Scan(&slug).Error)
			return slug
		}
		assert.Equal(t, "ada-lovelace-"+ada[len(ada)-5:], slugOf(ada))
		assert.Equal(t, "ada-lovelace-"+adaTwin[len(adaTwin)-5:], slugOf(adaTwin))
		assert.NotEqual(t, slugOf(ada), slugOf(adaTwin))
		assert.Equal(t, "user-"+emoji[len(emoji)-5:], slugOf(emoji))

		var nullable string
		require.NoError(t, tx.Raw(`SELECT is_nullable FROM information_schema.columns WHERE table_name = 'user_profiles' AND column_name = 'slug'`).Scan(&nullable).Error)
		assert.Equal(t, "NO", nullable, "the column is required once backfilled")
	}))
}
