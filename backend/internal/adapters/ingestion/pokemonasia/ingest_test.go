package pokemonasia

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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
	listing := expansionListing{Series: series.Name, Code: code, Name: "Original Name", ReleaseDate: mustParseDate(t, "01-02-2026"), ImageURL: "https://example.test/a.png"}

	first, err := in.upsertExpansionSet(ctx, game.ID, locale.ID, series.ID, listing)
	require.NoError(t, err)
	assert.Equal(t, "Original Name", first.Name)
	assert.Equal(t, "https://example.test/a.png", first.ImageURL)
	require.NotNil(t, first.ReleaseDate)
	assert.True(t, first.ReleaseDate.Equal(listing.ReleaseDate))
	require.NotNil(t, first.SeriesID)
	assert.Equal(t, series.ID, *first.SeriesID)

	listing.Name = "Updated Name"
	listing.ImageURL = "https://example.test/b.png"
	second, err := in.upsertExpansionSet(ctx, game.ID, locale.ID, series.ID, listing)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "must update the existing row, not create a duplicate")
	assert.Equal(t, "Updated Name", second.Name)
	assert.Equal(t, "https://example.test/b.png", second.ImageURL)

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

// TestIngester_Run_EndToEnd exercises the full Run() flow (enumerate with a
// single regulation=all pass -> sweep the rarity[] filter -> per-Series/Set
// -> card detail, with bounded concurrency) against a fake server, and
// confirms a second run is idempotent (no duplicate rows).
func TestIngester_Run_EndToEnd(t *testing.T) {
	setCode := "MA" + uniqueCode(t)[:8]
	seriesName := "Evolusi Mega"
	cardDetailID := "16488"
	rarityFilterID := "7"
	rarityCode := "SAR"

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
			q := r.URL.Query()
			switch {
			case q.Get("regulation") == "all" && q.Get("pageNo") == "1":
				// The enumeration pass's own response also carries the
				// rarity[] filter widget (real site behavior - see
				// mapper.go's parseRarityFilterOptions).
				_, _ = fmt.Fprintf(w, `<html><body>
					<div class="rarities"><div class="rarityOption">
						<input type="checkbox" name="rarity[]" value="%s"><label for="rarity_%s">%s</label>
					</div></div>
					<ul class="list"><li class="card"><a href="/id/card-search/detail/%s/"></a></li></ul>
					</body></html>`, rarityFilterID, rarityFilterID, rarityCode, cardDetailID)
				return
			case q.Get("rarity[]") == rarityFilterID && q.Get("pageNo") == "1":
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

	summary, err := in.Run(context.Background(), "", "")
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Sets)
	assert.Equal(t, 1, summary.Cards)
	assert.Equal(t, 1, summary.Rarities)

	// Re-running must be idempotent: same row counts, not duplicated.
	summary, err = in.Run(context.Background(), "", "")
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
	// "I" comes straight from pokemonDetailFixture's span.alpha (the
	// Regulation Mark), never from the rarity[] filter - proving rarity and
	// Regulation Mark are sourced independently (ADR-0010).
	assert.Equal(t, "I", cards[0].Attributes["regulationMark"])

	rarity, err := in.rarities.FindFirst(context.Background(), crud.Specification[entity.Rarity]{
		Model: entity.Rarity{GameID: game.ID, Code: rarityCode},
	})
	require.NoError(t, err)
	assert.Equal(t, cards[0].RarityID, rarity.ID, "rarity must resolve to the rarity[] filter's code, not the detail page")
}

