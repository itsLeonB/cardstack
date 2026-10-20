package tcgcollector

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// The card list fixture's set is the first set of setsPage, and its card
// pages are the fixtures named below.
const (
	cardListURL  = "https://www.tcgcollector.com/sets/1/seeded"
	oddishURL    = "https://www.tcgcollector.com/cards/71277/oddish-shiny-treasure-ex-001-190"
	basicFireURL = "https://www.tcgcollector.com/cards/71900/basic-fire-energy-shiny-treasure-ex-191-190"
	victoryURL   = "https://www.tcgcollector.com/cards/71901/victory-symbol-shiny-treasure-ex-192-190"
)

// seedCardGame gives a test its own Game with the Rarities the card fixtures
// name: RR, U, and the Tanpa Tanda that a card page's "—" maps to.
func seedCardGame(t *testing.T, db *gorm.DB) entity.Game {
	t.Helper()

	game := seedGame(t, db)
	seedRarity(t, db, game.ID, "RR", "Double Rare")
	seedRarity(t, db, game.ID, "U", "Uncommon")
	seedRarity(t, db, game.ID, uniqueCode(t), tanpaTanda)
	return game
}

func seedRarity(t *testing.T, db *gorm.DB, gameID uuid.UUID, code, name string) {
	t.Helper()

	_, err := crud.NewRepository[entity.Rarity](db).Insert(context.Background(), entity.Rarity{
		GameID: gameID,
		Code:   code,
		Name:   name,
	})
	require.NoError(t, err)
}

// expectNewSetFetches answers the fetches of a dry run or apply that finds
// the set absent: the sets page, the card list and all three card pages.
func expectNewSetFetches(t *testing.T, pages *mocks.MockPageSource, victoryPage string) {
	t.Helper()

	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(pageSet{code: "SM-P"}), nil)
	pages.EXPECT().Fetch(mock.Anything, cardListURL).Return(fixtureBytes(t, "card-list-shiny-treasure.html"), nil)
	pages.EXPECT().Fetch(mock.Anything, oddishURL).Return(fixtureBytes(t, "card-pokemon-mega-venusaur.html"), nil)
	pages.EXPECT().Fetch(mock.Anything, basicFireURL).Return(fixtureBytes(t, "card-energy-basic-fire.html"), nil)
	pages.EXPECT().Fetch(mock.Anything, victoryURL).Return(fixtureBytes(t, victoryPage), nil)
}

// cardsOfSet reads a set's Cards by local ID, with their Rarity loaded.
func cardsOfSet(t *testing.T, db *gorm.DB, setID uuid.UUID) map[string]entity.Card {
	t.Helper()

	cards, err := crud.NewRepository[entity.Card](db).FindAll(context.Background(), crud.Specification[entity.Card]{
		Model:            entity.Card{ExpansionSetID: setID},
		PreloadRelations: []string{"Rarity"},
	})
	require.NoError(t, err)

	byID := map[string]entity.Card{}
	for _, card := range cards {
		byID[card.LocalID] = card
	}
	return byID
}

func TestIngester_Apply_AddsTheCardsOfANewSetAndKeepsItUnpublished(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedCardGame(t, db)

	pages := mocks.NewMockPageSource(t)
	expectNewSetFetches(t, pages, "card-trainer-caitlin.html")

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	require.Len(t, report.Cards, 1)
	cards := report.Cards[0]
	assert.Equal(t, []string{"001 Oddish", "191 Basic Fire Energy", "192 Victory Symbol"}, cards.Added,
		"a new set takes Basic Energy and the Victory Symbol like any card, and pads numbers to 3")
	assert.Empty(t, cards.Skipped)
	assert.Empty(t, cards.Failed)
	assert.False(t, cards.Published, "the new set is not published, so its cards are not yet visible")

	set := setsByCode(t, db, game.ID)["SM-P"]
	assert.False(t, set.Published)
	added := cardsOfSet(t, db, set.ID)
	require.Len(t, added, 3)

	oddish := added["001"]
	assert.Equal(t, "Oddish", oddish.Name)
	assert.Equal(t, categoryPokemon, oddish.Category)
	assert.Equal(t, "RR", oddish.Rarity.Code)
	assert.Equal(t, "5ban Graphics", oddish.Illustrator)
	assert.Equal(t, "I", oddish.Attributes["regulationMark"])
	assert.Equal(t, "https://static.tcgcollector.com/content/images/70/50/45/7050451f2b790a0e571d4f2b049a58d765d43a90ff8c08d3bfe8c3f3d2df25c7.webp", oddish.SourceImageURL)
	assert.Equal(t, entity.SourceTCGCollector, oddish.CreatedBy)
	assert.Equal(t, entity.SourceTCGCollector, oddish.UpdatedBy)
	assert.True(t, strings.HasPrefix(oddish.Raw, "<main"), "raw is the card page's main element")

	basicFire := added["191"]
	assert.Equal(t, categoryEnergi, basicFire.Category)
	assert.Equal(t, []string{"Energi Dasar"}, []string(basicFire.Tags))
	assert.Equal(t, tanpaTanda, basicFire.Rarity.Name, `a page's "—" is the Tanpa Tanda rarity`)

	victory := added["192"]
	assert.Equal(t, categoryTrainer, victory.Category)
	assert.Empty(t, victory.SourceImageURL, "a card with no image in the list stores none")
	assert.Equal(t, 1, cards.WithoutImage)
}

