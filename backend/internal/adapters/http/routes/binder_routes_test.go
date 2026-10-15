package routes

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type binderLayoutJSON struct {
	Rows      int `json:"rows"`
	Columns   int `json:"columns"`
	LeafCount int `json:"leafCount"`
	Capacity  int `json:"capacity"`
}

type binderLayoutEnvelope struct {
	Data binderLayoutJSON `json:"data"`
}

type binderPageEnvelope struct {
	Data struct {
		Page              int              `json:"page"`
		Leaf              int              `json:"leaf"`
		Side              string           `json:"side"`
		Layout            binderLayoutJSON `json:"layout"`
		CardTotal         int              `json:"cardTotal"`
		UnidentifiedCount int              `json:"unidentifiedCount"`
		Slots             []struct {
			Row    int    `json:"row"`
			Column int    `json:"column"`
			State  string `json:"state"`
			Card   *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"card"`
		} `json:"slots"`
	} `json:"data"`
}

type errorCodeBody struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func decodeError(t *testing.T, body []byte) errorCodeBody {
	t.Helper()
	var e errorCodeBody
	require.NoError(t, json.Unmarshal(body, &e), string(body))
	return e
}

func newBinderCollection(t *testing.T, api testAPI, token string) string {
	t.Helper()
	resp := api.Post("/collections", bearer(token), map[string]any{"title": "Binder"})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	var c collectionEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &c))
	return c.Data.ID
}

func createBinder(t *testing.T, api testAPI, token, id string, rows, columns, leafCount int) {
	t.Helper()
	resp := api.Post(binderPath(id), bearer(token), map[string]any{"rows": rows, "columns": columns, "leafCount": leafCount})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
}

func binderPath(id string) string { return "/collections/" + id + "/binder" }

func getPage(t *testing.T, api testAPI, token, id string, page int) binderPageEnvelope {
	t.Helper()
	resp := api.Get(binderPath(id)+"/pages/"+strconv.Itoa(page), bearer(token))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var env binderPageEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &env))
	return env
}

// placeSlot writes one occupied Slot straight into the database: no route
// places a card until the Slot editing ticket, so the read and refusal tests
// need the state set up directly. page is the Page number, counted in turning order.
func placeSlot(t *testing.T, api testAPI, collectionID string, page, row, column int, cardID *uuid.UUID, unidentified bool) {
	t.Helper()
	cid := uuid.MustParse(collectionID)
	var leaf entity.BinderLeaf
	require.NoError(t, api.db.Where("collection_id = ? AND position = ?", cid, (page-1)/2+1).First(&leaf).Error)
	side := "front"
	if page%2 == 0 {
		side = "back"
	}
	slot := entity.BinderSlot{
		CollectionID: cid,
		LeafID:       leaf.ID,
		Side:         side,
		Row:          row,
		Column:       column,
		CardID:       cardID,
		Unidentified: unidentified,
	}
	require.NoError(t, api.db.Omit("Card").Create(&slot).Error)
}

func TestBinderLayoutLifecycle(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)

	resp := api.Post("/collections", bearer(owner), map[string]any{"title": "Binder", "maxCardCount": 50})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	var created collectionEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &created))
	id := created.Data.ID

	resp = api.Post(binderPath(id), bearer(owner), map[string]any{"rows": 2, "columns": 3, "leafCount": 4})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	var layout binderLayoutEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &layout))
	assert.Equal(t, binderLayoutJSON{Rows: 2, Columns: 3, LeafCount: 4, Capacity: 48}, layout.Data)

	// Setting a Layout clears the Collection's own cap, since capacity is now the Slot count.
	getResp := api.Get("/collections/"+id, bearer(owner))
	require.Equal(t, http.StatusOK, getResp.Code, getResp.Body.String())
	var c collectionEnvelope
	require.NoError(t, json.Unmarshal(getResp.Body.Bytes(), &c))
	assert.Zero(t, c.Data.MaxCardCount)

	resp = api.Post(binderPath(id), bearer(owner), map[string]any{"rows": 2, "columns": 2})
	assert.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())

	resp = api.Put(binderPath(id)+"/layout", bearer(owner), map[string]any{"rows": 3, "columns": 3})
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &layout))
	assert.Equal(t, binderLayoutJSON{Rows: 3, Columns: 3, LeafCount: 4, Capacity: 72}, layout.Data)

	resp = api.Post(binderPath(id)+"/leaves", bearer(owner), map[string]any{"leaf": 1, "placement": "after"})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &layout))
	assert.Equal(t, 5, layout.Data.LeafCount)
	assert.Equal(t, 90, layout.Data.Capacity)

	resp = api.Post(binderPath(id)+"/leaves", bearer(owner), map[string]any{"leaf": 1, "placement": "sideways"})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())

	resp = api.Post(binderPath(id)+"/leaves", bearer(owner), map[string]any{"leaf": 9, "placement": "before"})
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	resp = api.Delete(binderPath(id)+"/leaves/5", bearer(owner))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &layout))
	assert.Equal(t, 4, layout.Data.LeafCount)

	resp = api.Delete(binderPath(id), bearer(owner))
	assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
	resp = api.Get(binderPath(id)+"/pages/1", bearer(owner))
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
}

func TestBinderLayoutValidation(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)

	cases := []struct {
		name string
		body map[string]any
	}{
		{name: "rows below 1", body: map[string]any{"rows": 0, "columns": 3, "leafCount": 10}},
		{name: "columns above 6", body: map[string]any{"rows": 3, "columns": 7, "leafCount": 10}},
		{name: "leaf count above 100", body: map[string]any{"rows": 3, "columns": 3, "leafCount": 101}},
		{name: "leaf count below 1", body: map[string]any{"rows": 3, "columns": 3, "leafCount": 0}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			id := newBinderCollection(t, api, owner)
			resp := api.Post(binderPath(id), bearer(owner), tt.body)
			assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())

			resp = api.Get(binderPath(id)+"/pages/1", bearer(owner))
			assert.Equal(t, http.StatusNotFound, resp.Code, "a refused Layout must not make a binder")
		})
	}

	id := newBinderCollection(t, api, owner)
	resp := api.Post(binderPath(id), bearer(owner), map[string]any{"rows": 2, "columns": 2})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	var layout binderLayoutEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &layout))
	assert.Equal(t, 10, layout.Data.LeafCount, "leaf count defaults to 10")
}

func TestBinderLeafLimits(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)

	full := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, full, 2, 2, 100)
	resp := api.Post(binderPath(full)+"/leaves", bearer(owner), map[string]any{"leaf": 1, "placement": "before"})
	assert.Equal(t, http.StatusBadRequest, resp.Code, "a 101st Leaf must be refused: %s", resp.Body.String())

	single := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, single, 2, 2, 1)
	resp = api.Delete(binderPath(single)+"/leaves/1", bearer(owner))
	assert.Equal(t, http.StatusBadRequest, resp.Code, "the last Leaf must not be removable: %s", resp.Body.String())
}

func TestBinderPageRead(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 2)

	front := getPage(t, api, owner, id, 1)
	assert.Equal(t, 1, front.Data.Page)
	assert.Equal(t, 1, front.Data.Leaf)
	assert.Equal(t, "front", front.Data.Side)
	assert.Equal(t, binderLayoutJSON{Rows: 2, Columns: 2, LeafCount: 2, Capacity: 16}, front.Data.Layout)
	assert.Zero(t, front.Data.CardTotal)
	require.Len(t, front.Data.Slots, 4)
	for i, want := range [][2]int{{1, 1}, {1, 2}, {2, 1}, {2, 2}} {
		assert.Equal(t, want[0], front.Data.Slots[i].Row, "slot %d in grid order", i)
		assert.Equal(t, want[1], front.Data.Slots[i].Column, "slot %d in grid order", i)
		assert.Equal(t, "empty", front.Data.Slots[i].State)
	}

	back := getPage(t, api, owner, id, 4)
	assert.Equal(t, 2, back.Data.Leaf)
	assert.Equal(t, "back", back.Data.Side)

	resp := api.Get(binderPath(id)+"/pages/5", bearer(owner))
	assert.Equal(t, http.StatusNotFound, resp.Code, "page 5 is past the last Leaf")
	resp = api.Get(binderPath(id)+"/pages/0", bearer(owner))
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
}

func TestBinderPageShowsCardsAndUnidentified(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)

	cards := newTestCards(t, 1)
	placeSlot(t, api, id, 1, 1, 2, &cards[0], false)
	placeSlot(t, api, id, 1, 2, 2, nil, true)

	page := getPage(t, api, owner, id, 1)
	assert.Equal(t, 2, page.Data.CardTotal)
	assert.Equal(t, 1, page.Data.UnidentifiedCount)
	require.Len(t, page.Data.Slots, 4)
	assert.Equal(t, "empty", page.Data.Slots[0].State)
	assert.Equal(t, "card", page.Data.Slots[1].State)
	require.NotNil(t, page.Data.Slots[1].Card)
	assert.Equal(t, cards[0].String(), page.Data.Slots[1].Card.ID)
	assert.NotEmpty(t, page.Data.Slots[1].Card.Name)
	assert.Equal(t, "unidentified", page.Data.Slots[3].State)
	assert.Nil(t, page.Data.Slots[3].Card)
}

func TestBinderAddLeafRenumbersPages(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 2)

	cards := newTestCards(t, 1)
	placeSlot(t, api, id, 3, 1, 1, &cards[0], false)

	resp := api.Post(binderPath(id)+"/leaves", bearer(owner), map[string]any{"leaf": 1, "placement": "after"})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	// The card was on leaf 2's front (page 3). A Leaf added after leaf 1 pushes it to leaf 3's front (page 5).
	assert.Equal(t, "empty", getPage(t, api, owner, id, 3).Data.Slots[0].State)
	assert.Equal(t, "card", getPage(t, api, owner, id, 5).Data.Slots[0].State)

	resp = api.Post(binderPath(id)+"/leaves", bearer(owner), map[string]any{"leaf": 1, "placement": "before"})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	assert.Equal(t, "card", getPage(t, api, owner, id, 7).Data.Slots[0].State)
}

func TestBinderRemoveLeafRefusedWhenPageHoldsSlot(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 2)

	placeSlot(t, api, id, 4, 2, 2, nil, true)

	resp := api.Delete(binderPath(id)+"/leaves/2", bearer(owner))
	require.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
	assert.Equal(t, "binder_leaf_in_use", decodeError(t, resp.Body.Bytes()).Code)
	assert.Contains(t, decodeError(t, resp.Body.Bytes()).Detail, "page 4")

	// Leaf 1 is empty, so removing it is allowed, and the occupied Leaf
	// renumbers to 1 with its back Page now page 2.
	resp = api.Delete(binderPath(id)+"/leaves/1", bearer(owner))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.Equal(t, 1, getPage(t, api, owner, id, 2).Data.UnidentifiedCount)
}

func TestBinderLayoutChangeRefusedWhenSlotFilled(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	placeSlot(t, api, id, 1, 1, 1, nil, true)

	resp := api.Put(binderPath(id)+"/layout", bearer(owner), map[string]any{"rows": 3, "columns": 3})
	require.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
	assert.Equal(t, "binder_layout_locked", decodeError(t, resp.Body.Bytes()).Code)

	resp = api.Delete(binderPath(id), bearer(owner))
	require.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
	assert.Equal(t, "binder_layout_locked", decodeError(t, resp.Body.Bytes()).Code)
}

func TestSetLayoutRefusedWhenCollectionHoldsEntries(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	cards := newTestCards(t, 1)

	resp := api.Post("/collections/"+id+"/entries", bearer(owner), map[string]any{"cardId": cards[0], "quantity": 1})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

	resp = api.Post(binderPath(id), bearer(owner), map[string]any{"rows": 2, "columns": 2})
	require.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
	assert.Equal(t, "collection_not_empty", decodeError(t, resp.Body.Bytes()).Code)
}

func TestBinderRefusesQuantityWrites(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	cards := newTestCards(t, 1)
	entries := "/collections/" + id + "/entries"

	resp := api.Post(entries, bearer(owner), map[string]any{"cardId": cards[0], "quantity": 1})
	assert.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
	assert.Equal(t, "binder_quantity_locked", decodeError(t, resp.Body.Bytes()).Code)

	resp = api.Patch(entries, bearer(owner), map[string]any{"items": []map[string]any{{"cardId": cards[0], "quantity": 1}}})
	assert.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
	assert.Equal(t, "binder_quantity_locked", decodeError(t, resp.Body.Bytes()).Code)

	resp = api.Put(entries+"/"+cards[0].String(), bearer(owner), map[string]any{"quantity": 2})
	assert.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
	assert.Equal(t, "binder_quantity_locked", decodeError(t, resp.Body.Bytes()).Code)

	resp = api.Delete(entries+"/"+cards[0].String(), bearer(owner))
	assert.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
	assert.Equal(t, "binder_quantity_locked", decodeError(t, resp.Body.Bytes()).Code)
}

func TestDeleteBinderDeletesSlots(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 2)
	cards := newTestCards(t, 1)
	placeSlot(t, api, id, 3, 1, 1, &cards[0], false)
	placeSlot(t, api, id, 4, 2, 2, nil, true)

	resp := api.Delete("/collections/"+id, bearer(owner))
	require.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())

	cid := uuid.MustParse(id)
	var slots, leaves, layouts int64
	require.NoError(t, api.db.Model(&entity.BinderSlot{}).Where("collection_id = ?", cid).Count(&slots).Error)
	require.NoError(t, api.db.Model(&entity.BinderLeaf{}).Where("collection_id = ?", cid).Count(&leaves).Error)
	require.NoError(t, api.db.Model(&entity.CollectionLayout{}).Where("collection_id = ?", cid).Count(&layouts).Error)
	assert.Zero(t, slots)
	assert.Zero(t, leaves)
	assert.Zero(t, layouts)
}

func TestBinderOwnership(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	other := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 2)

	resp := api.Get(binderPath(id)+"/pages/1", bearer(other))
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	resp = api.Post(binderPath(id)+"/leaves", bearer(other), map[string]any{"leaf": 1, "placement": "after"})
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	resp = api.Put(binderPath(id)+"/layout", bearer(other), map[string]any{"rows": 3, "columns": 3})
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	resp = api.Delete(binderPath(id), bearer(other))
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	resp = api.Get(binderPath(id) + "/pages/1")
	assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
}

func TestBinderSummaryOnCollections(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	plain := newBinderCollection(t, api, owner)
	binder := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, binder, 2, 2, 3)

	type summaryEnvelope struct {
		Data struct {
			ID     string            `json:"id"`
			Binder *binderLayoutJSON `json:"binder"`
		} `json:"data"`
	}

	resp := api.Get("/collections/"+binder, bearer(owner))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var got summaryEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &got))
	assert.Equal(t, &binderLayoutJSON{Rows: 2, Columns: 2, LeafCount: 3, Capacity: 24}, got.Data.Binder)

	resp = api.Get("/collections/"+plain, bearer(owner))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &got))
	assert.Nil(t, got.Data.Binder)

	resp = api.Get("/collections", bearer(owner))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var list struct {
		Data []summaryEnvelope `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &list))
	for _, c := range list.Data {
		switch c.Data.ID {
		case binder:
			assert.Equal(t, &binderLayoutJSON{Rows: 2, Columns: 2, LeafCount: 3, Capacity: 24}, c.Data.Binder)
		case plain:
			assert.Nil(t, c.Data.Binder)
		}
	}
}

func TestBinderRefusesMaxCardCount(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)

	resp := api.Put("/collections/"+id, bearer(owner), map[string]any{"title": "Binder", "maxCardCount": 50})
	require.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
	assert.Equal(t, "binder_max_card_count", decodeError(t, resp.Body.Bytes()).Code)

	resp = api.Put("/collections/"+id, bearer(owner), map[string]any{"title": "Binder", "maxCardCount": 0})
	assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
}