// TestIngester_EnumerateExpansions_SetFilter confirms setFilter narrows
// enumeration to the one Expansion Set with that site code, independent of
// (and regardless of a mismatched) seriesFilter — see Run's doc comment on
// the -set CLI flag this backs.
func TestIngester_EnumerateExpansions_SetFilter(t *testing.T) {
	wantCode := "MA" + uniqueCode(t)[:8]
	otherCode := "SV" + uniqueCode(t)[:8]

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/card-search/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("pageNo") != "1" {
			w.Write([]byte(`<html><body></body></html>`)) //nolint:errcheck
			return
		}
		_, _ = fmt.Fprintf(w, `<html><body><ul class="expansionList">
			<li class="expansion"><a class="expansionLink" href="/id/card-search/list/?expansionCodes=%s">
				<div class="seriesBlock"><span class="series">Evolusi Mega</span></div>
				<h3 class="expansionTitle">Wanted Set</h3>
				<time class="relaseDate" datetime="01-15-2026"></time>
			</a></li>
			<li class="expansion"><a class="expansionLink" href="/id/card-search/list/?expansionCodes=%s">
				<div class="seriesBlock"><span class="series">Scarlet & Violet</span></div>
				<h3 class="expansionTitle">Other Set</h3>
				<time class="relaseDate" datetime="01-15-2026"></time>
			</a></li>
			</ul></body></html>`, wantCode, otherCode)
	}))
	t.Cleanup(server.Close)

	in := testIngester(t)
	in.client.baseURL = server.URL

	listings, err := in.enumerateExpansions(context.Background(), "", wantCode)
	require.NoError(t, err)
	require.Len(t, listings, 1)
	assert.Equal(t, wantCode, listings[0].Code)

	// A setFilter that doesn't match the given seriesFilter's Series yields
	// nothing - the two filters AND together rather than one overriding the
	// other.
	listings, err = in.enumerateExpansions(context.Background(), "Scarlet & Violet", wantCode)
	require.NoError(t, err)
	assert.Empty(t, listings)
}

func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(releaseDateLayout, s)
	require.NoError(t, err)
	return d
}

// TestIngester_EnumerateSetCardIDs_SinglePassRegulationAll confirms
// enumeration issues a single request with regulation=all (see ADR-0010)
// and never queries any of the old regulation=1/2/3 partition values - the
// ingester's own cross-bucket dedup logic for that partition is gone
// entirely, since there's only ever one pass to dedup against.
func TestIngester_EnumerateSetCardIDs_SinglePassRegulationAll(t *testing.T) {
	cardDetailID := "16488"
	var seenRegulations []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/card-search/list/":
			seenRegulations = append(seenRegulations, r.URL.Query().Get("regulation"))
			if r.URL.Query().Get("pageNo") == "1" {
				_, _ = fmt.Fprintf(w, `<html><body><ul class="list"><li class="card">
					<a href="/id/card-search/detail/%s/"></a></li></ul></body></html>`, cardDetailID)
				return
			}
			w.Write([]byte(`<html><body><ul class="list"></ul></body></html>`)) //nolint:errcheck
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	in := testIngester(t)
	in.client.baseURL = server.URL

	ids, _, err := in.enumerateSetCardIDs(context.Background(), "TEST"+uniqueCode(t)[:8])
	require.NoError(t, err)
	assert.Equal(t, []string{cardDetailID}, ids)
	require.NotEmpty(t, seenRegulations)
	for _, reg := range seenRegulations {
		assert.Equal(t, "all", reg, "must never query the old regulation=1/2/3 partition")
	}
}

// TestIngester_SweepRarities_ResolvesCodesPerCard confirms a card's rarity
// code is resolved by querying the results-list endpoint once per
// dynamically-parsed rarity[] filter id (including an unfamiliar/synthetic
// one never seen on the real site), not from anything on the detail page.
func TestIngester_SweepRarities_ResolvesCodesPerCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/card-search/list/":
			if r.URL.Query().Get("pageNo") != "1" {
				w.Write([]byte(`<html><body><ul class="list"></ul></body></html>`)) //nolint:errcheck
				return
			}
			switch r.URL.Query().Get("rarity[]") {
			case "1":
				_, _ = fmt.Fprint(w, `<html><body><ul class="list"><li class="card"><a href="/id/card-search/detail/100/"></a></li></ul></body></html>`)
			case "999":
				_, _ = fmt.Fprint(w, `<html><body><ul class="list"><li class="card"><a href="/id/card-search/detail/200/"></a></li></ul></body></html>`)
			default:
				w.Write([]byte(`<html><body><ul class="list"></ul></body></html>`)) //nolint:errcheck
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	in := testIngester(t)
	in.client.baseURL = server.URL

	byCard, incomplete, err := in.sweepRarities(context.Background(), "TEST", map[string]string{"1": "C", "999": "ZZZ"})
	require.NoError(t, err)
	assert.False(t, incomplete)
	assert.Equal(t, map[string]string{"100": "C", "200": "ZZZ"}, byCard)
}