func TestIngester_Apply_RerunAddsNothingAndFetchesNoCardPages(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedCardGame(t, db)

	first := mocks.NewMockPageSource(t)
	expectNewSetFetches(t, first, "card-trainer-caitlin.html")
	in := NewIngester(db, first)
	in.gameSlug = game.Slug
	_, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	// The second run has no card page expectations: fetching one fails the test.
	second := mocks.NewMockPageSource(t)
	second.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(pageSet{code: "SM-P"}), nil)
	second.EXPECT().Fetch(mock.Anything, cardListURL).Return(fixtureBytes(t, "card-list-shiny-treasure.html"), nil)
	in = NewIngester(db, second)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	cards := report.Cards[0]
	assert.Empty(t, cards.Added)
	assert.Equal(t, 3, cards.Held)
	assert.Empty(t, cards.Failed)

	set := setsByCode(t, db, game.ID)["SM-P"]
	assert.Len(t, cardsOfSet(t, db, set.ID), 3, "no card is inserted twice")
}

func TestIngester_DryRun_ListsTheCardsItWouldAddAndWritesNothing(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedCardGame(t, db)

	pages := mocks.NewMockPageSource(t)
	expectNewSetFetches(t, pages, "card-trainer-caitlin.html")

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{})
	require.NoError(t, err)

	require.Len(t, report.Cards, 1)
	assert.Equal(t, []string{"001 Oddish", "191 Basic Fire Energy", "192 Victory Symbol"}, report.Cards[0].Added)

	assert.Empty(t, setsByCode(t, db, game.ID), "a dry run creates no set")
}

func TestIngester_Apply_SkipsBasicEnergyAndVictorySymbolInAnExistingSet(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedCardGame(t, db)
	locale := seedLocale(t, db)
	set := seedSet(t, db, game.ID, locale.ID, "SM-P")
	// "1" holds Oddish's number as an integer, and the set code as an ID is
	// what makes its Victory Symbol a duplicate.
	seedCard(t, db, game.ID, set.ID, "1", "")
	seedCard(t, db, game.ID, set.ID, "SM-P", "")

	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(pageSet{code: "SM-P"}), nil)
	pages.EXPECT().Fetch(mock.Anything, cardListURL).Return(fixtureBytes(t, "card-list-shiny-treasure.html"), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	cards := report.Cards[0]
	assert.Equal(t, 1, cards.Held, "Oddish 001 matches the existing 1 by integer value")
	assert.Empty(t, cards.Added)
	assert.Equal(t, []string{
		"Basic Fire Energy: basic energy in a set that exists",
		"Victory Symbol: the set already has a card with ID SM-P",
	}, cards.Skipped)
	assert.Len(t, cardsOfSet(t, db, set.ID), 2, "nothing is inserted")
}

func TestIngester_Apply_AddsIntoAPublishedSetWithItsOwnNumberWidth(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedCardGame(t, db)
	locale := seedLocale(t, db)
	set := seedSet(t, db, game.ID, locale.ID, "SM-P")
	seedCard(t, db, game.ID, set.ID, "1", "")

	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(pageSet{code: "SM-P"}), nil)
	pages.EXPECT().Fetch(mock.Anything, cardListURL).Return(fixtureBytes(t, "card-list-shiny-treasure.html"), nil)
	pages.EXPECT().Fetch(mock.Anything, victoryURL).Return(fixtureBytes(t, "card-trainer-caitlin.html"), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	cards := report.Cards[0]
	assert.True(t, cards.Published)
	assert.Equal(t, []string{"192 Victory Symbol"}, cards.Added, "the existing 1 makes the width 1, so no padding")
	assert.Equal(t, []string{"Basic Fire Energy: basic energy in a set that exists"}, cards.Skipped)

	added := cardsOfSet(t, db, set.ID)
	require.Contains(t, added, "192")
	assert.Equal(t, categoryTrainer, added["192"].Category)

	var out bytes.Buffer
	require.NoError(t, report.Render(&out))
	assert.Contains(t, out.String(), "added to a published set: 1")
}

