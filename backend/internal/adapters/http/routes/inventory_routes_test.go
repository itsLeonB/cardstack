package routes

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// newTestCards inserts n Cards (with their own Game/Locale/Rarity/Set) and
// returns their IDs.
func newTestCards(t *testing.T, n int) []uuid.UUID {
	t.Helper()
	return newTestCardsInSet(t, n, nil)
}

// newTestCardsInSet is newTestCards with the Set's release date (nil = unknown).
func newTestCardsInSet(t *testing.T, n int, releaseDate *time.Time) []uuid.UUID {
	t.Helper()
	dsn := "host=" + envOr("DB_HOST", "localhost") +
		" port=" + envOr("DB_PORT", "5432") +
		" user=" + envOr("DB_USER", "cardstack") +
		" password=" + envOr("DB_PASSWORD", "cardstack") +
		" dbname=" + envOr("DB_NAME", "cardstack") +
		" sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)

	s := uuid.NewString()
	game := entity.Game{Slug: "inv-" + s, Name: "Inv " + s}
	require.NoError(t, db.Create(&game).Error)
	locale := entity.Locale{Code: "inv-" + s}
	require.NoError(t, db.Create(&locale).Error)
	rarity := entity.Rarity{GameID: game.ID, Code: "R-" + s, Name: "Rarity " + s}
	require.NoError(t, db.Create(&rarity).Error)
	set := entity.ExpansionSet{GameID: game.ID, Code: "set-" + s, Name: "Set " + s, LocaleID: locale.ID, ReleaseDate: releaseDate}
	require.NoError(t, db.Create(&set).Error)

	ids := make([]uuid.UUID, n)
	for i := range ids {
		card := entity.Card{ExpansionSetID: set.ID, LocalID: string(rune('a' + i)), Name: "Card " + s, Category: "Pokémon",
			Tags: datatypes.JSONSlice[string]{}, RarityID: rarity.ID, Attributes: datatypes.JSONMap{}}
		require.NoError(t, db.Create(&card).Error)
		ids[i] = card.ID
	}
	return ids
}