// TestIngester_IngestSet_RarityPageFailure_DoesNotFalselyReportCardsMissing
// confirms that when a rarity page fetch exhausts its retries, the cards
// that sweep couldn't reach are recorded with a message pointing at the
// incomplete sweep, not the misleading "not found under any known rarity
// filter id" (which asserts a data-shape problem that isn't what happened).
func TestIngester_IngestSet_RarityPageFailure_DoesNotFalselyReportCardsMissing(t *testing.T) {
	unresolvedID := "16488"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/card-search/list/":
			if r.URL.Query().Get("regulation") == "all" && r.URL.Query().Get("pageNo") == "1" {
				_, _ = fmt.Fprintf(w, `<html><body>
					<div class="rarities"><div class="rarityOption"><input name="rarity[]" value="1"><label>C</label></div></div>
					<ul class="list"><li class="card"><a href="/id/card-search/detail/%s/"></a></li></ul>
					</body></html>`, unresolvedID)
				return
			}
			if r.URL.Query().Get("rarity[]") == "1" && r.URL.Query().Get("pageNo") == "1" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write([]byte(`<html><body><ul class="list"></ul></body></html>`)) //nolint:errcheck

		case "/card-search/detail/" + unresolvedID + "/":
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

	count, err := in.ingestSet(context.Background(), set.ID, "RARITYFAIL"+uniqueCode(t)[:8])
	require.NoError(t, err)
	assert.Equal(t, 0, count, "the card must not be ingested with an unresolved rarity")

	in.failuresMu.Lock()
	defer in.failuresMu.Unlock()
	var cardFailure *IngestFailure
	for i := range in.failures {
		if in.failures[i].CardID == unresolvedID {
			cardFailure = &in.failures[i]
		}
	}
	require.NotNil(t, cardFailure, "the unresolved card must still be recorded as a failure")
	assert.Contains(
		t, cardFailure.Err.Error(), "sweep",
		"the message must point at the incomplete sweep, not falsely claim the card is absent from every rarity bucket",
	)
}

// TestIngester_IngestSet_CardNotFoundUnderAnyRarity_LogsAndContinues
// confirms a card enumerated via regulation=all but not returned by any
// known rarity[] filter id is recorded as a failure and does not block
// ingestion of the rest of the set.
func TestIngester_IngestSet_CardNotFoundUnderAnyRarity_LogsAndContinues(t *testing.T) {
	resolvableID := "16488"
	unresolvableID := "99999"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/card-search/list/":
			q := r.URL.Query()
			switch {
			case q.Get("pageNo") != "1":
				w.Write([]byte(`<html><body><ul class="list"></ul></body></html>`)) //nolint:errcheck
			case q.Get("rarity[]") == "":
				// The single enumeration pass: both cards, plus the
				// rarity[] filter widget with just one known code.
				_, _ = fmt.Fprintf(w, `<html><body>
					<div class="rarities"><div class="rarityOption">
						<input type="checkbox" name="rarity[]" value="1"><label for="rarity_1">C</label>
					</div></div>
					<ul class="list">
						<li class="card"><a href="/id/card-search/detail/%s/"></a></li>
						<li class="card"><a href="/id/card-search/detail/%s/"></a></li>
					</ul>
					</body></html>`, resolvableID, unresolvableID)
			case q.Get("rarity[]") == "1":
				_, _ = fmt.Fprintf(w, `<html><body><ul class="list"><li class="card">
					<a href="/id/card-search/detail/%s/"></a></li></ul></body></html>`, resolvableID)
			default:
				w.Write([]byte(`<html><body><ul class="list"></ul></body></html>`)) //nolint:errcheck
			}
		case "/card-search/detail/" + resolvableID + "/":
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

	count, err := in.ingestSet(context.Background(), set.ID, "TEST"+uniqueCode(t)[:8])
	require.NoError(t, err)
	assert.Equal(t, 1, count, "the resolvable card must still be ingested despite the other one failing")

	require.Len(t, in.failures, 1)
	assert.Equal(t, unresolvableID, in.failures[0].CardID)
}