func raritiesOf(t *testing.T, db *gorm.DB, gameID uuid.UUID) []entity.Rarity {
	t.Helper()
	rows, err := crud.NewRepository[entity.Rarity](db).FindAll(context.Background(), crud.Specification[entity.Rarity]{
		Model: entity.Rarity{GameID: gameID},
	})
	require.NoError(t, err)
	return rows
}

func TestIngester_Apply_CreatesTheRarityOfACardWhoseCodeIsNew(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedGame(t, db)
	seedRarity(t, db, game.ID, "RR", "Double Rare")
	seedRarity(t, db, game.ID, uniqueCode(t), tanpaTanda)
	// No U: the Victory Symbol's page is Uncommon, so its code is new.

	pages := mocks.NewMockPageSource(t)
	expectNewSetFetches(t, pages, "card-trainer-caitlin.html")

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	cards := report.Cards[0]
	assert.Equal(t, []string{"001 Oddish", "191 Basic Fire Energy", "192 Victory Symbol"}, cards.Added)
	assert.Empty(t, cards.Failed)

	set := setsByCode(t, db, game.ID)["SM-P"]
	byID := cardsOfSet(t, db, set.ID)
	require.Len(t, byID, 3)
	assert.Equal(t, "U", byID["192"].Rarity.Code)
	assert.Equal(t, "U", byID["192"].Rarity.Name, "a new rarity is named by its code, as the Asia ingester does")
	assert.Len(t, raritiesOf(t, db, game.ID), 3)
}

func TestIngester_DryRun_CreatesNoRarity(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedGame(t, db)
	seedRarity(t, db, game.ID, "RR", "Double Rare")
	seedRarity(t, db, game.ID, uniqueCode(t), tanpaTanda)

	pages := mocks.NewMockPageSource(t)
	expectNewSetFetches(t, pages, "card-trainer-caitlin.html")

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{})
	require.NoError(t, err)

	assert.Contains(t, report.Cards[0].Added, "192 Victory Symbol")
	assert.Empty(t, report.Cards[0].Failed)
	assert.Len(t, raritiesOf(t, db, game.ID), 2)
}

