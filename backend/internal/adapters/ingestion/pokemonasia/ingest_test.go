package pokemonasia

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func testIngester(t *testing.T) *Ingester {
	t.Helper()
	in := NewIngester(testDB(t))
	in.client.limiter = rate.NewLimiter(rate.Inf, 0)
	return in
}

func TestIngester_UpsertGame_Idempotent(t *testing.T) {
	in := testIngester(t)
	ctx := context.Background()

	first, err := in.upsertGame(ctx)
	require.NoError(t, err)
	assert.Equal(t, gameSlug, first.Slug)

	second, err := in.upsertGame(ctx)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "second call must find the existing row, not create a duplicate")

	rows, err := in.games.FindAll(ctx, crud.Specification[entity.Game]{Model: entity.Game{Slug: gameSlug}})
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestIngester_UpsertSeries_IdempotentAndUpdates(t *testing.T) {
	in := testIngester(t)
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)

	name := "Test Series " + uniqueCode(t)

	first, err := in.upsertSeries(ctx, game.ID, name)
	require.NoError(t, err)
	assert.Equal(t, name, first.Name)
	assert.Equal(t, slugifySeries(name), first.Code)

	second, err := in.upsertSeries(ctx, game.ID, name)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "second call must find the existing row, not create a duplicate")

	rows, err := in.series.FindAll(ctx, crud.Specification[entity.Series]{
		Model: entity.Series{GameID: game.ID, Code: slugifySeries(name)},
	})
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestIngester_Series_UniqueConstraint(t *testing.T) {
	in := testIngester(t)
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	code := uniqueCode(t)

	_, err = in.series.Insert(ctx, entity.Series{GameID: game.ID, Code: code, Name: "A"})
	require.NoError(t, err)

	_, err = in.series.Insert(ctx, entity.Series{GameID: game.ID, Code: code, Name: "B"})
	assert.Error(t, err, "duplicate (game_id, code) must be rejected by the unique index")
}

func TestIngester_UpsertExpansionSet_IdempotentAndPopulatesReleaseDate(t *testing.T) {
	in := testIngester(t)
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	locale, err := in.upsertLocale(ctx, "id")
	require.NoError(t, err)
	series, err := in.upsertSeries(ctx, game.ID, "Test Series "+uniqueCode(t))
	require.NoError(t, err)

	code := uniqueCode(t)
	listing := expansionListing{Series: series.Name, Code: code, Name: "Original Name", ReleaseDate: mustParseDate(t, "01-02-2026")}

	first, err := in.upsertExpansionSet(ctx, game.ID, locale.ID, series.ID, listing)
	require.NoError(t, err)
	assert.Equal(t, "Original Name", first.Name)
	require.NotNil(t, first.ReleaseDate)
	assert.True(t, first.ReleaseDate.Equal(listing.ReleaseDate))
	require.NotNil(t, first.SeriesID)
	assert.Equal(t, series.ID, *first.SeriesID)

	listing.Name = "Updated Name"
	second, err := in.upsertExpansionSet(ctx, game.ID, locale.ID, series.ID, listing)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "must update the existing row, not create a duplicate")
	assert.Equal(t, "Updated Name", second.Name)

	rows, err := in.sets.FindAll(ctx, crud.Specification[entity.ExpansionSet]{
		Model: entity.ExpansionSet{GameID: game.ID, Code: code},
	})
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestIngester_ExpansionSet_UniqueConstraint(t *testing.T) {
	in := testIngester(t)
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	locale, err := in.upsertLocale(ctx, "id")
	require.NoError(t, err)
	code := uniqueCode(t)

	_, err = in.sets.Insert(ctx, entity.ExpansionSet{GameID: game.ID, Code: code, Name: "A", LocaleID: locale.ID})
	require.NoError(t, err)

	_, err = in.sets.Insert(ctx, entity.ExpansionSet{GameID: game.ID, Code: code, Name: "B", LocaleID: locale.ID})
	assert.Error(t, err, "duplicate (game_id, code) must be rejected by the unique index")
}