// TestIngester_IngestCard_CorrectsRarityReferenceOnRerun confirms
// re-ingesting the same card after its resolved rarity code changes (as a
// corrected re-run following ticket 11's fix would) updates the existing
// Card row's rarity reference in place, rather than creating a duplicate
// Card or Rarity row.
func TestIngester_IngestCard_CorrectsRarityReferenceOnRerun(t *testing.T) {
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(pokemonDetailFixture)) //nolint:errcheck
	}))
	t.Cleanup(server.Close)
	in.client.baseURL = server.URL

	staleCode := "I" // what the pre-fix bug would have wrongly resolved (the Regulation Mark)
	correctedCode := "SAR" + uniqueCode(t)[:6]

	require.NoError(t, in.ingestCard(ctx, set.ID, "16488", staleCode))
	first, err := in.cards.FindFirst(ctx, crud.Specification[entity.Card]{
		Model: entity.Card{ExpansionSetID: set.ID, LocalID: "001"},
	})
	require.NoError(t, err)
	require.False(t, first.IsZero())

	require.NoError(t, in.ingestCard(ctx, set.ID, "16488", correctedCode))
	second, err := in.cards.FindFirst(ctx, crud.Specification[entity.Card]{
		Model: entity.Card{ExpansionSetID: set.ID, LocalID: "001"},
	})
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "must update the existing Card row, not create a duplicate")
	assert.NotEqual(t, first.RarityID, second.RarityID, "rarity reference must be corrected to the newly resolved code")

	rarityRows, err := in.rarities.FindAll(ctx, crud.Specification[entity.Rarity]{
		Model: entity.Rarity{GameID: game.ID, Code: correctedCode},
	})
	require.NoError(t, err)
	assert.Len(t, rarityRows, 1, "must not create a duplicate Rarity row")
}

// TestStaleRegulationMarkCodes_IsRegulationMarkLetterRange pins
// CleanupStaleRarities' real candidate set to exactly the observed
// Regulation Mark letters that can never be a real Kelangkaan rarity code
// (A and C are excluded - see the var's own doc comment) - a regression
// guard against an accidental edit widening or narrowing what a live run is
// allowed to delete.
func TestStaleRegulationMarkCodes_IsRegulationMarkLetterRange(t *testing.T) {
	assert.Equal(t, []string{"B", "D", "E", "F", "G", "H", "I", "J"}, staleRegulationMarkCodes)
}

// TestIngester_CleanupRaritiesByCode_DeletesUnreferencedRow confirms an
// unreferenced candidate-coded rarity row (as the pre-fix bug would have
// left behind once a corrected re-ingestion moves its Cards onto the real
// rarity code) is deleted. Uses an isolated, uniquely-generated code rather
// than a real staleRegulationMarkCodes letter, since this package's shared,
// non-truncated test DB (see testdb_test.go) may already carry real ingested
// data referencing every one of those letters.
func TestIngester_CleanupRaritiesByCode_DeletesUnreferencedRow(t *testing.T) {
	in := testIngester(t)
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	in.gameID = game.ID

	code := "STALE" + uniqueCode(t)[:8]
	_, err = in.resolveRarityID(ctx, code)
	require.NoError(t, err)

	results, err := in.cleanupRaritiesByCode(ctx, []string{code})
	require.NoError(t, err)

	found := findCleanupResult(t, results, code)
	assert.True(t, found.Deleted)

	rows, err := in.rarities.FindAll(ctx, crud.Specification[entity.Rarity]{Model: entity.Rarity{GameID: game.ID, Code: code}})
	require.NoError(t, err)
	assert.Empty(t, rows, "the stale row must actually be deleted")
}