// seedTCGCollectorSet inserts an unpublished set that TCG Collector created,
// as a run that stopped partway would leave it.
func seedTCGCollectorSet(t *testing.T, db *gorm.DB, gameID, localeID uuid.UUID, code string) entity.ExpansionSet {
	t.Helper()

	set, err := crud.NewRepository[entity.ExpansionSet](db).Insert(context.Background(), entity.ExpansionSet{
		CreatedBy: entity.SourceTCGCollector,
		UpdatedBy: entity.SourceTCGCollector,
		GameID:    gameID,
		Code:      code,
		Name:      "seeded " + code,
		LocaleID:  localeID,
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(&set).Update("published", false).Error)
	set.Published = false
	return set
}

func TestIngester_Apply_RerunAddsTheBasicEnergyAndVictorySymbolOfASetItCreated(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedCardGame(t, db)
	locale := seedLocale(t, db)
	set := seedTCGCollectorSet(t, db, game.ID, locale.ID, "SM-P")
	// The first run wrote Oddish and then stopped, so the set is only partly filled.
	seedCard(t, db, game.ID, set.ID, "001", "")

	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(pageSet{code: "SM-P"}), nil)
	pages.EXPECT().Fetch(mock.Anything, cardListURL).Return(fixtureBytes(t, "card-list-shiny-treasure.html"), nil)
	pages.EXPECT().Fetch(mock.Anything, basicFireURL).Return(fixtureBytes(t, "card-energy-basic-fire.html"), nil)
	pages.EXPECT().Fetch(mock.Anything, victoryURL).Return(fixtureBytes(t, "card-trainer-caitlin.html"), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	cards := report.Cards[0]
	assert.Equal(t, []string{"191 Basic Fire Energy", "192 Victory Symbol"}, cards.Added,
		"a set TCG Collector created is new to the run, so its Basic Energy and Victory Symbol are not skipped")
	assert.Empty(t, cards.Skipped)
	assert.Equal(t, 1, cards.Held)
	assert.Len(t, cardsOfSet(t, db, set.ID), 3)
}

func TestIngester_Apply_RerunOfAPublishedSetTheRunCreatedReportsItPublished(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedCardGame(t, db)
	locale := seedLocale(t, db)
	set := seedTCGCollectorSet(t, db, game.ID, locale.ID, "SM-P")
	require.NoError(t, db.Model(&set).Update("published", true).Error)
	seedCard(t, db, game.ID, set.ID, "001", "")

	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(pageSet{code: "SM-P"}), nil)
	pages.EXPECT().Fetch(mock.Anything, cardListURL).Return(fixtureBytes(t, "card-list-shiny-treasure.html"), nil)
	pages.EXPECT().Fetch(mock.Anything, basicFireURL).Return(fixtureBytes(t, "card-energy-basic-fire.html"), nil)
	pages.EXPECT().Fetch(mock.Anything, victoryURL).Return(fixtureBytes(t, "card-trainer-caitlin.html"), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	cards := report.Cards[0]
	assert.True(t, cards.Published, "the set is published in the catalog, so its Cards are visible at once")
	assert.Equal(t, []string{"191 Basic Fire Energy", "192 Victory Symbol"}, cards.Added)
	assert.Len(t, cardsOfSet(t, db, set.ID), 3)
}

func TestIngester_Apply_BasicEnergyWhoseNumberIsHeldIsSkippedNotHeld(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedCardGame(t, db)
	locale := seedLocale(t, db)
	set := seedSet(t, db, game.ID, locale.ID, "SM-P")
	seedCard(t, db, game.ID, set.ID, "1", "")
	seedCard(t, db, game.ID, set.ID, "191", "")
	seedCard(t, db, game.ID, set.ID, "SM-P", "")

	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(pageSet{code: "SM-P"}), nil)
	pages.EXPECT().Fetch(mock.Anything, cardListURL).Return(fixtureBytes(t, "card-list-shiny-treasure.html"), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	cards := report.Cards[0]
	assert.Equal(t, 1, cards.Held, "only Oddish's 1 is held; the 191 is a skip, not a hold")
	assert.Equal(t, []string{
		"Basic Fire Energy: basic energy in a set that exists",
		"Victory Symbol: the set already has a card with ID SM-P",
	}, cards.Skipped)
}

// A byte that is not UTF-8 is one Postgres refuses in text, so Victory Symbol's
// insert fails after the set's other Cards are written in the same transaction.
// That rolls back the set and its Cards, and the other set is still written.
func TestIngester_Apply_RollsBackASetWhoseCardInsertFails(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedCardGame(t, db)

	brokenList := bytes.ReplaceAll(fixtureBytes(t, "card-list-shiny-treasure.html"), []byte("Victory Symbol"), []byte("Victory\xffSymbol"))
	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(pageSet{code: "SM-P"}, pageSet{code: "PGO"}), nil)
	pages.EXPECT().Fetch(mock.Anything, "https://www.tcgcollector.com/sets/1/seeded").Return(brokenList, nil)
	pages.EXPECT().Fetch(mock.Anything, "https://www.tcgcollector.com/sets/2/seeded").Return(fixtureBytes(t, "card-list-shiny-treasure.html"), nil)
	pages.EXPECT().Fetch(mock.Anything, oddishURL).Return(fixtureBytes(t, "card-pokemon-mega-venusaur.html"), nil)
	pages.EXPECT().Fetch(mock.Anything, basicFireURL).Return(fixtureBytes(t, "card-energy-basic-fire.html"), nil)
	pages.EXPECT().Fetch(mock.Anything, victoryURL).Return(fixtureBytes(t, "card-trainer-caitlin.html"), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	require.Len(t, report.Failed, 1)
	assert.Contains(t, report.Failed[0], "SM-P:")
	assert.Empty(t, report.Cards[0].Added, "the failed set reports none of its Cards as written")

	sets := setsByCode(t, db, game.ID)
	assert.NotContains(t, sets, "SM-P", "the set goes back with the Cards it wrote")
	require.Contains(t, sets, "PGO")
	assert.Len(t, cardsOfSet(t, db, sets["PGO"].ID), 3)
}

// mewRGBURL is the page of Mew, whose printed number "R/RGB" has no digits.
const mewRGBURL = "https://www.tcgcollector.com/cards/72001/mew-ultra-rare-r-rgb"

func TestIngester_Apply_AddsACardWhoseNumberHasNoDigitsUnderItsWholeNumber(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedCardGame(t, db)

	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(pageSet{code: "SM-P"}), nil)
	pages.EXPECT().Fetch(mock.Anything, cardListURL).Return(fixtureBytes(t, "card-list-mew-rgb.html"), nil)
	pages.EXPECT().Fetch(mock.Anything, mewRGBURL).Return(fixtureBytes(t, "card-pokemon-mega-venusaur.html"), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	cards := report.Cards[0]
	assert.Equal(t, []string{"R/RGB Mew"}, cards.Added)
	assert.Empty(t, cards.Failed)

	set := setsByCode(t, db, game.ID)["SM-P"]
	byID := cardsOfSet(t, db, set.ID)
	require.Len(t, byID, 1)
	assert.Equal(t, "RR", byID["R/RGB"].Rarity.Code, "the rarity comes from the card page")
}

func TestIngester_DryRun_ListsACardWhoseNumberHasNoDigitsAndWritesNothing(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedCardGame(t, db)

	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(pageSet{code: "SM-P"}), nil)
	pages.EXPECT().Fetch(mock.Anything, cardListURL).Return(fixtureBytes(t, "card-list-mew-rgb.html"), nil)
	pages.EXPECT().Fetch(mock.Anything, mewRGBURL).Return(fixtureBytes(t, "card-pokemon-mega-venusaur.html"), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{})
	require.NoError(t, err)

	require.Len(t, report.Cards, 1)
	assert.Equal(t, []string{"R/RGB Mew"}, report.Cards[0].Added)
	assert.Empty(t, report.Cards[0].Failed)

	assert.Empty(t, setsByCode(t, db, game.ID), "a dry run creates no set")
}

