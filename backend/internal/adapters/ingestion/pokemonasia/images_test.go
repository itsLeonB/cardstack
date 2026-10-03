package pokemonasia

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

// pngBytes starts with the real PNG signature, which is all content sniffing
// looks at.
var pngBytes = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...)

const sourceImagePath = "/id/card-img/id00016488.png"

// sourceHostTransport serves every image request from handler without any
// network, whatever host the request names, and records the hosts it saw. The
// ingester's own host allow-list is what decides whether a request is made at
// all, so tests assert on seenHosts rather than on server addresses.
type sourceHostTransport struct {
	handler http.Handler

	mu        sync.Mutex
	seenHosts []string
}

func (s *sourceHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	s.seenHosts = append(s.seenHosts, req.URL.Host)
	s.mu.Unlock()
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}

func (s *sourceHostTransport) hosts() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.seenHosts...)
}

// useSourceImages routes the ingester's image downloads to handler.
func useSourceImages(in *Ingester, handler http.HandlerFunc) *sourceHostTransport {
	rt := &sourceHostTransport{handler: handler}
	in.client.imageClient.Transport = rt
	return rt
}

func servePNG(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(pngBytes)
}

func TestClient_FetchImage_ReturnsBytesAndSniffedType(t *testing.T) {
	c := newClient()
	c.limiter = rate.NewLimiter(rate.Inf, 0)
	c.imageClient.Transport = &sourceHostTransport{handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream") // the declared type is not trusted
		_, _ = w.Write(pngBytes)
	})}

	body, contentType, err := c.fetchImage(context.Background(), "https://asia.pokemon-card.com"+sourceImagePath)

	require.NoError(t, err)
	assert.Equal(t, pngBytes, body)
	assert.Equal(t, "image/png", contentType)
}

func TestClient_FetchImage_ResolvesRelativeAddressOnSourceHost(t *testing.T) {
	c := newClient()
	c.limiter = rate.NewLimiter(rate.Inf, 0)
	rt := &sourceHostTransport{handler: http.HandlerFunc(servePNG)}
	c.imageClient.Transport = rt

	_, _, err := c.fetchImage(context.Background(), "/id/products/cover.png")

	require.NoError(t, err)
	assert.Equal(t, []string{"asia.pokemon-card.com"}, rt.hosts())
}

func TestClient_FetchImage_RejectsBeforeRequesting(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"other host", "https://evil.example.test/x.png"},
		{"lookalike host", "https://asia.pokemon-card.com.evil.example.test/x.png"},
		{"userinfo trick", "https://asia.pokemon-card.com@evil.example.test/x.png"},
		{"other scheme", "ftp://asia.pokemon-card.com/x.png"},
		{"plain http downgrade", "http://asia.pokemon-card.com/x.png"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newClient()
			c.limiter = rate.NewLimiter(rate.Inf, 0)
			rt := &sourceHostTransport{handler: http.HandlerFunc(servePNG)}
			c.imageClient.Transport = rt

			_, _, err := c.fetchImage(context.Background(), tt.url)

			assert.Error(t, err)
			assert.Empty(t, rt.hosts(), "a rejected address must never be requested")
		})
	}
}

func TestClient_FetchImage_RefusesRedirectOffSourceHost(t *testing.T) {
	c := newClient()
	c.limiter = rate.NewLimiter(rate.Inf, 0)
	rt := &sourceHostTransport{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.example.test/x.png", http.StatusFound)
	})}
	c.imageClient.Transport = rt

	_, _, err := c.fetchImage(context.Background(), "https://asia.pokemon-card.com"+sourceImagePath)

	assert.Error(t, err)
	assert.Equal(t, []string{"asia.pokemon-card.com"}, rt.hosts(), "the redirect target must not be requested")
}