// TestInventoryFlow covers the add/update/remove/view happy path, the
// capacity-limit rejection and cross-user isolation; branch cases live in
// the service unit tests.
func TestInventoryFlow(t *testing.T) {
	services := authTestServices(t)
	_, api := humatest.New(t, httpapi.NewConfig())
	RegisterRoutes(api, services)
	cards := newTestCards(t, 2)

	owner := registerAndLogin(t, api, uuid.NewString()+"@example.com", "correct-horse-battery-staple")
	other := registerAndLogin(t, api, uuid.NewString()+"@example.com", "correct-horse-battery-staple")

	createResp := api.Post("/collections", cookieHeader(owner), csrfHeader(owner), map[string]any{"title": "Binder", "maxCardCount": 5})
	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	var created collectionEnvelope
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &created))
	base := "/collections/" + created.Data.ID + "/entries"

	assert.Equal(t, http.StatusUnauthorized, api.Get(base).Code)

	resp := api.Post(base, cookieHeader(owner), csrfHeader(owner), map[string]any{"cardId": cards[0], "quantity": 3})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	resp = api.Post(base, cookieHeader(owner), csrfHeader(owner), map[string]any{"cardId": cards[1], "quantity": 2})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

	// A Card already in the Collection is a conflict, not a merge.
	resp = api.Post(base, cookieHeader(owner), csrfHeader(owner), map[string]any{"cardId": cards[0], "quantity": 1})
	assert.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())

	// Full: any further increase is rejected, a decrease is fine.
	resp = api.Put(base+"/"+cards[0].String(), cookieHeader(owner), csrfHeader(owner), map[string]any{"quantity": 4})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	resp = api.Put(base+"/"+cards[0].String(), cookieHeader(owner), csrfHeader(owner), map[string]any{"quantity": 1})
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

	resp = api.Get(base, cookieHeader(owner))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var list struct {
		Data []struct {
			Card     struct{ ID string } `json:"card"`
			Quantity int                 `json:"quantity"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &list))
	got := map[string]int{}
	for _, it := range list.Data {
		got[it.Card.ID] = it.Quantity
	}
	assert.Equal(t, map[string]int{cards[0].String(): 1, cards[1].String(): 2}, got)

	// Another user sees a missing Collection on every verb.
	assert.Equal(t, http.StatusNotFound, api.Get(base, cookieHeader(other)).Code)
	resp = api.Post(base, cookieHeader(other), csrfHeader(other), map[string]any{"cardId": cards[0], "quantity": 1})
	assert.Equal(t, http.StatusNotFound, resp.Code)
	resp = api.Put(base+"/"+cards[0].String(), cookieHeader(other), csrfHeader(other), map[string]any{"quantity": 1})
	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.Equal(t, http.StatusNotFound, api.Delete(base+"/"+cards[0].String(), cookieHeader(other), csrfHeader(other)).Code)

	// CSRF guard covers the mutating routes.
	assert.Equal(t, http.StatusForbidden, api.Delete(base+"/"+cards[0].String(), cookieHeader(owner)).Code)

	resp = api.Delete(base+"/"+cards[0].String(), cookieHeader(owner), csrfHeader(owner))
	assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
	resp = api.Delete(base+"/"+cards[0].String(), cookieHeader(owner), csrfHeader(owner))
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	// A removed entry stays removed: update reports 404, never resurrects it.
	resp = api.Put(base+"/"+cards[0].String(), cookieHeader(owner), csrfHeader(owner), map[string]any{"quantity": 1})
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	resp = api.Get(base, cookieHeader(owner))
	require.Equal(t, http.StatusOK, resp.Code)
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &list))
	assert.Len(t, list.Data, 1)
}

// TestInventoryListOrder: the list sorts by Set release date, unknown last.
func TestInventoryListOrder(t *testing.T) {
	services := authTestServices(t)
	_, api := humatest.New(t, httpapi.NewConfig())
	RegisterRoutes(api, services)
	released := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	unknown := newTestCardsInSet(t, 1, nil)[0]
	known := newTestCardsInSet(t, 1, &released)[0]

	owner := registerAndLogin(t, api, uuid.NewString()+"@example.com", "correct-horse-battery-staple")
	createResp := api.Post("/collections", cookieHeader(owner), csrfHeader(owner), map[string]any{"title": "Binder"})
	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	var created collectionEnvelope
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &created))
	base := "/collections/" + created.Data.ID + "/entries"

	for _, id := range []uuid.UUID{unknown, known} {
		resp := api.Post(base, cookieHeader(owner), csrfHeader(owner), map[string]any{"cardId": id, "quantity": 1})
		require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	}

	resp := api.Get(base, cookieHeader(owner))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var list struct {
		Data []struct {
			Card struct{ ID string } `json:"card"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &list))
	require.Len(t, list.Data, 2)
	assert.Equal(t, known.String(), list.Data[0].Card.ID)
	assert.Equal(t, unknown.String(), list.Data[1].Card.ID)
}

type bulkResult struct {
	CardID   string `json:"cardId"`
	Quantity int    `json:"quantity"`
	Status   string `json:"status"`
	Reason   string `json:"reason"`
	Message  string `json:"message"`
}

