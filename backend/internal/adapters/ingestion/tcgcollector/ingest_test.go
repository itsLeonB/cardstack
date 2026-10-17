package tcgcollector

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// seedGame gives the test its own Game, so the Expansion Sets it sees are only
// the ones it seeds, whatever else the shared database holds.
func seedGame(t *testing.T, db *gorm.DB) entity.Game {
	t.Helper()

	game, err := crud.NewRepository[entity.Game](db).Insert(context.Background(), entity.Game{
		Slug: uniqueCode(t),
		Name: "test game",
	})
	require.NoError(t, err)
	return game
}

func seedLocale(t *testing.T, db *gorm.DB) entity.Locale {
	t.Helper()

	locale, err := crud.NewRepository[entity.Locale](db).Insert(context.Background(), entity.Locale{Code: uniqueCode(t)})
	require.NoError(t, err)
	return locale
}

func seedSet(t *testing.T, db *gorm.DB, gameID, localeID uuid.UUID, code string) entity.ExpansionSet {
	t.Helper()

	set, err := crud.NewRepository[entity.ExpansionSet](db).Insert(context.Background(), entity.ExpansionSet{
		GameID:   gameID,
		Code:     code,
		Name:     "seeded " + code,
		LocaleID: localeID,
	})
	require.NoError(t, err)
	return set
}

// seedCard adds one Card to a set; imageKey "" means it has no hosted image.
func seedCard(t *testing.T, db *gorm.DB, gameID, setID uuid.UUID, localID, imageKey string) {
	t.Helper()

	rarity, err := crud.NewRepository[entity.Rarity](db).Insert(context.Background(), entity.Rarity{
		GameID: gameID,
		Code:   uniqueCode(t),
		Name:   "test rarity",
	})
	require.NoError(t, err)

	_, err = crud.NewRepository[entity.Card](db).Insert(context.Background(), entity.Card{
		ExpansionSetID: setID,
		LocalID:        localID,
		Name:           "seeded card",
		Category:       "Pokémon",
		RarityID:       rarity.ID,
		Tags:           datatypes.JSONSlice[string]{},
		Attributes:     datatypes.JSONMap{},
		ImageKey:       imageKey,
	})
	require.NoError(t, err)
}

// fixturePage returns the saved sets page the fake PageSource serves.
func fixturePage(t *testing.T) []byte {
	t.Helper()

	page, err := os.ReadFile("testdata/sets-id.html")
	require.NoError(t, err)
	return page
}

func TestIngester_Run_ClassifiesSetsAndWritesNothing(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	game := seedGame(t, db)
	locale := seedLocale(t, db)
	seedSet(t, db, game.ID, locale.ID, "SM-P")
	seedSet(t, db, game.ID, locale.ID, "AS4D")
	unpublished := seedSet(t, db, game.ID, locale.ID, "SV-P")
	seedCard(t, db, game.ID, unpublished.ID, "001", "")
	seedCard(t, db, game.ID, unpublished.ID, "002", "hosted.webp")

	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(fixturePage(t), nil)
	lister := mocks.NewMockUnpublishedSetLister(t)
	lister.EXPECT().ListUnpublished(mock.Anything, game.ID).Return([]entity.ExpansionSet{unpublished}, nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	in.unpublished = lister

	report, err := in.Run(ctx, Options{})
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{"SM-P", "AS4D", "SV-P"}, listingCodes(report.Matched), "aliased codes match")
	assert.ElementsMatch(t, []string{"MA6", "M-P", "S8a-P", "S-P", "PGO"}, listingCodes(report.New), "unknown codes are new")

	byCode := make(map[string]SetListing, len(report.New))
	for _, set := range report.New {
		byCode[set.Code] = set
	}
	assert.Equal(t, "Evolusi Mega", byCode["MA6"].Series)
	assert.Equal(t, 207, byCode["MA6"].CardCount)
	assert.Empty(t, byCode["PGO"].Series, "unknown heading is reported as an empty Series")

	require.Len(t, report.Unpublished, 1)
	assert.Equal(t, UnpublishedSet{
		Code:              "SV-P",
		Name:              "seeded SV-P",
		CardCount:         2,
		CardsWithoutImage: 1,
		SeriesUnknown:     true,
	}, report.Unpublished[0])

	sets, err := crud.NewRepository[entity.ExpansionSet](db).FindAll(ctx, crud.Specification[entity.ExpansionSet]{
		Model: entity.ExpansionSet{GameID: game.ID},
	})
	require.NoError(t, err)
	assert.Len(t, sets, 3, "a dry run creates no Expansion Set")
}

func TestIngester_Run_SetFilterSelectsByEitherCode(t *testing.T) {
	db := testDB(t)

	game := seedGame(t, db)
	locale := seedLocale(t, db)
	seedSet(t, db, game.ID, locale.ID, "SM-P")

	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(fixturePage(t), nil)
	lister := mocks.NewMockUnpublishedSetLister(t)
	lister.EXPECT().ListUnpublished(mock.Anything, game.ID).Return(nil, nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	in.unpublished = lister

	report, err := in.Run(context.Background(), Options{SetCode: "SMP"})
	require.NoError(t, err)
	assert.Equal(t, []string{"SM-P"}, listingCodes(report.Matched))
	assert.Empty(t, report.New)

	_, err = in.Run(context.Background(), Options{SetCode: "NOPE"})
	assert.Error(t, err, "a code not on the page is an error, not an empty report")
}

func TestIngester_Run_UnseenGameListsEverySetAsNew(t *testing.T) {
	db := testDB(t)

	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(fixturePage(t), nil)

	// No EXPECT: the lister must not be asked about a Game that does not exist.
	lister := mocks.NewMockUnpublishedSetLister(t)

	in := NewIngester(db, pages)
	in.gameSlug = uniqueCode(t)
	in.unpublished = lister

	report, err := in.Run(context.Background(), Options{})
	require.NoError(t, err)
	assert.Empty(t, report.Matched)
	assert.Len(t, report.New, 8)
	assert.Empty(t, report.Unpublished)
}

func TestReport_Render_WritesPublishLine(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := Report{
		Unpublished: []UnpublishedSet{{Code: "SV-P", Name: "Scarlet & Violet Promos", CardCount: 287, CardsWithoutImage: 3}},
	}.Render(&out)
	require.NoError(t, err)

	assert.True(t, strings.Contains(out.String(), "UPDATE expansion_sets SET published = true WHERE code = 'SV-P';"), out.String())
}

func listingCodes(sets []SetListing) []string {
	codes := make([]string, 0, len(sets))
	for _, set := range sets {
		codes = append(codes, set.Code)
	}
	return codes
}
