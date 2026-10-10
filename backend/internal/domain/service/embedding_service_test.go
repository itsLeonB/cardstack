package service

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/db/postgres/migrations"
	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/pgvector/pgvector-go"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// embeddingDB opens the test Postgres (the DB_* variables the repository tests
// read) and migrates it. These tests run the batch service against real rows
// and fake only the provider and the image host.
func embeddingDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		envOr("DB_HOST", "localhost"), envOr("DB_PORT", "5432"), envOr("DB_USER", "cardstack"),
		envOr("DB_PASSWORD", "cardstack"), envOr("DB_NAME", "cardstack"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	goose.SetBaseFS(migrations.Migrations)
	require.NoError(t, goose.SetDialect("postgres"))
	require.NoError(t, goose.Up(sqlDB, "."))
	return db
}

func envOr(key, fallback string) string {
	return cmp.Or(os.Getenv(key), fallback)
}

// embedSet is one Expansion Set unique to a test, so the job's set scope never
// reaches rows another test or an earlier run left behind.
type embedSet struct {
	code     string
	id       uuid.UUID
	rarityID uuid.UUID
}

func newEmbedSet(t *testing.T, db *gorm.DB) embedSet {
	t.Helper()
	suffix := uuid.NewString()

	game := entity.Game{Slug: "embed-" + suffix, Name: "Embed Test " + suffix}
	require.NoError(t, db.Create(&game).Error)
	locale := entity.Locale{Code: "embed-" + suffix}
	require.NoError(t, db.Create(&locale).Error)
	rarity := entity.Rarity{GameID: game.ID, Code: "E-" + suffix, Name: "Embed Test Rarity"}
	require.NoError(t, db.Create(&rarity).Error)
	set := entity.ExpansionSet{GameID: game.ID, Code: "embed-set-" + suffix, Name: "Embed Test Set", LocaleID: locale.ID}
	require.NoError(t, db.Create(&set).Error)

	return embedSet{code: set.Code, id: set.ID, rarityID: rarity.ID}
}

// addCard inserts a card in set with the given local ID and hosted image key
// ("" for none).
func addCard(t *testing.T, db *gorm.DB, set embedSet, localID, imageKey string) entity.Card {
	t.Helper()
	card := entity.Card{
		ExpansionSetID: set.id,
		LocalID:        localID,
		Name:           "Embed Test Card " + localID,
		Category:       "Pokémon",
		Tags:           datatypes.JSONSlice[string]{},
		RarityID:       set.rarityID,
		Attributes:     datatypes.JSONMap{},
		ImageKey:       imageKey,
	}
	require.NoError(t, db.Create(&card).Error)
	return card
}

// unit is a 1536-dimensional unit vector, the shape the provider returns.
func unit() []float32 {
	v := make([]float32, 1536)
	v[0] = 1
	return v
}

// newProviderMocks returns the provider and image host fakes with no
// expectations set; each test states the calls it expects.
func newProviderMocks(t *testing.T) (*mocks.MockEmbedder, *mocks.MockImageFetcher) {
	t.Helper()
	return mocks.NewMockEmbedder(t), mocks.NewMockImageFetcher(t)
}

func newEmbeddingJob(db *gorm.DB, embedder embedding.Embedder, fetcher embedding.ImageFetcher, model string) EmbeddingService {
	return NewEmbeddingService(
		repository.NewEmbeddingRepository(crud.NewRepository[entity.CardEmbedding](db)),
		embedder,
		fetcher,
		mapper.NewImageHost(testImageBase),
		model,
	)
}

// embeddedCardIDs returns which of ids have an embedding from model.
func embeddedCardIDs(t *testing.T, db *gorm.DB, model string, ids ...uuid.UUID) []uuid.UUID {
	t.Helper()
	var rows []entity.CardEmbedding
	require.NoError(t, db.Where("card_id IN ? AND model = ?", ids, model).Find(&rows).Error)
	out := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		out[i] = r.CardID
	}
	return out
}

func TestEmbeddingService_EmbedsOnlyHostedCardsAndSkipsEmbeddedOnes(t *testing.T) {
	db := embeddingDB(t)
	set := newEmbedSet(t, db)
	model := "job-model-" + uuid.NewString()

	noImage := addCard(t, db, set, "001", "")
	done := addCard(t, db, set, "002", "cards/done")
	require.NoError(t, db.Create(&entity.CardEmbedding{CardID: done.ID, Model: model, Embedding: pgvector.NewVector(unit())}).Error)
	first := addCard(t, db, set, "003", "cards/first")
	second := addCard(t, db, set, "004", "cards/second")

	embedder, fetcher := newProviderMocks(t)
	fetcher.EXPECT().Fetch(mock.Anything, testImageBase+"/cards/first").Return([]byte("a"), nil).Once()
	fetcher.EXPECT().Fetch(mock.Anything, testImageBase+"/cards/second").Return([]byte("b"), nil).Once()
	embedder.EXPECT().Embed(mock.Anything, []byte("a")).Return(unit(), nil).Once()
	embedder.EXPECT().Embed(mock.Anything, []byte("b")).Return(unit(), nil).Once()

	summary, err := newEmbeddingJob(db, embedder, fetcher, model).EmbedCatalog(context.Background(), dto.EmbedCatalogRequest{Set: set.code})
	require.NoError(t, err)

	assert.Equal(t, dto.EmbedCatalogSummary{Total: 4, NoImage: 1, AlreadyEmbedded: 1, Embedded: 2}, withoutElapsed(summary))
	assert.ElementsMatch(t, []uuid.UUID{done.ID, first.ID, second.ID}, embeddedCardIDs(t, db, model, done.ID, first.ID, second.ID, noImage.ID))
}