// TestIngester_ResolveRarityID_FindOrCreatesMidRun confirms a new rarity
// code is find-or-created the first time it's seen, not pre-seeded (see the
// plan's "Rarity is a first-class, per-Game lookup table" decision).
func TestIngester_ResolveRarityID_FindOrCreatesMidRun(t *testing.T) {
	in := testIngester(t)
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	in.gameID = game.ID

	code := "Z" + uniqueCode(t)[:8]

	existing, err := in.rarities.FindFirst(ctx, crud.Specification[entity.Rarity]{
		Model: entity.Rarity{GameID: game.ID, Code: code},
	})
	require.NoError(t, err)
	require.True(t, existing.IsZero(), "rarity must not be seeded before first use")

	first, err := in.resolveRarityID(ctx, code)
	require.NoError(t, err)

	second, err := in.resolveRarityID(ctx, code)
	require.NoError(t, err)
	assert.Equal(t, first, second, "second call must return the cached/existing row's ID, not create a duplicate")

	rows, err := in.rarities.FindAll(ctx, crud.Specification[entity.Rarity]{
		Model: entity.Rarity{GameID: game.ID, Code: code},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, code, rows[0].Name, "Name defaults to the raw code when the source exposes no readable name")
}

func TestIngester_Rarity_UniqueConstraint(t *testing.T) {
	in := testIngester(t)
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	code := uniqueCode(t)

	_, err = in.rarities.Insert(ctx, entity.Rarity{GameID: game.ID, Code: code, Name: code})
	require.NoError(t, err)

	_, err = in.rarities.Insert(ctx, entity.Rarity{GameID: game.ID, Code: code, Name: code})
	assert.Error(t, err, "duplicate (game_id, code) must be rejected by the unique index")
}

func TestIngester_UpsertCard_IdempotentAndUpdates(t *testing.T) {
	in := testIngester(t)
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	in.gameID = game.ID
	locale, err := in.upsertLocale(ctx, "id")
	require.NoError(t, err)
	series, err := in.upsertSeries(ctx, game.ID, "Test Series "+uniqueCode(t))
	require.NoError(t, err)
	set, err := in.upsertExpansionSet(ctx, game.ID, locale.ID, series.ID, expansionListing{
		Code: uniqueCode(t), Name: "Set", ReleaseDate: mustParseDate(t, "01-01-2026"),
	})
	require.NoError(t, err)

	rarityID, err := in.resolveRarityID(ctx, "C")
	require.NoError(t, err)

	localID := "001"
	card := mapCard(cardDetail{LocalID: localID, Name: "First", Category: categoryTrainer}, set.ID, rarityID, "img1", []byte("raw1"))

	first, err := in.upsertCard(ctx, card)
	require.NoError(t, err)
	assert.Equal(t, "First", first.Name)

	updated := mapCard(cardDetail{LocalID: localID, Name: "Second", Category: categoryTrainer}, set.ID, rarityID, "img2", []byte("raw2"))
	second, err := in.upsertCard(ctx, updated)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "must update the existing row, not create a duplicate")
	assert.Equal(t, "Second", second.Name)
	assert.Equal(t, "img2", second.ImageURL)

	rows, err := in.cards.FindAll(ctx, crud.Specification[entity.Card]{
		Model: entity.Card{ExpansionSetID: set.ID, LocalID: localID},
	})
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestIngester_Card_UniqueConstraint(t *testing.T) {
	in := testIngester(t)
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	in.gameID = game.ID
	locale, err := in.upsertLocale(ctx, "id")
	require.NoError(t, err)
	series, err := in.upsertSeries(ctx, game.ID, "Test Series "+uniqueCode(t))
	require.NoError(t, err)
	set, err := in.upsertExpansionSet(ctx, game.ID, locale.ID, series.ID, expansionListing{
		Code: uniqueCode(t), Name: "Set", ReleaseDate: mustParseDate(t, "01-01-2026"),
	})
	require.NoError(t, err)
	rarityID, err := in.resolveRarityID(ctx, "C")
	require.NoError(t, err)

	localID := "001"
	_, err = in.cards.Insert(ctx, mapCard(cardDetail{LocalID: localID, Category: categoryTrainer}, set.ID, rarityID, "", nil))
	require.NoError(t, err)

	_, err = in.cards.Insert(ctx, mapCard(cardDetail{LocalID: localID, Category: categoryTrainer}, set.ID, rarityID, "", nil))
	assert.Error(t, err, "duplicate (expansion_set_id, local_id) must be rejected by the unique index")
}

// TestIngester_Run_EndToEnd exercises the full Run() flow (enumerate ->
// per-Series/Set -> per-regulation results pages -> card detail, with
// bounded concurrency) against a fake server, and confirms a second run is
// idempotent (no duplicate rows).
func TestIngester_Run_EndToEnd(t *testing.T) {
	setCode := "MA" + uniqueCode(t)[:8]
	seriesName := "Evolusi Mega"
	cardDetailID := "16488"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/card-search/":
			if r.URL.Query().Get("pageNo") == "1" {
				_, _ = fmt.Fprintf(w, `<html><body><ul class="expansionList"><li class="expansion">
					<a class="expansionLink" href="/id/card-search/list/?expansionCodes=%s">
					<div class="seriesBlock"><span class="series">%s</span></div>
					<h3 class="expansionTitle">Test Set</h3>
					<time class="relaseDate" datetime="01-15-2026"></time>
					</a></li></ul></body></html>`, setCode, seriesName)
				return
			}
			w.Write([]byte(`<html><body></body></html>`)) //nolint:errcheck

		case "/card-search/list/":
			if r.URL.Query().Get("regulation") == "1" && r.URL.Query().Get("pageNo") == "1" {
				_, _ = fmt.Fprintf(w, `<html><body><ul class="list"><li class="card">
					<a href="/id/card-search/detail/%s/"></a></li></ul></body></html>`, cardDetailID)
				return
			}
			w.Write([]byte(`<html><body><ul class="list"></ul></body></html>`)) //nolint:errcheck

		case "/card-search/detail/" + cardDetailID + "/":
			w.Write([]byte(pokemonDetailFixture)) //nolint:errcheck

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	db := testDB(t)
	// seriesName is deliberately one of the real target Series (Run's own
	// enumeration filter only keeps those) — clean up the test-created
	// ExpansionSet (and its Cards, via ON DELETE CASCADE) so repeated test
	// runs don't accumulate fake sets under the real series in this shared
	// DB (see testdb_test.go's doc comment on why it isn't truncated).
	t.Cleanup(func() {
		db.Where("code = ?", setCode).Delete(&entity.ExpansionSet{}) //nolint:errcheck
	})

	in := NewIngester(db)
	in.client.limiter = rate.NewLimiter(rate.Inf, 0)
	in.client.baseURL = server.URL

	summary, err := in.Run(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Sets)
	assert.Equal(t, 1, summary.Cards)
	assert.Equal(t, 1, summary.Rarities)

	// Re-running must be idempotent: same row counts, not duplicated.
	summary, err = in.Run(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Sets)
	assert.Equal(t, 1, summary.Cards)

	game, err := in.upsertGame(context.Background())
	require.NoError(t, err)
	set, err := in.sets.FindFirst(context.Background(), crud.Specification[entity.ExpansionSet]{
		Model: entity.ExpansionSet{GameID: game.ID, Code: setCode},
	})
	require.NoError(t, err)
	require.False(t, set.IsZero())
	require.NotNil(t, set.ReleaseDate)
	require.NotNil(t, set.SeriesID)

	cards, err := in.cards.FindAll(context.Background(), crud.Specification[entity.Card]{
		Model: entity.Card{ExpansionSetID: set.ID},
	})
	require.NoError(t, err)
	require.Len(t, cards, 1)
	assert.Equal(t, "Mega Venusaur ex", cards[0].Name)
	assert.Equal(t, "001", cards[0].LocalID)
	assert.NotEqual(t, uuid.Nil, cards[0].RarityID)
	assert.Equal(t, "Standar", cards[0].Attributes["regulation"])
}

func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(releaseDateLayout, s)
	require.NoError(t, err)
	return d
}