func TestClient_FetchImage_RejectsBadResponses(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"non-200", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }},
		{"html declared as image", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("<html><body>not an image</body></html>"))
		}},
		{"svg", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "image/svg+xml")
			_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`))
		}},
		{"empty body", func(http.ResponseWriter, *http.Request) {}},
		{"over the size limit", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, maxImageBytes)...))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newClient()
			c.limiter = rate.NewLimiter(rate.Inf, 0)
			c.imageClient.Transport = &sourceHostTransport{handler: tt.handler}

			_, _, err := c.fetchImage(context.Background(), "https://asia.pokemon-card.com"+sourceImagePath)

			assert.Error(t, err)
		})
	}
}

func TestClient_FetchImage_RejectsDeclaredLengthOverLimit(t *testing.T) {
	c := newClient()
	c.limiter = rate.NewLimiter(rate.Inf, 0)
	c.imageClient.Transport = &sourceHostTransport{handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(maxImageBytes+1))
		w.WriteHeader(http.StatusOK)
	})}

	_, _, err := c.fetchImage(context.Background(), "https://asia.pokemon-card.com"+sourceImagePath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit")
}

func TestImageKeys_AreDeterministicAndNamespaced(t *testing.T) {
	id := uuid.New()

	assert.Equal(t, "cards/"+id.String(), cardImageKey(id))
	assert.Equal(t, "expansion-sets/"+id.String(), expansionSetImageKey(id))
	assert.Equal(t, cardImageKey(id), cardImageKey(id))
	assert.NotEqual(t, cardImageKey(id), expansionSetImageKey(id), "cards and Expansion Sets must not share a key namespace")
}

// hostingFixture is one Expansion Set row to attach cards to, with an
// ingester wired to a mock object store and a served source host.
type hostingFixture struct {
	in    *Ingester
	store *mocks.MockObjectStore
	set   entity.ExpansionSet
}

func newHostingFixture(t *testing.T) hostingFixture {
	t.Helper()
	store := mocks.NewMockObjectStore(t)
	in := testIngester(t)
	in.store = store
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

	detail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(pokemonDetailFixture))
	}))
	t.Cleanup(detail.Close)
	in.client.baseURL = detail.URL

	return hostingFixture{in: in, store: store, set: set}
}

func (f hostingFixture) card(t *testing.T) entity.Card {
	t.Helper()
	card, err := f.in.cards.FindFirst(context.Background(), crud.Specification[entity.Card]{
		Model: entity.Card{ExpansionSetID: f.set.ID, LocalID: "001"},
	})
	require.NoError(t, err)
	require.False(t, card.IsZero())
	return card
}

func TestIngester_IngestCard_HostsOriginalUnderDeterministicKey(t *testing.T) {
	f := newHostingFixture(t)
	rt := useSourceImages(f.in, servePNG)
	f.store.EXPECT().Put(mock.Anything, mock.MatchedBy(func(key string) bool { return strings.HasPrefix(key, "cards/") }), "image/png", pngBytes).Return(nil).Once()

	require.NoError(t, f.in.ingestCard(context.Background(), f.set.ID, f.set.Code, "16488", "SAR"))

	card := f.card(t)
	assert.Equal(t, "cards/"+card.ID.String(), card.ImageKey)
	assert.Equal(t, cardImageURL("16488"), card.SourceImageURL, "the scraped address is kept alongside the key")
	assert.Equal(t, []string{"asia.pokemon-card.com"}, rt.hosts())
	assert.Equal(t, int64(1), f.in.imagesHosted.Load())
}

func TestIngester_IngestCard_SkipsImageAlreadyHosted(t *testing.T) {
	f := newHostingFixture(t)
	rt := useSourceImages(f.in, servePNG)
	f.store.EXPECT().Put(mock.Anything, mock.Anything, "image/png", pngBytes).Return(nil).Once() // the first run only

	require.NoError(t, f.in.ingestCard(context.Background(), f.set.ID, f.set.Code, "16488", "SAR"))
	firstKey := f.card(t).ImageKey
	require.NotEmpty(t, firstKey)

	require.NoError(t, f.in.ingestCard(context.Background(), f.set.ID, f.set.Code, "16488", "SAR"))

	assert.Equal(t, firstKey, f.card(t).ImageKey)
	assert.Len(t, rt.hosts(), 1, "a hosted image is not downloaded again")
}

func TestIngester_IngestCard_DownloadFailureLeavesKeyEmptyAndRetriesNextRun(t *testing.T) {
	f := newHostingFixture(t)
	healthy := false
	useSourceImages(f.in, func(w http.ResponseWriter, r *http.Request) {
		if !healthy {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		servePNG(w, r)
	})

	require.NoError(t, f.in.ingestCard(context.Background(), f.set.ID, f.set.Code, "16488", "SAR"), "a failed image must not fail the card")
	assert.Empty(t, f.card(t).ImageKey)
	require.Len(t, f.in.failures, 1)
	assert.Equal(t, "hosting card image", f.in.failures[0].Stage)

	healthy = true
	f.store.EXPECT().Put(mock.Anything, mock.Anything, "image/png", pngBytes).Return(nil).Once()
	require.NoError(t, f.in.ingestCard(context.Background(), f.set.ID, f.set.Code, "16488", "SAR"))
	assert.NotEmpty(t, f.card(t).ImageKey, "the next run heals it")
}

func TestIngester_IngestCard_UploadFailureLeavesKeyEmpty(t *testing.T) {
	f := newHostingFixture(t)
	useSourceImages(f.in, servePNG)
	f.store.EXPECT().Put(mock.Anything, mock.Anything, "image/png", pngBytes).Return(fmt.Errorf("bucket unavailable")).Once()

	require.NoError(t, f.in.ingestCard(context.Background(), f.set.ID, f.set.Code, "16488", "SAR"))

	assert.Empty(t, f.card(t).ImageKey)
	require.Len(t, f.in.failures, 1)
}

func TestIngester_IngestCard_NotAnImageIsRejectedBeforeUpload(t *testing.T) {
	f := newHostingFixture(t)
	useSourceImages(f.in, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("<html>blocked</html>"))
	})
	// no Put expectation: the mock fails the test on any upload

	require.NoError(t, f.in.ingestCard(context.Background(), f.set.ID, f.set.Code, "16488", "SAR"))

	assert.Empty(t, f.card(t).ImageKey)
}

func TestIngester_IngestCard_WithoutObjectStoreHostsNothing(t *testing.T) {
	f := newHostingFixture(t)
	f.in.store = nil
	rt := useSourceImages(f.in, servePNG)

	require.NoError(t, f.in.ingestCard(context.Background(), f.set.ID, f.set.Code, "16488", "SAR"))

	assert.Empty(t, f.card(t).ImageKey)
	assert.Empty(t, rt.hosts(), "no credentials means no image download either")
	assert.Empty(t, f.in.failures, "skipping is not a failure")
}

func TestIngester_UpsertCard_PreservesImageKeyWhenSourceChanges(t *testing.T) {
	f := newHostingFixture(t)
	ctx := context.Background()
	rarityID, err := f.in.resolveRarityID(ctx, "C")
	require.NoError(t, err)

	first := mapCard(cardDetail{LocalID: "001", Name: "A", Category: categoryTrainer}, f.set.ID, rarityID, "https://source.test/a.png", nil)
	saved, err := f.in.upsertCard(ctx, first)
	require.NoError(t, err)
	saved.ImageKey = cardImageKey(saved.ID)
	_, err = f.in.cards.Update(ctx, saved)
	require.NoError(t, err)

	second := mapCard(cardDetail{LocalID: "001", Name: "A", Category: categoryTrainer}, f.set.ID, rarityID, "https://source.test/b.png", nil)
	updated, err := f.in.upsertCard(ctx, second)
	require.NoError(t, err)

	assert.Equal(t, saved.ImageKey, updated.ImageKey, "re-scraping must not clear a hosted key")
	assert.Equal(t, "https://source.test/b.png", updated.SourceImageURL)
}

func TestIngester_HostSetCover_HostsOnceUnderSetNamespace(t *testing.T) {
	f := newHostingFixture(t)
	rt := useSourceImages(f.in, servePNG)
	f.store.EXPECT().Put(mock.Anything, "expansion-sets/"+f.set.ID.String(), "image/png", pngBytes).Return(nil).Once()
	f.set.SourceImageURL = "https://asia.pokemon-card.com/id/products/cover.png"

	f.in.hostSetCover(context.Background(), f.set, "CODE")
	hosted, err := f.in.sets.FindFirst(context.Background(), crud.Specification[entity.ExpansionSet]{Model: entity.ExpansionSet{GameID: f.in.gameID, Code: f.set.Code}})
	require.NoError(t, err)
	assert.Equal(t, "expansion-sets/"+f.set.ID.String(), hosted.ImageKey)

	f.in.hostSetCover(context.Background(), hosted, "CODE") // skip-if-hosted: Put is expected only once
	assert.Len(t, rt.hosts(), 1)
}

const coverURL = "https://asia.pokemon-card.com/id/products/cover.png"

// newCrawlServer serves one Expansion Set (with a cover) holding one card, and
// counts the requests that reach the card-search list and detail pages.
func newCrawlServer(t *testing.T, setCode string) (server *httptest.Server, cardRequests *atomic.Int64) {
	t.Helper()
	cardRequests = &atomic.Int64{}
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case r.URL.Path == "/card-search/" && q.Get("pageNo") == "1":
			_, _ = fmt.Fprintf(w, `<html><body><ul class="expansionList"><li class="expansion">
				<a class="expansionLink" href="/id/card-search/list/?expansionCodes=%s">
				<div class="leftColumn"><div class="imageContainer"><img src="%s"></div></div>
				<div class="seriesBlock"><span class="series">Evolusi Mega</span></div>
				<h3 class="expansionTitle">Test Set</h3>
				<time class="relaseDate" datetime="01-15-2026"></time>
				</a></li></ul></body></html>`, setCode, coverURL)
		case r.URL.Path == "/card-search/":
			_, _ = w.Write([]byte(`<html><body></body></html>`))
		case r.URL.Path == "/card-search/list/":
			cardRequests.Add(1)
			if q.Get("pageNo") != "1" {
				_, _ = w.Write([]byte(`<html><body><ul class="list"></ul></body></html>`))
				return
			}
			_, _ = w.Write([]byte(`<html><body>
				<div class="rarities"><div class="rarityOption"><input type="checkbox" name="rarity[]" value="7"><label for="rarity_7">SAR</label></div></div>
				<ul class="list"><li class="card"><a href="/id/card-search/detail/16488/"></a></li></ul></body></html>`))
		case r.URL.Path == "/card-search/detail/16488/":
			cardRequests.Add(1)
			_, _ = w.Write([]byte(pokemonDetailFixture))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server, cardRequests
}

func newCrawlIngester(t *testing.T, server *httptest.Server, setCode string, store objectstore.ObjectStore) *Ingester {
	t.Helper()
	db := testDB(t)
	// Evolusi Mega is a real target Series; remove the fake set (cards cascade)
	// so the shared test DB does not accumulate it.
	t.Cleanup(func() { db.Where("code = ?", setCode).Delete(&entity.ExpansionSet{}) }) //nolint:errcheck
	in := NewIngester(db, store)
	in.client.limiter = rate.NewLimiter(rate.Inf, 0)
	in.client.baseURL = server.URL
	return in
}

func TestIngester_Run_HostsSetCoverAndCardImagesOnceAcrossReruns(t *testing.T) {
	setCode := "MA" + uniqueCode(t)[:8]
	server, _ := newCrawlServer(t, setCode)
	store := mocks.NewMockObjectStore(t)
	in := newCrawlIngester(t, server, setCode, store)
	rt := useSourceImages(in, servePNG)
	store.EXPECT().Put(mock.Anything, mock.MatchedBy(func(k string) bool { return strings.HasPrefix(k, "expansion-sets/") }), "image/png", pngBytes).Return(nil).Once()
	store.EXPECT().Put(mock.Anything, mock.MatchedBy(func(k string) bool { return strings.HasPrefix(k, "cards/") }), "image/png", pngBytes).Return(nil).Once()

	first, err := in.Run(context.Background(), "", setCode)
	require.NoError(t, err)
	assert.Equal(t, 2, first.ImagesHosted)
	assert.Empty(t, first.Failures)

	second, err := in.Run(context.Background(), "", setCode)
	require.NoError(t, err)
	assert.Equal(t, 2, second.ImagesHosted, "the counter is cumulative for this ingester")
	assert.Len(t, rt.hosts(), 2, "the re-run downloads and uploads nothing: every key is already hosted")

	game, err := in.upsertGame(context.Background())
	require.NoError(t, err)
	set, err := in.sets.FindFirst(context.Background(), crud.Specification[entity.ExpansionSet]{Model: entity.ExpansionSet{GameID: game.ID, Code: setCode}})
	require.NoError(t, err)
	assert.Equal(t, "expansion-sets/"+set.ID.String(), set.ImageKey)
	assert.Equal(t, coverURL, set.SourceImageURL)
	cards, err := in.cards.FindAll(context.Background(), crud.Specification[entity.Card]{Model: entity.Card{ExpansionSetID: set.ID}})
	require.NoError(t, err)
	require.Len(t, cards, 1)
	assert.Equal(t, "cards/"+cards[0].ID.String(), cards[0].ImageKey)
}

func TestIngester_SyncExpansionSets_BackfillsCoverWithoutCrawlingCards(t *testing.T) {
	setCode := "MA" + uniqueCode(t)[:8]
	server, cardRequests := newCrawlServer(t, setCode)
	store := mocks.NewMockObjectStore(t)
	in := newCrawlIngester(t, server, setCode, store)
	useSourceImages(in, servePNG)
	store.EXPECT().Put(mock.Anything, mock.MatchedBy(func(k string) bool { return strings.HasPrefix(k, "expansion-sets/") }), "image/png", pngBytes).Return(nil).Once()

	summary, err := in.SyncExpansionSets(context.Background(), "", setCode)

	require.NoError(t, err)
	assert.Equal(t, 1, summary.ImagesHosted)
	assert.Zero(t, summary.Cards)
	assert.Zero(t, cardRequests.Load(), "backfilling covers must not crawl any card page")
}

func TestIngester_Run_FailedCoverDoesNotAbortAndCardsStillIngest(t *testing.T) {
	setCode := "MA" + uniqueCode(t)[:8]
	server, _ := newCrawlServer(t, setCode)
	store := mocks.NewMockObjectStore(t)
	in := newCrawlIngester(t, server, setCode, store)
	useSourceImages(in, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) })

	summary, err := in.Run(context.Background(), "", setCode)

	require.NoError(t, err)
	assert.Equal(t, 1, summary.Cards, "a failed image never stops the card from being ingested")
	assert.Zero(t, summary.ImagesHosted)
	stages := []string{}
	for _, f := range summary.Failures {
		stages = append(stages, f.Stage)
	}
	assert.ElementsMatch(t, []string{"hosting expansion set cover", "hosting card image"}, stages)
}

func TestIngester_Run_WithoutObjectStoreSkipsHostingAndStillIngests(t *testing.T) {
	setCode := "MA" + uniqueCode(t)[:8]
	server, _ := newCrawlServer(t, setCode)
	in := newCrawlIngester(t, server, setCode, nil)
	rt := useSourceImages(in, servePNG)

	summary, err := in.Run(context.Background(), "", setCode)

	require.NoError(t, err)
	assert.Equal(t, 1, summary.Cards)
	assert.Zero(t, summary.ImagesHosted)
	assert.Empty(t, summary.Failures)
	assert.Empty(t, rt.hosts(), "with no R2 configured no image is downloaded")
}

// newHTTPImageClient returns a client whose image allow-list is the given
// httptest server, reached over real HTTP with the production redirect policy.
// Only tests can do this: production always uses the https source host.
func newHTTPImageClient(t *testing.T, source *httptest.Server) *client {
	t.Helper()
	c := newClient()
	c.limiter = rate.NewLimiter(rate.Inf, 0)
	c.baseURL = source.URL
	c.imageHost = strings.TrimPrefix(source.URL, "http://")
	c.imageScheme = "http"
	return c
}

func TestClient_FetchImage_OverRealHTTP(t *testing.T) {
	var offHostHits atomic.Int64
	offHost := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offHostHits.Add(1)
		servePNG(w, r)
	}))
	t.Cleanup(offHost.Close)

	mux := http.NewServeMux()
	mux.HandleFunc("/ok.png", servePNG)
	mux.HandleFunc("/to-off-host", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, offHost.URL+"/x.png", http.StatusFound)
	})
	mux.HandleFunc("/to-ok", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/ok.png", http.StatusFound) })
	mux.HandleFunc("/loop", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/loop", http.StatusFound) })
	mux.HandleFunc("/streamed-too-big", func(w http.ResponseWriter, _ *http.Request) {
		// No Content-Length (chunked): only the streaming cap can stop this.
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("\x89PNG\r\n\x1a\n"))
		chunk := bytes.Repeat([]byte{0}, 1<<20)
		for range maxImageBytes/len(chunk) + 1 {
			if _, err := w.Write(chunk); err != nil {
				return
			}
			w.(http.Flusher).Flush()
		}
	})
	mux.HandleFunc("/declared-too-big", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(maxImageBytes+1))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("\x89PNG")) // far short of the declared length
	})
	source := httptest.NewServer(mux)
	t.Cleanup(source.Close)
	c := newHTTPImageClient(t, source)
	ctx := context.Background()

	t.Run("plain fetch", func(t *testing.T) {
		body, contentType, err := c.fetchImage(ctx, source.URL+"/ok.png")
		require.NoError(t, err)
		assert.Equal(t, pngBytes, body)
		assert.Equal(t, "image/png", contentType)
	})
	t.Run("same-host redirect is followed", func(t *testing.T) {
		body, _, err := c.fetchImage(ctx, source.URL+"/to-ok")
		require.NoError(t, err)
		assert.Equal(t, pngBytes, body)
	})
	t.Run("off-host redirect is refused and never requested", func(t *testing.T) {
		_, _, err := c.fetchImage(ctx, source.URL+"/to-off-host")
		assert.Error(t, err)
		assert.Zero(t, offHostHits.Load())
	})
	t.Run("redirect loop is capped", func(t *testing.T) {
		_, _, err := c.fetchImage(ctx, source.URL+"/loop")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "redirects")
	})
	t.Run("streamed body over the cap", func(t *testing.T) {
		_, _, err := c.fetchImage(ctx, source.URL+"/streamed-too-big")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit")
	})
	t.Run("declared length over the cap", func(t *testing.T) {
		_, _, err := c.fetchImage(ctx, source.URL+"/declared-too-big")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit")
	})
	t.Run("off-host address is rejected without a request", func(t *testing.T) {
		_, _, err := c.fetchImage(ctx, offHost.URL+"/x.png")
		assert.Error(t, err)
		assert.Zero(t, offHostHits.Load())
	})
}
