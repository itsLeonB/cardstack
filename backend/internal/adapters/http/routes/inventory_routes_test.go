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
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
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

// TestInventoryBulkUpdateFlow covers the bulk endpoint end to end: apply,
// idempotent retry, auth/CSRF/ownership, and concurrent batches on one
// Collection serializing on its row lock. Branch cases (capacity, unknown
// card, duplicates) live in the service unit tests.
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
	patch := func(user []*http.Cookie, path string, body any) (int, []dto.InventoryChangeResult) {
		resp := api.Patch(path, cookieHeader(user), csrfHeader(user), body)
		var out struct {
			Data []dto.InventoryChangeResult `json:"data"`
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

	code, res := patch(owner, base, items(cards[0], 3, cards[1], 2, cards[3], 0))
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, []dto.InventoryChangeResult{
		{CardID: cards[0], Quantity: 3, Status: dto.InventoryStatusApplied},
		{CardID: cards[1], Quantity: 2, Status: dto.InventoryStatusApplied},
		{CardID: cards[3], Status: dto.InventoryStatusRemoved},
	}, res)
	assert.Equal(t, map[string]int{cards[0].String(): 3, cards[1].String(): 2}, quantities())

	// Idempotent retry of the same absolute targets, plus a remove.
	code, res = patch(owner, base, items(cards[0], 3, cards[1], 0))
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, dto.InventoryStatusApplied, res[0].Status)
	assert.Equal(t, dto.InventoryStatusRemoved, res[1].Status)
	assert.Equal(t, map[string]int{cards[0].String(): 3}, quantities())

	// Auth, CSRF, ownership.
	// CSRF is checked first, so an anonymous caller needs a matching token pair to reach the 401.
	assert.Equal(t, http.StatusUnauthorized, api.Patch(base, "Cookie: csrf_token=x", "X-CSRF-Token: x", items(cards[0], 1)).Code)
	assert.Equal(t, http.StatusForbidden, api.Patch(base, cookieHeader(owner), items(cards[0], 1)).Code)
	code, _ = patch(other, base, items(cards[0], 1))
	assert.Equal(t, http.StatusNotFound, code)
	code, _ = patch(owner, "/collections/"+uuid.NewString()+"/entries", items(cards[0], 1))
	assert.Equal(t, http.StatusNotFound, code)
	assert.Equal(t, map[string]int{cards[0].String(): 3}, quantities())

	// Concurrent batches serialize: 3 of 5 are used and each batch wants 2
	// more of its own Card, so only one fits. Exactly one is applied, and the
	// limit holds.
	var wg sync.WaitGroup
	statuses := make([]string, 2)
	for i, id := range []uuid.UUID{cards[1], cards[2]} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, r := patch(owner, base, items(id, 2))
			if assert.Equal(t, http.StatusOK, c) {
				statuses[i] = r[0].Status
			}
		}()
	}
	wg.Wait()
	assert.ElementsMatch(t, []string{dto.InventoryStatusApplied, dto.InventoryStatusDeclined}, statuses)
	total := 0
	for _, q := range quantities() {
		total += q
	}
	assert.Equal(t, 5, total)
}

