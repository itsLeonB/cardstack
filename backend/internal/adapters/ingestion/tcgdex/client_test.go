package tcgdex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testClient(t *testing.T, handler http.HandlerFunc) *client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &client{baseURL: server.URL, httpClient: server.Client()}
}

func TestClient_GetSeries(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/id/series/sv", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "SV", "name": "Scarlet & Violet",
			"sets": [{"id": "SV1V", "name": "Violet ex"}]
		}`))
	})

	got, err := c.getSeries(context.Background(), "id", "sv")
	require.NoError(t, err)
	assert.Equal(t, seriesResponse{
		ID:   "SV",
		Name: "Scarlet & Violet",
		Sets: []seriesSetRef{{ID: "SV1V", Name: "Violet ex"}},
	}, got)
}

func TestClient_GetSet(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/id/sets/SV1V", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"cardCount": {"total": 78, "official": 78},
			"cards": [{"id": "SV1V-001", "localId": "001", "name": "Pineco", "image": "https://assets.tcgdex.net/id/SV/SV1V/001"}]
		}`))
	})

	got, err := c.getSet(context.Background(), "id", "SV1V")
	require.NoError(t, err)
	assert.Equal(t, setResponse{
		CardCount: setCardCount{Total: 78, Official: 78},
		Cards: []setCardRef{
			{ID: "SV1V-001", LocalID: "001", Name: "Pineco", Image: "https://assets.tcgdex.net/id/SV/SV1V/001"},
		},
	}, got)
}

func TestClient_GetCard_Found(t *testing.T) {
	// futureField is deliberately not modeled by cardResponse, to prove raw
	// carries the exact response bytes rather than a remarshal of the
	// decoded struct (which would silently drop it).
	const body = `{
		"id": "SV1V-008", "localId": "008", "name": "Spidops ex",
		"category": "Pokemon", "rarity": "Double rare",
		"variants": {"holo": true, "normal": false, "reverse": false, "firstEdition": false, "wPromo": false},
		"hp": 260, "futureField": {"nested": true}
	}`

	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/id/cards/SV1V-008", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})

	got, raw, found, err := c.getCard(context.Background(), "id", "SV1V-008")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "Spidops ex", got.Name)
	assert.Equal(t, 260, got.HP)
	assert.True(t, got.Variants.Holo)
	assert.JSONEq(t, body, string(raw), "raw must be the exact response bytes, including fields cardResponse doesn't model")
}

func TestClient_GetCard_NotFoundIsNotAnError(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	got, raw, found, err := c.getCard(context.Background(), "en", "SV1V-008")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, cardResponse{}, got)
	assert.Nil(t, raw)
}

func TestClient_GetCard_ServerError(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, raw, found, err := c.getCard(context.Background(), "id", "SV1V-008")
	assert.Error(t, err)
	assert.False(t, found)
	assert.Nil(t, raw)
}
