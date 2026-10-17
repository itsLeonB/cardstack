package tcgcollector

import (
	"bytes"
	"context"
	"fmt"
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
// noCardLists answers the card list requests of the set tests: an empty page
// makes the card phase record a failure for each set, and write nothing.
func noCardLists(pages *mocks.MockPageSource) {
	notSetsPage := mock.MatchedBy(func(pageURL string) bool { return pageURL != setsPageURL })
	pages.EXPECT().Fetch(mock.Anything, notSetsPage).Return([]byte{}, nil).Maybe()
}

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
		CreatedBy: entity.SourcePokemonAsia,
		UpdatedBy: entity.SourcePokemonAsia,
		GameID:    gameID,
		Code:      code,
		Name:      "seeded " + code,
		LocaleID:  localeID,
	})
	require.NoError(t, err)
	return set
}

// seedUnpublishedSet inserts a set and then hides it. Create leaves Published's
// zero value to the column default (true), so the hide has to be an update.
func seedUnpublishedSet(t *testing.T, db *gorm.DB, gameID, localeID uuid.UUID, code string) entity.ExpansionSet {
	t.Helper()

	set := seedSet(t, db, gameID, localeID, code)
	require.NoError(t, db.Model(&set).Update("published", false).Error)
	set.Published = false
	return set
}

