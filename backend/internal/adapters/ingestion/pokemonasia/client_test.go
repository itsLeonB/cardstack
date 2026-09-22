package pokemonasia

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

// testClient builds a client pointed at an httptest.Server with an
// unrestricted rate limiter — tests must not pay client.go's real-world
// politeness delay.
func testClient(t *testing.T, handler http.HandlerFunc) *client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &client{baseURL: server.URL, httpClient: server.Client(), limiter: rate.NewLimiter(rate.Inf, 0)}
}

func TestClient_ExpansionListPage(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/card-search/", r.URL.Path)
		assert.Equal(t, "2", r.URL.Query().Get("pageNo"))
		_, _ = w.Write([]byte(expansionListFixture))
	})

	doc, raw, err := c.expansionListPage(context.Background(), 2)
	require.NoError(t, err)
	assert.NotEmpty(t, raw)
	assert.Len(t, parseExpansionListings(doc), 2)
}

func TestClient_ResultsPage(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/card-search/list/", r.URL.Path)
		assert.Equal(t, "MA1", r.URL.Query().Get("expansionCodes"))
		assert.Equal(t, "1", r.URL.Query().Get("regulation"))
		assert.Equal(t, "all", r.URL.Query().Get("cardType"))
		_, _ = w.Write([]byte(resultsPageFixture))
	})

	doc, _, err := c.resultsPage(context.Background(), "MA1", 1, 1)
	require.NoError(t, err)
	assert.Equal(t, []string{"16488", "16489"}, parseResultCardIDs(doc))
}

func TestClient_CardDetail(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/card-search/detail/16488/", r.URL.Path)
		_, _ = w.Write([]byte(pokemonDetailFixture))
	})

	doc, raw, err := c.cardDetail(context.Background(), "16488")
	require.NoError(t, err)
	assert.Equal(t, []byte(pokemonDetailFixture), raw, "raw must be the exact response bytes")
	detail := parseCardDetail(doc)
	assert.Equal(t, "Mega Venusaur ex", detail.Name)
}

func TestClient_Get_ServerError(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, _, err := c.cardDetail(context.Background(), "1")
	assert.Error(t, err)
}

func TestClient_Get_TooManyRequests(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, _, err := c.cardDetail(context.Background(), "1")
	assert.Error(t, err, "a 429 must surface as an error, not be silently swallowed")
}