// TestIngester_CleanupRaritiesByCode_KeepsReferencedRow confirms a
// candidate-coded rarity row still referenced by a Card is reported, not
// deleted - deleting it would dangle that Card's rarity_id.
func TestIngester_CleanupRaritiesByCode_KeepsReferencedRow(t *testing.T) {
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

	code := "STALE" + uniqueCode(t)[:8]
	rarityID, err := in.resolveRarityID(ctx, code)
	require.NoError(t, err)
	card := mapCard(cardDetail{LocalID: uniqueCode(t)[:8], Category: categoryTrainer}, set.ID, rarityID, "", nil)
	_, err = in.cards.Insert(ctx, card)
	require.NoError(t, err)

	results, err := in.cleanupRaritiesByCode(ctx, []string{code})
	require.NoError(t, err)

	found := findCleanupResult(t, results, code)
	assert.False(t, found.Deleted)
	assert.GreaterOrEqual(t, found.CardCount, 1)

	rows, err := in.rarities.FindAll(ctx, crud.Specification[entity.Rarity]{Model: entity.Rarity{GameID: game.ID, Code: code}})
	require.NoError(t, err)
	require.Len(t, rows, 1, "the referenced row must not be deleted")
	assert.Equal(t, rarityID, rows[0].ID)
}

// TestIngester_CleanupStaleRarities_RunsAgainstRealCodes smoke-tests the
// exported entrypoint end-to-end against the real staleRegulationMarkCodes
// list; the delete-vs-report logic itself is covered in isolation by
// TestIngester_CleanupRaritiesByCode_* above.
func TestIngester_CleanupStaleRarities_RunsAgainstRealCodes(t *testing.T) {
	in := testIngester(t)

	results, err := in.CleanupStaleRarities(context.Background())
	require.NoError(t, err)
	for _, r := range results {
		assert.Contains(t, staleRegulationMarkCodes, r.Code)
	}
}

func findCleanupResult(t *testing.T, results []CleanupResult, code string) CleanupResult {
	t.Helper()
	for _, r := range results {
		if r.Code == code {
			return r
		}
	}
	t.Fatalf("no CleanupResult found for code %q", code)
	return CleanupResult{}
}

// TestIngester_SyncExpansionSets_ListingOnly proves the lightweight mode
// upserts Series/Expansion Sets (create-or-update, image URL included) and
// never requests anything under /card-search/list/ or /card-search/detail/.
func TestIngester_SyncExpansionSets_ListingOnly(t *testing.T) {
	setCode := "MA" + uniqueCode(t)[:8]
	imageURL := "https://asia.pokemon-card.com/id/products/MATL_PKG_IDN.png"

	var cardCrawlRequests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/card-search/" {
			cardCrawlRequests.Add(1)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("pageNo") != "1" {
			w.Write([]byte(`<html><body></body></html>`)) //nolint:errcheck
			return
		}
		_, _ = fmt.Fprintf(w, `<html><body><ul class="expansionList"><li class="expansion">
			<a class="expansionLink" href="/id/card-search/list/?expansionCodes=%s">
			<div class="leftColumn"><div class="imageContainer"><img src="%s"></div></div>
			<div class="seriesBlock"><span class="series">Evolusi Mega</span></div>
			<h3 class="expansionTitle">Test Set</h3>
			<time class="relaseDate" datetime="01-15-2026"></time>
			</a></li></ul></body></html>`, setCode, imageURL)
	}))
	t.Cleanup(server.Close)

	db := testDB(t)
	t.Cleanup(func() {
		db.Where("code = ?", setCode).Delete(&entity.ExpansionSet{}) //nolint:errcheck
	})

	in := NewIngester(db)
	in.client.limiter = rate.NewLimiter(rate.Inf, 0)
	in.client.baseURL = server.URL

	// Run twice: first creates the set (not in the DB yet), second updates it.
	for range 2 {
		summary, err := in.SyncExpansionSets(context.Background(), "", "")
		require.NoError(t, err)
		assert.Equal(t, 1, summary.Sets)
		assert.Zero(t, summary.Cards)
		assert.Zero(t, summary.Rarities)
	}
	assert.Zero(t, cardCrawlRequests.Load(), "lightweight sync must not fetch any card-search list/detail page")

	set, err := in.sets.FindFirst(context.Background(), crud.Specification[entity.ExpansionSet]{
		Model: entity.ExpansionSet{GameID: in.gameID, Code: setCode},
	})
	require.NoError(t, err)
	require.False(t, set.IsZero())
	assert.Equal(t, imageURL, set.ImageURL)
}