// seedSetUpdatedBy inserts a set that an Admin last updated, so no ingester may change it.
func seedSetUpdatedBy(t *testing.T, db *gorm.DB, gameID, localeID uuid.UUID, code string) entity.ExpansionSet {
	t.Helper()

	set, err := crud.NewRepository[entity.ExpansionSet](db).Insert(context.Background(), entity.ExpansionSet{
		CreatedBy: entity.SourcePokemonAsia,
		UpdatedBy: entity.SourceAdmin,
		GameID:    gameID,
		Code:      code,
		Name:      "seeded " + code,
		LocaleID:  localeID,
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
		CreatedBy:      entity.SourcePokemonAsia,
		UpdatedBy:      entity.SourcePokemonAsia,
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
	unpublished := seedUnpublishedSet(t, db, game.ID, locale.ID, "SV-P")
	seedCard(t, db, game.ID, unpublished.ID, "001", "")
	seedCard(t, db, game.ID, unpublished.ID, "002", "hosted.webp")

	pages := mocks.NewMockPageSource(t)
	noCardLists(pages)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(fixturePage(t), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug

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
	noCardLists(pages)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(fixturePage(t), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug

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
	noCardLists(pages)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(fixturePage(t), nil)

	in := NewIngester(db, pages)
	in.gameSlug = uniqueCode(t)

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

func TestReport_Render_MarksCreatedSetsAndFailures(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := Report{
		Unpublished: []UnpublishedSet{{Code: "MA6", Name: "Mega Test", Created: true}},
		Failed:      []string{"PGO: boom"},
	}.Render(&out)
	require.NoError(t, err)

	assert.Contains(t, out.String(), "created this run", out.String())
	assert.Contains(t, out.String(), "Failed to write: 1", out.String())
	assert.Contains(t, out.String(), "  PGO: boom", out.String())
}

func TestIngester_Apply_CreatesMissingSetsUnpublished(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	game := seedGame(t, db)
	// Every Series the sets page maps is in the catalog, so only PGO's heading is unknown.
	seriesByName := map[string]entity.Series{}
	for _, name := range []string{"Evolusi Mega", "Scarlet & Violet", "Pedang & Perisai", "Matahari & Bulan"} {
		series, err := crud.NewRepository[entity.Series](db).Insert(ctx, entity.Series{
			GameID: game.ID,
			Code:   uniqueCode(t),
			Name:   name,
		})
		require.NoError(t, err)
		seriesByName[name] = series
	}

	pages := mocks.NewMockPageSource(t)
	noCardLists(pages)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(fixturePage(t), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug

	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)
	assert.Empty(t, report.Failed)

	locale, err := crud.NewRepository[entity.Locale](db).FindFirst(ctx, crud.Specification[entity.Locale]{
		Model: entity.Locale{Code: localeCode},
	})
	require.NoError(t, err)
	require.False(t, locale.IsZero())

	sets := setsByCode(t, db, game.ID)
	require.Len(t, sets, len(report.New))
	for _, listing := range report.New {
		set, ok := sets[listing.Code]
		require.True(t, ok, listing.Code)
		assert.Equal(t, listing.Name, set.Name, listing.Code)
		assert.Equal(t, entity.SourceTCGCollector, set.CreatedBy, listing.Code)
		assert.Equal(t, entity.SourceTCGCollector, set.UpdatedBy, listing.Code)
		assert.False(t, set.Published, listing.Code)
		assert.Equal(t, locale.ID, set.LocaleID, listing.Code)
		assert.Equal(t, listing.LogoURL, set.SourceLogoURL, listing.Code)
		require.NotNil(t, set.ReleaseDate, listing.Code)
		assert.True(t, listing.ReleaseDate.Equal(*set.ReleaseDate), listing.Code)
	}
	evolusiMega := seriesByName["Evolusi Mega"].ID
	assert.Equal(t, &evolusiMega, sets["MA6"].SeriesID, "a heading the catalog knows gets its Series")
	assert.Nil(t, sets["PGO"].SeriesID, "a heading the catalog does not know leaves the Series empty")

	require.Len(t, report.Unpublished, len(report.New))
	for _, set := range report.Unpublished {
		assert.True(t, set.Created, set.Code)
		assert.Equal(t, set.Code == "PGO", set.SeriesUnknown, set.Code)
	}
}

func TestIngester_Apply_FillsEmptyLogosWithoutTouchingTheRow(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	game := seedGame(t, db)
	locale := seedLocale(t, db)
	seedSet(t, db, game.ID, locale.ID, "SM-P")
	seedSetUpdatedBy(t, db, game.ID, locale.ID, "AS4D")
	kept := seedSet(t, db, game.ID, locale.ID, "SV-P")
	require.NoError(t, db.Model(&kept).UpdateColumn("source_logo_url", "kept.webp").Error)
	before := setsByCode(t, db, game.ID)

	pages := mocks.NewMockPageSource(t)
	noCardLists(pages)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(
		pageSet{code: "SM-P", logo: "https://static.example/sm-p.webp"},
		pageSet{code: "AS4D", logo: "https://static.example/as4d.webp"},
		pageSet{code: "SV-P", logo: "https://static.example/sv-p.webp"},
	), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug

	_, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	after := setsByCode(t, db, game.ID)
	assert.Equal(t, "https://static.example/sm-p.webp", after["SM-P"].SourceLogoURL, "an Asia-created set with no logo gets one")
	assert.Equal(t, entity.SourcePokemonAsia, after["SM-P"].UpdatedBy)
	assert.Equal(t, "https://static.example/as4d.webp", after["AS4D"].SourceLogoURL, "an Admin-updated set with no logo gets one")
	assert.Equal(t, entity.SourceAdmin, after["AS4D"].UpdatedBy, "the logo fill does not take the row from its writer")
	assert.Equal(t, "kept.webp", after["SV-P"].SourceLogoURL, "a set that already has a logo is left alone")

	for code, set := range before {
		assert.Equal(t, set.Name, after[code].Name, code)
		assert.Equal(t, set.UpdatedBy, after[code].UpdatedBy, code)
		assert.Equal(t, set.UpdatedAt, after[code].UpdatedAt, "the logo fill leaves updated_at alone: "+code)
	}
}

func TestIngester_Apply_RerunChangesNothing(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	game := seedGame(t, db)
	pages := mocks.NewMockPageSource(t)
	noCardLists(pages)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(fixturePage(t), nil).Times(2)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug

	_, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)
	before := setsByCode(t, db, game.ID)

	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)
	assert.Empty(t, report.New, "a rerun finds every set already in the catalog")
	assert.Len(t, report.Matched, 8)
	assert.Empty(t, report.Failed)
	for _, set := range report.Unpublished {
		assert.False(t, set.Created, set.Code)
	}

	after := setsByCode(t, db, game.ID)
	require.Len(t, after, len(before))
	for code, set := range before {
		assert.Equal(t, set.UpdatedAt, after[code].UpdatedAt, code)
		assert.Equal(t, set.SourceLogoURL, after[code].SourceLogoURL, code)
	}
}

func TestIngester_Apply_SetCodeCreatesOnlyThatSet(t *testing.T) {
	db := testDB(t)

	game := seedGame(t, db)
	pages := mocks.NewMockPageSource(t)
	noCardLists(pages)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(fixturePage(t), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug

	report, err := in.Run(context.Background(), Options{SetCode: "MA6", Apply: true})
	require.NoError(t, err)
	assert.Equal(t, []string{"MA6"}, listingCodes(report.New))

	sets := setsByCode(t, db, game.ID)
	require.Len(t, sets, 1)
	assert.Contains(t, sets, "MA6")
}

func TestIngester_Apply_NeedsTheGame(t *testing.T) {
	db := testDB(t)

	pages := mocks.NewMockPageSource(t)
	noCardLists(pages)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(fixturePage(t), nil)

	in := NewIngester(db, pages)
	in.gameSlug = uniqueCode(t)

	_, err := in.Run(context.Background(), Options{Apply: true})
	assert.Error(t, err, "a set needs the Game that the Pokémon Asia ingester creates")
}

func listingCodes(sets []SetListing) []string {
	codes := make([]string, 0, len(sets))
	for _, set := range sets {
		codes = append(codes, set.Code)
	}
	return codes
}

// setsByCode returns the Game's Expansion Sets keyed by code.
func setsByCode(t *testing.T, db *gorm.DB, gameID uuid.UUID) map[string]entity.ExpansionSet {
	t.Helper()

	sets, err := crud.NewRepository[entity.ExpansionSet](db).FindAll(context.Background(), crud.Specification[entity.ExpansionSet]{
		Model: entity.ExpansionSet{GameID: gameID},
	})
	require.NoError(t, err)

	byCode := make(map[string]entity.ExpansionSet, len(sets))
	for _, set := range sets {
		byCode[set.Code] = set
	}
	return byCode
}

// pageSet is one set on a sets page built by setsPage. A logo of "" means the set has none.
type pageSet struct {
	code string
	logo string
}

// setsPage builds a sets page holding the given sets under one Series heading,
// with the markup parseSets reads.
func setsPage(sets ...pageSet) []byte {
	var b strings.Builder
	b.WriteString(`<div class="set-logo-grid"><h2 class="set-logo-grid-title">Mega Evolution Series</h2>`)
	for i, set := range sets {
		fmt.Fprintf(&b,
			`<div class="set-logo-grid-item" data-set-id="%d"><span class="set-logo-grid-item-code">%s</span>`+
				`<a class="set-logo-grid-item-name" href="/sets/%d/seeded?setCardCountMode=anyCardVariant">seeded %s</a>`+
				`<div class="set-logo-grid-item-release-date">Sep 16, 2026</div>`+
				`<div class="progress-label">0/10</div>`,
			i+1, set.code, i+1, set.code)
		if set.logo != "" {
			fmt.Fprintf(&b, `<img class="set-logo-grid-item-logo" srcset="%s 272w">`, set.logo)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	return []byte(b.String())
}