// TestInventoryFilterAndFacetsFlow: the entries list filters and paginates
// like catalog search over only the Collection's Cards, facets are scoped to
// them, and another profile's Collection is 404 on both.
func TestInventoryFilterAndFacetsFlow(t *testing.T) {
	services := authTestServices(t)
	_, api := humatest.New(t, httpapi.NewConfig())
	RegisterRoutes(api, services)
	cards := newTestCards(t, 3) // one shared set/rarity, category Pokémon

	owner := registerAndLogin(t, api, uuid.NewString()+"@example.com", "correct-horse-battery-staple")
	other := registerAndLogin(t, api, uuid.NewString()+"@example.com", "correct-horse-battery-staple")
	createResp := api.Post("/collections", cookieHeader(owner), csrfHeader(owner), map[string]any{"title": "Binder"})
	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	var created collectionEnvelope
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &created))
	base := "/collections/" + created.Data.ID

	// Only the first two Cards are in the Collection.
	for i, q := range []int{3, 1} {
		resp := api.Post(base+"/entries", cookieHeader(owner), csrfHeader(owner), map[string]any{"cardId": cards[i], "quantity": q})
		require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	}

	type listResp struct {
		Data []struct {
			Card     struct{ ID string } `json:"card"`
			Quantity int                 `json:"quantity"`
		} `json:"data"`
		Meta struct{ Total, Page, Limit int } `json:"meta"`
	}
	get := func(path string, who []*http.Cookie) (int, listResp) {
		resp := api.Get(path, cookieHeader(who))
		var out listResp
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
		return resp.Code, out
	}

	code, out := get(base+"/entries?limit=1&page=2", owner)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, 2, out.Meta.Total)
	require.Len(t, out.Data, 1)
	assert.Equal(t, cards[1].String(), out.Data[0].Card.ID)
	assert.Equal(t, 1, out.Data[0].Quantity)

	// The third Card exists in the catalog but is filtered out of the Collection.
	code, out = get(base+"/entries?localId=c", owner)
	require.Equal(t, http.StatusOK, code)
	assert.Empty(t, out.Data)
	code, out = get(base+"/entries?category=Pok%C3%A9mon&category=Trainer", owner)
	require.Equal(t, http.StatusOK, code)
	assert.Len(t, out.Data, 2)

	resp := api.Get(base+"/facets", cookieHeader(owner))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var facets struct {
		Data struct {
			ExpansionSets []struct{ ID string }    `json:"expansionSets"`
			Categories    []struct{ Value string } `json:"categories"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &facets))
	assert.Len(t, facets.Data.ExpansionSets, 1)
	assert.Equal(t, "Pokémon", facets.Data.Categories[0].Value)

	assert.Equal(t, http.StatusNotFound, api.Get(base+"/entries", cookieHeader(other)).Code)
	assert.Equal(t, http.StatusNotFound, api.Get(base+"/facets", cookieHeader(other)).Code)
	assert.Equal(t, http.StatusUnauthorized, api.Get(base+"/facets").Code)
}

// TestCardHoldingsFlow: holdings list only the caller's own Collections, and
// are empty (not an error) for a user who holds nothing.
func TestCardHoldingsFlow(t *testing.T) {
	services := authTestServices(t)
	_, api := humatest.New(t, httpapi.NewConfig())
	RegisterRoutes(api, services)
	card := newTestCards(t, 1)[0]
	path := "/inventory/cards/" + card.String() + "/holdings"

	owner := registerAndLogin(t, api, uuid.NewString()+"@example.com", "correct-horse-battery-staple")
	other := registerAndLogin(t, api, uuid.NewString()+"@example.com", "correct-horse-battery-staple")
	assert.Equal(t, http.StatusUnauthorized, api.Get(path).Code)

	createResp := api.Post("/collections", cookieHeader(owner), csrfHeader(owner), map[string]any{"title": "Binder"})
	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	var created collectionEnvelope
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &created))
	resp := api.Post("/collections/"+created.Data.ID+"/entries", cookieHeader(owner), csrfHeader(owner), map[string]any{"cardId": card, "quantity": 4})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

	resp = api.Get(path, cookieHeader(owner))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.JSONEq(t, `{"data":[{"collection":{"id":"`+created.Data.ID+`","name":"Binder"},"quantity":4}]}`, resp.Body.String())

	resp = api.Get(path, cookieHeader(other))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.JSONEq(t, `{"data":[]}`, resp.Body.String())
}

// TestMasterInventoryFlow: quantities sum across the caller's Collections,
// another user's entries are never aggregated, and a Card removed from every
// Collection drops out immediately.
func TestMasterInventoryFlow(t *testing.T) {
	services := authTestServices(t)
	_, api := humatest.New(t, httpapi.NewConfig())
	RegisterRoutes(api, services)
	cards := newTestCards(t, 2)
	path := "/inventory/cards"

	owner := registerAndLogin(t, api, uuid.NewString()+"@example.com", "correct-horse-battery-staple")
	other := registerAndLogin(t, api, uuid.NewString()+"@example.com", "correct-horse-battery-staple")
	assert.Equal(t, http.StatusUnauthorized, api.Get(path).Code)

	newCollection := func(who []*http.Cookie, title string) string {
		resp := api.Post("/collections", cookieHeader(who), csrfHeader(who), map[string]any{"title": title})
		require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
		var c collectionEnvelope
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &c))
		return c.Data.ID
	}
	add := func(who []*http.Cookie, colID string, card uuid.UUID, qty int) {
		resp := api.Post("/collections/"+colID+"/entries", cookieHeader(who), csrfHeader(who), map[string]any{"cardId": card, "quantity": qty})
		require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	}
	binder, box, theirs := newCollection(owner, "Binder"), newCollection(owner, "Box"), newCollection(other, "Theirs")
	add(owner, binder, cards[0], 4)
	add(owner, box, cards[0], 3)
	add(other, theirs, cards[0], 50)
	add(other, theirs, cards[1], 9)

	type listResp struct {
		Data []struct {
			Card     struct{ ID string } `json:"card"`
			Quantity int                 `json:"quantity"`
		} `json:"data"`
		Meta struct{ Total, Page, Limit int } `json:"meta"`
	}
	get := func(who []*http.Cookie) listResp {
		resp := api.Get(path, cookieHeader(who))
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		var out listResp
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
		return out
	}

	out := get(owner)
	require.Len(t, out.Data, 1)
	assert.Equal(t, 1, out.Meta.Total)
	assert.Equal(t, cards[0].String(), out.Data[0].Card.ID)
	assert.Equal(t, 7, out.Data[0].Quantity)

	for _, col := range []string{binder, box} {
		resp := api.Delete("/collections/"+col+"/entries/"+cards[0].String(), cookieHeader(owner), csrfHeader(owner))
		require.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
	}
	assert.Empty(t, get(owner).Data)
}