func TestIngester_Apply_RerunHoldsACardWhoseNumberHasNoDigitsAndFetchesNoCardPage(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedCardGame(t, db)

	first := mocks.NewMockPageSource(t)
	first.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(pageSet{code: "SM-P"}), nil)
	first.EXPECT().Fetch(mock.Anything, cardListURL).Return(fixtureBytes(t, "card-list-mew-rgb.html"), nil)
	first.EXPECT().Fetch(mock.Anything, mewRGBURL).Return(fixtureBytes(t, "card-pokemon-mega-venusaur.html"), nil)
	in := NewIngester(db, first)
	in.gameSlug = game.Slug
	_, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	// The second run has no card page expectation: fetching one fails the test.
	second := mocks.NewMockPageSource(t)
	second.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(pageSet{code: "SM-P"}), nil)
	second.EXPECT().Fetch(mock.Anything, cardListURL).Return(fixtureBytes(t, "card-list-mew-rgb.html"), nil)
	in = NewIngester(db, second)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	cards := report.Cards[0]
	assert.Empty(t, cards.Added)
	assert.Equal(t, 1, cards.Held)
	assert.Empty(t, cards.Failed)

	set := setsByCode(t, db, game.ID)["SM-P"]
	assert.Len(t, cardsOfSet(t, db, set.ID), 1, "the card is not inserted twice")
}

func TestIngester_Apply_HoldsACardWhoseNumberHasNoDigitsWhenTheSetHoldsIt(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedCardGame(t, db)
	locale := seedLocale(t, db)
	set := seedSet(t, db, game.ID, locale.ID, "SM-P")
	seedCard(t, db, game.ID, set.ID, "R/RGB", "")

	// No card page expectation: a held number must not fetch its page.
	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(pageSet{code: "SM-P"}), nil)
	pages.EXPECT().Fetch(mock.Anything, cardListURL).Return(fixtureBytes(t, "card-list-mew-rgb.html"), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	cards := report.Cards[0]
	assert.Empty(t, cards.Added)
	assert.Equal(t, 1, cards.Held)
	assert.Empty(t, cards.Failed)
}

func TestIngester_Apply_FailsASecondCardWhoseTextNumberRepeatsInTheList(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	game := seedCardGame(t, db)

	// The second entry is failed before its page is fetched, so only the first has a page expectation.
	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return(setsPage(pageSet{code: "SM-P"}), nil)
	pages.EXPECT().Fetch(mock.Anything, cardListURL).Return(fixtureBytes(t, "card-list-mew-rgb-twice.html"), nil)
	pages.EXPECT().Fetch(mock.Anything, mewRGBURL).Return(fixtureBytes(t, "card-pokemon-mega-venusaur.html"), nil)

	in := NewIngester(db, pages)
	in.gameSlug = game.Slug
	report, err := in.Run(ctx, Options{Apply: true})
	require.NoError(t, err)

	cards := report.Cards[0]
	assert.Equal(t, []string{"R/RGB Mew"}, cards.Added)
	assert.Equal(t, []string{`Mew: number "R/RGB" is listed more than once in this card list`}, cards.Failed)

	set := setsByCode(t, db, game.ID)["SM-P"]
	assert.Len(t, cardsOfSet(t, db, set.ID), 1)
}
