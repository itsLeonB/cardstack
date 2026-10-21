package main

import (
	"cmp"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/db/postgres/migrations"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// testDB opens the local test Postgres (docs/agents/testing.md) and migrates it,
// as the repository and route tests do. It truncates nothing: the database is
// shared with other packages' tests, so each test uses its own embedding model
// name and its own set.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "host=" + envOr("DB_HOST", "localhost") +
		" port=" + envOr("DB_PORT", "5432") +
		" user=" + envOr("DB_USER", "cardstack") +
		" password=" + envOr("DB_PASSWORD", "cardstack") +
		" dbname=" + envOr("DB_NAME", "cardstack") +
		" sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err, "connecting to test postgres")
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })

	require.NoError(t, migrations.Up(t.Context(), sqlDB))
	return db
}

func envOr(key, fallback string) string {
	return cmp.Or(os.Getenv(key), fallback)
}

// fixture is one Expansion Set of a test Game, with the rarity its Cards need.
type fixture struct {
	db     *gorm.DB
	set    entity.ExpansionSet
	rarity entity.Rarity
}

func newFixture(t *testing.T, db *gorm.DB) fixture {
	t.Helper()
	suffix := uuid.NewString()

	game := entity.Game{Slug: "tune-scan-" + suffix, Name: "Tune Scan Game " + suffix}
	require.NoError(t, db.Create(&game).Error)
	locale := entity.Locale{Code: "tune-scan-" + suffix}
	require.NoError(t, db.Create(&locale).Error)
	rarity := entity.Rarity{GameID: game.ID, Code: "TS-" + suffix, Name: "Tune Scan Rarity " + suffix}
	require.NoError(t, db.Create(&rarity).Error)
	set := entity.ExpansionSet{
		CreatedBy: entity.SourcePokemonAsia, UpdatedBy: entity.SourcePokemonAsia,
		GameID: game.ID, Code: "tu" + suffix[:8], Name: "Tune Scan Set " + suffix, LocaleID: locale.ID,
	}
	require.NoError(t, db.Create(&set).Error)

	return fixture{db: db, set: set, rarity: rarity}
}

// card inserts a Card of the fixture's set. Local ids must be unique in a set.
func (f fixture) card(t *testing.T, name, localID string) entity.Card {
	t.Helper()
	card := entity.Card{
		CreatedBy: entity.SourcePokemonAsia, UpdatedBy: entity.SourcePokemonAsia,
		ExpansionSetID: f.set.ID, LocalID: localID, Name: name, Category: "Pokémon",
		Tags: datatypes.JSONSlice[string]{}, RarityID: f.rarity.ID, Attributes: datatypes.JSONMap{},
	}
	require.NoError(t, f.db.Create(&card).Error)
	return card
}

// embed stores v as cardID's catalog embedding of model.
func embed(t *testing.T, db *gorm.DB, cardID uuid.UUID, model string, v []float32) {
	t.Helper()
	require.NoError(t, db.Create(&entity.CardEmbedding{CardID: cardID, Model: model, Source: "catalog", Embedding: pgvector.NewVector(v)}).Error)
}

const testEmbeddingDims = 1536

// unitVector is a vector with a single 1 along axis.
func unitVector(axis int) []float32 {
	v := make([]float32, testEmbeddingDims)
	v[axis] = 1
	return v
}

// mixVector puts each weight along its axis. The weights must square to 1 for a
// unit vector, so the cosine with unitVector(a) is weights[a].
func mixVector(weights map[int]float64) []float32 {
	v := make([]float32, testEmbeddingDims)
	for axis, w := range weights {
		v[axis] = float32(w)
	}
	return v
}

// writeCounts counts the rows a report could write for the fixture's set and
// the test model: a report must leave every count as it found it.
func writeCounts(t *testing.T, db *gorm.DB, f fixture, model string) (sets, cards, embeddings int64) {
	t.Helper()
	require.NoError(t, db.Model(&entity.ExpansionSet{}).Where("id = ?", f.set.ID).Count(&sets).Error)
	require.NoError(t, db.Model(&entity.Card{}).Where("expansion_set_id = ?", f.set.ID).Count(&cards).Error)
	require.NoError(t, db.Model(&entity.CardEmbedding{}).Where("model = ?", model).Count(&embeddings).Error)
	return sets, cards, embeddings
}