func TestEmbeddingService_ContinuesAfterOneCardFails(t *testing.T) {
	db := embeddingDB(t)
	set := newEmbedSet(t, db)
	model := "job-model-" + uuid.NewString()

	broken := addCard(t, db, set, "001", "cards/broken")
	fine := addCard(t, db, set, "002", "cards/fine")

	embedder, fetcher := newProviderMocks(t)
	fetcher.EXPECT().Fetch(mock.Anything, testImageBase+"/cards/broken").Return(nil, errors.New("HTTP 404")).Once()
	fetcher.EXPECT().Fetch(mock.Anything, testImageBase+"/cards/fine").Return([]byte("ok"), nil).Once()
	embedder.EXPECT().Embed(mock.Anything, []byte("ok")).Return(unit(), nil).Once()

	summary, err := newEmbeddingJob(db, embedder, fetcher, model).EmbedCatalog(context.Background(), dto.EmbedCatalogRequest{Set: set.code})
	require.NoError(t, err)

	assert.Equal(t, 1, summary.Embedded)
	assert.Equal(t, 1, summary.Failed)
	assert.False(t, summary.Stopped)
	assert.Equal(t, []uuid.UUID{fine.ID}, embeddedCardIDs(t, db, model, broken.ID, fine.ID))
}

func TestEmbeddingService_ReembedsEveryCardWhenTheModelChanges(t *testing.T) {
	db := embeddingDB(t)
	set := newEmbedSet(t, db)
	oldModel := "job-old-" + uuid.NewString()
	newModel := "job-new-" + uuid.NewString()

	card := addCard(t, db, set, "001", "cards/model")

	embedder, fetcher := newProviderMocks(t)
	fetcher.EXPECT().Fetch(mock.Anything, mock.Anything).Return([]byte("img"), nil).Twice()
	embedder.EXPECT().Embed(mock.Anything, mock.Anything).Return(unit(), nil).Twice()

	_, err := newEmbeddingJob(db, embedder, fetcher, oldModel).EmbedCatalog(context.Background(), dto.EmbedCatalogRequest{Set: set.code})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{card.ID}, embeddedCardIDs(t, db, oldModel, card.ID))

	summary, err := newEmbeddingJob(db, embedder, fetcher, newModel).EmbedCatalog(context.Background(), dto.EmbedCatalogRequest{Set: set.code})
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Embedded, "a card embedded by another model is pending again")

	var rows []entity.CardEmbedding
	require.NoError(t, db.Where("card_id = ?", card.ID).Find(&rows).Error)
	require.Len(t, rows, 1, "the new embedding replaces the old one")
	assert.Equal(t, newModel, rows[0].Model)
}

func TestEmbeddingService_StopsCleanlyWhenTheDailyQuotaIsSpent(t *testing.T) {
	db := embeddingDB(t)
	set := newEmbedSet(t, db)
	model := "job-model-" + uuid.NewString()

	first := addCard(t, db, set, "001", "cards/one")
	second := addCard(t, db, set, "002", "cards/two")
	third := addCard(t, db, set, "003", "cards/three")

	embedder, fetcher := newProviderMocks(t)
	fetcher.EXPECT().Fetch(mock.Anything, mock.Anything).Return([]byte("img"), nil).Times(2)
	embedder.EXPECT().Embed(mock.Anything, mock.Anything).Return(unit(), nil).Once()
	embedder.EXPECT().Embed(mock.Anything, mock.Anything).Return(nil, embedding.ErrDailyQuotaExhausted).Once()

	summary, err := newEmbeddingJob(db, embedder, fetcher, model).EmbedCatalog(context.Background(), dto.EmbedCatalogRequest{Set: set.code})
	require.NoError(t, err, "a spent quota is a clean stop, not an error")

	assert.True(t, summary.Stopped)
	assert.Equal(t, 1, summary.Embedded)
	assert.Equal(t, 0, summary.Failed)
	assert.Equal(t, []uuid.UUID{first.ID}, embeddedCardIDs(t, db, model, first.ID, second.ID, third.ID), "the third card is never attempted")
}

// withoutElapsed zeroes the wall-clock field so summaries compare by value.
func withoutElapsed(s dto.EmbedCatalogSummary) dto.EmbedCatalogSummary {
	s.Elapsed = 0
	return s
}