// TestInventoryBulkUpdateFlow covers the bulk endpoint end to end: mixed
// batch, partial success, auth/CSRF/ownership, and concurrent batches on one
// Collection serializing on its row lock. Branch cases live in the service
// unit tests.
func TestInventoryBulkUpdateFlow(t *testing.T) {
	services := authTestServices(t)
	_, api := humatest.New(t, httpapi.NewConfig())
	RegisterRoutes(api, services)
	cards := newTestCards(t, 4)

	owner := registerAndLogin(t, api, uuid.NewString()+"@example.com", "correct-horse-battery-staple")
	other := registerAndLogin(t, api, uuid.NewString()+"@example.com", "correct-horse-battery-staple")

	createResp := api.Post("/collections", cookieHeader(owner), csrfHeader(owner), map[string]any{"title": "Binder", "maxCardCount": 5})
	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	var created collectionEnvelope
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &created))
	base := "/collections/" + created.Data.ID + "/entries"

	items := func(pairs ...any) map[string]any {
		var list []map[string]any
		for i := 0; i < len(pairs); i += 2 {
			list = append(list, map[string]any{"cardId": pairs[i], "quantity": pairs[i+1]})
		}
		return map[string]any{"items": list}
	}
	patch := func(user []*http.Cookie, path string, body any) (int, []bulkResult) {
		resp := api.Patch(path, cookieHeader(user), csrfHeader(user), body)
		var out struct {
			Data []bulkResult `json:"data"`
		}
		if resp.Code == http.StatusOK {
			require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out), resp.Body.String())
		}
		return resp.Code, out.Data
	}
	quantities := func() map[string]int {
		resp := api.Get(base, cookieHeader(owner))
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		var list struct {
			Data []struct {
				Card     struct{ ID string } `json:"card"`
				Quantity int                 `json:"quantity"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &list))
		got := map[string]int{}
		for _, it := range list.Data {
			got[it.Card.ID] = it.Quantity
		}
		return got
	}

	// Mixed batch: create, create, then capacity (5) declines the third while
	// the unknown Card is declined and a later removal still applies.
	code, res := patch(owner, base, items(cards[0], 3, cards[1], 2, cards[2], 1, uuid.New(), 1, cards[3], 0))
	require.Equal(t, http.StatusOK, code)
	require.Len(t, res, 5)
	assert.Equal(t, bulkResult{CardID: cards[0].String(), Quantity: 3, Status: "applied"}, res[0])
	assert.Equal(t, bulkResult{CardID: cards[1].String(), Quantity: 2, Status: "applied"}, res[1])
	assert.Equal(t, "declined", res[2].Status)
	assert.Equal(t, "capacity_exceeded", res[2].Reason)
	assert.NotEmpty(t, res[2].Message)
	assert.Equal(t, "card_not_found", res[3].Reason)
	assert.Equal(t, bulkResult{CardID: cards[3].String(), Status: "removed"}, res[4])
	assert.Equal(t, map[string]int{cards[0].String(): 3, cards[1].String(): 2}, quantities())

	// Idempotent retry; decrease frees room, remove and re-add in one batch.
	code, res = patch(owner, base, items(cards[0], 3, cards[1], 0, cards[2], 2))
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, []string{"applied", "removed", "applied"}, []string{res[0].Status, res[1].Status, res[2].Status})
	assert.Equal(t, map[string]int{cards[0].String(): 3, cards[2].String(): 2}, quantities())

	// Client errors change nothing.
	code, _ = patch(owner, base, items(cards[0], 1, cards[0], 2))
	assert.Equal(t, http.StatusBadRequest, code, "duplicate cardId")
	code, _ = patch(owner, base, items(cards[0], -1))
	assert.Equal(t, http.StatusUnprocessableEntity, code, "negative")
	code, _ = patch(owner, base, map[string]any{"items": []any{}})
	assert.Equal(t, http.StatusUnprocessableEntity, code, "empty")
	assert.Equal(t, map[string]int{cards[0].String(): 3, cards[2].String(): 2}, quantities())

	// Auth, CSRF, ownership.
	// CSRF is checked first, so an anonymous caller needs a matching token pair to reach the 401.
	assert.Equal(t, http.StatusUnauthorized, api.Patch(base, "Cookie: csrf_token=x", "X-CSRF-Token: x", items(cards[0], 1)).Code)
	assert.Equal(t, http.StatusForbidden, api.Patch(base, cookieHeader(owner), items(cards[0], 1)).Code)
	code, _ = patch(other, base, items(cards[0], 1))
	assert.Equal(t, http.StatusNotFound, code)
	code, _ = patch(owner, "/collections/"+uuid.NewString()+"/entries", items(cards[0], 1))
	assert.Equal(t, http.StatusNotFound, code)
	assert.Equal(t, map[string]int{cards[0].String(): 3, cards[2].String(): 2}, quantities())

	// Concurrent batches serialize: each wants 3 more of its own Card with 0
	// room left for both (5 - 5 = 0 after the first fills), so exactly one
	// wins and the limit holds.
	code, _ = patch(owner, base, items(cards[0], 1, cards[2], 1))
	require.Equal(t, http.StatusOK, code)
	var wg sync.WaitGroup
	statuses := make([]string, 2)
	for i, id := range []uuid.UUID{cards[1], cards[3]} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, r := patch(owner, base, items(id, 3))
			if assert.Equal(t, http.StatusOK, c) {
				statuses[i] = r[0].Status
			}
		}()
	}
	wg.Wait()
	assert.ElementsMatch(t, []string{"applied", "declined"}, statuses)
	total := 0
	for _, q := range quantities() {
		total += q
	}
	assert.Equal(t, 5, total)
}