// TestIngester_IngestSet_DedupesAcrossRegulationBuckets confirms a card id
// returned by more than one regulation bucket (confirmed live: the site's
// regulation query param doesn't reliably partition results) is ingested
// exactly once, tagged with the lowest-numbered bucket it appeared in.
func TestIngester_IngestSet_DedupesAcrossRegulationBuckets(t *testing.T) {
	cardDetailID := "16488"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/card-search/list/":
			// regulation=1 and regulation=2 both return the same card
			// (mirrors the live overlap); regulation=3 is empty.
			if reg := r.URL.Query().Get("regulation"); (reg == "1" || reg == "2") && r.URL.Query().Get("pageNo") == "1" {
				_, _ = fmt.Fprintf(w, `<html><body><ul class="list"><li class="card">
					<a href="/id/card-search/detail/%s/"></a></li></ul></body></html>`, cardDetailID)
				return
			}
			w.Write([]byte(`<html><body><ul class="list"></ul></body></html>`)) //nolint:errcheck

		case "/card-search/detail/" + cardDetailID + "/":
			w.Write([]byte(pokemonDetailFixture)) //nolint:errcheck

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	in := testIngester(t)
	in.client.baseURL = server.URL
	game, err := in.upsertGame(context.Background())
	require.NoError(t, err)
	in.gameID = game.ID
	locale, err := in.upsertLocale(context.Background(), "id")
	require.NoError(t, err)
	series, err := in.upsertSeries(context.Background(), game.ID, "Test Series "+uniqueCode(t))
	require.NoError(t, err)
	set, err := in.upsertExpansionSet(context.Background(), game.ID, locale.ID, series.ID, expansionListing{
		Code: uniqueCode(t), Name: "Set", ReleaseDate: mustParseDate(t, "01-01-2026"),
	})
	require.NoError(t, err)

	count, err := in.ingestSet(context.Background(), set.ID, "DEDUPE"+uniqueCode(t)[:8])
	require.NoError(t, err)
	assert.Equal(t, 1, count, "the overlapping card must be ingested exactly once, not once per bucket")

	cards, err := in.cards.FindAll(context.Background(), crud.Specification[entity.Card]{
		Model: entity.Card{ExpansionSetID: set.ID},
	})
	require.NoError(t, err)
	require.Len(t, cards, 1)
	assert.Equal(t, "Standar", cards[0].Attributes["regulation"], "must keep the lowest-numbered bucket it appeared in")
}
