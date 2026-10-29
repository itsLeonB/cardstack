package migrations

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
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
	ada, adaTwin, emoji := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	runProfileSlugBackfill(t, func(tx *gorm.DB) {
		seedProfile(t, tx, ada, "Ada  Lovelace")
		seedProfile(t, tx, adaTwin, "Ada  Lovelace")
		seedProfile(t, tx, emoji, "🔥🔥")
	}, func(tx *gorm.DB) {
		assert.Equal(t, "ada-lovelace-"+ada.String()[31:], slugOfProfile(t, tx, ada))
		assert.Equal(t, "ada-lovelace-"+adaTwin.String()[31:], slugOfProfile(t, tx, adaTwin))
		assert.NotEqual(t, slugOfProfile(t, tx, ada), slugOfProfile(t, tx, adaTwin))
		assert.Equal(t, "user-"+emoji.String()[31:], slugOfProfile(t, tx, emoji))

		var nullable string
		require.NoError(t, tx.Raw(`SELECT is_nullable FROM information_schema.columns WHERE table_name = 'user_profiles' AND column_name = 'slug'`).Scan(&nullable).Error)
		assert.Equal(t, "NO", nullable, "the column is required once backfilled")
	})
}

// TestProfileSlugMigration_DisambiguatesSuffixCollisions gives two profiles the
// same stem and the same last 5 hex of their ids, so their short candidates
// clash on idx_user_profiles_slug. Both get the full id as a disambiguator; a
// profile with no clash keeps its short slug.
func TestProfileSlugMigration_DisambiguatesSuffixCollisions(t *testing.T) {
	clashA := uuid.MustParse("0192f3a4-0001-7000-8000-00000000abcd")
	clashB := uuid.MustParse("0192f3a4-0002-7000-8000-00000000abcd")
	other := uuid.MustParse("0192f3a4-0003-7000-8000-000000001234")

	runProfileSlugBackfill(t, func(tx *gorm.DB) {
		seedProfile(t, tx, clashA, "Ada Lovelace")
		seedProfile(t, tx, clashB, "Ada Lovelace")
		seedProfile(t, tx, other, "Grace Hopper")
	}, func(tx *gorm.DB) {
		assert.Equal(t, "ada-lovelace-"+strings.ReplaceAll(clashA.String(), "-", ""), slugOfProfile(t, tx, clashA))
		assert.Equal(t, "ada-lovelace-"+strings.ReplaceAll(clashB.String(), "-", ""), slugOfProfile(t, tx, clashB))
		assert.Equal(t, "grace-hopper-01234", slugOfProfile(t, tx, other))
	})
}

// TestProfileSlugMigration_MatchesSlugOfForLongNames holds the backfill's length
// bound to mapper.SlugOf's, so a profile gets the same slug from either path.
func TestProfileSlugMigration_MatchesSlugOfForLongNames(t *testing.T) {
	id := uuid.MustParse("0192f3a4-0004-7000-8000-000000005678")
	name := strings.Repeat("Ada Lovelace ", 20)

	runProfileSlugBackfill(t, func(tx *gorm.DB) {
		seedProfile(t, tx, id, name)
	}, func(tx *gorm.DB) {
		assert.Equal(t, mapper.SlugOf(name, "user", id), slugOfProfile(t, tx, id))
	})
}

// runProfileSlugBackfill runs the migration's Down, seeds profiles, then its Up,
// and hands the transaction to check, all inside one rolled-back transaction.
// Down restores the column-less table, which is the state a real upgrade starts from.
func runProfileSlugBackfill(t *testing.T, seed func(tx *gorm.DB), check func(tx *gorm.DB)) {
	t.Helper()
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
		seed(tx)
		require.NoError(t, tx.Exec(up).Error)
		check(tx)
	}))
}

// seedProfile inserts a profile with a chosen id on a fresh user, so a test can
// pick ids that share a slug suffix.
func seedProfile(t *testing.T, tx *gorm.DB, id uuid.UUID, name string) {
	t.Helper()
	s := uuid.NewString()
	var userID string
	require.NoError(t, tx.Raw(`INSERT INTO users (auth_provider, auth_subject, email) VALUES ('clerk', ?, ?) RETURNING id`, "mig-"+s, s+"@example.com").Scan(&userID).Error)
	require.NoError(t, tx.Exec(`INSERT INTO user_profiles (id, user_id, name) VALUES (?, ?, ?)`, id, userID, name).Error)
}

func slugOfProfile(t *testing.T, tx *gorm.DB, id uuid.UUID) (slug string) {
	t.Helper()
	require.NoError(t, tx.Raw(`SELECT slug FROM user_profiles WHERE id = ?`, id).Scan(&slug).Error)
	return slug
}
