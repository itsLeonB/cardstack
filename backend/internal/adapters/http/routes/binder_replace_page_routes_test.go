package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func replacePageAt(api testAPI, token, id string, page int, slots ...map[string]any) *httptest.ResponseRecorder {
	return api.Put(binderPath(id)+"/pages/"+strconv.Itoa(page), bearer(token), map[string]any{"slots": slots})
}

func emptySlot() map[string]any        { return map[string]any{"state": "empty"} }
func unidentifiedSlot() map[string]any { return map[string]any{"state": "unidentified"} }
func cardSlot(card uuid.UUID) map[string]any {
	return map[string]any{"state": "card", "cardId": card}
}

func emptySlots(n int) []map[string]any {
	slots := make([]map[string]any, n)
	for i := range slots {
		slots[i] = emptySlot()
	}
	return slots
}

func decodeReplacePage(t *testing.T, resp *httptest.ResponseRecorder) dto.BinderPageReplaceResult {
	t.Helper()
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var env struct {
		Data dto.BinderPageReplaceResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &env))
	return env.Data
}

func TestBinderReplacePageSetsEverySlot(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	cards := newTestCards(t, 2)
	a, b := cards[0], cards[1]
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, a))
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 2, 2, a))
	decodeBinderPage(t, markUnidentifiedAt(api, owner, id, 1, 1, 2))
	requireEntriesMatchSlots(t, api, id)

	// Grid order is row by row: the request lists (1,1), (1,2), (2,1), (2,2).
	result := decodeReplacePage(t, replacePageAt(api, owner, id, 1,
		cardSlot(b), emptySlot(), unidentifiedSlot(), emptySlot()))
	require.Len(t, result.Slots, 4)
	assert.Equal(t, dto.BinderSlotCard, result.Slots[0].State)
	require.NotNil(t, result.Slots[0].CardID)
	assert.Equal(t, b, *result.Slots[0].CardID)
	assert.Equal(t, dto.BinderSlotEmpty, result.Slots[1].State)
	assert.Equal(t, dto.BinderSlotUnidentified, result.Slots[2].State)
	assert.Equal(t, dto.BinderSlotEmpty, result.Slots[3].State)

	page := getPage(t, api, owner, id, 1)
	state, cardID := slotState(page.Data, 1, 1)
	assert.Equal(t, "card", state)
	assert.Equal(t, b.String(), cardID)
	state, _ = slotState(page.Data, 1, 2)
	assert.Equal(t, "empty", state)
	state, _ = slotState(page.Data, 2, 1)
	assert.Equal(t, "unidentified", state)
	state, _ = slotState(page.Data, 2, 2)
	assert.Equal(t, "empty", state)
	assert.Equal(t, 2, page.Data.CardTotal, "one Card plus one Unidentified Slot")
	assert.Equal(t, 1, page.Data.UnidentifiedCount)

	assert.Equal(t, 0, masterQuantity(t, api, owner, a), "both copies of A left the Page")
	assert.Equal(t, 1, masterQuantity(t, api, owner, b))
	requireEntriesMatchSlots(t, api, id)
}

func TestBinderReplacePageTwiceGivesSameResponse(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	card := newTestCards(t, 1)[0]

	first := replacePageAt(api, owner, id, 1,
		cardSlot(card), unidentifiedSlot(), emptySlot(), cardSlot(card))
	second := replacePageAt(api, owner, id, 1,
		cardSlot(card), unidentifiedSlot(), emptySlot(), cardSlot(card))

	assert.Equal(t, first.Body.String(), second.Body.String())
	page := getPage(t, api, owner, id, 1)
	state, _ := slotState(page.Data, 2, 2)
	assert.Equal(t, "card", state)
	assert.Equal(t, 2, masterQuantity(t, api, owner, card), "two Slots hold two copies")
	requireEntriesMatchSlots(t, api, id)
}

func TestBinderReplacePageStoresUnknownCardAsUnidentified(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	card := newTestCards(t, 1)[0]
	unknown := uuid.New()

	result := decodeReplacePage(t, replacePageAt(api, owner, id, 1,
		cardSlot(unknown), cardSlot(card), emptySlot(), emptySlot()))
	require.Len(t, result.Slots, 4)
	assert.Equal(t, dto.BinderSlotUnidentified, result.Slots[0].State)
	assert.Equal(t, dto.BinderSlotReasonUnknownCard, result.Slots[0].Reason)
	assert.Nil(t, result.Slots[0].CardID)
	assert.Equal(t, dto.BinderSlotCard, result.Slots[1].State)
	assert.Empty(t, result.Slots[1].Reason)
	require.NotNil(t, result.Slots[1].CardID)
	assert.Equal(t, card, *result.Slots[1].CardID)

	page := getPage(t, api, owner, id, 1)
	state, _ := slotState(page.Data, 1, 1)
	assert.Equal(t, "unidentified", state)
	state, cardID := slotState(page.Data, 1, 2)
	assert.Equal(t, "card", state)
	assert.Equal(t, card.String(), cardID)
	assert.Equal(t, 1, page.Data.UnidentifiedCount)
	assert.Equal(t, 1, masterQuantity(t, api, owner, card))
	requireEntriesMatchSlots(t, api, id)
}

func TestBinderReplacePageRefusalsChangeNothing(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	card := newTestCards(t, 1)[0]
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, card))
	before := getPage(t, api, owner, id, 1)

	// A 2×2 Layout needs four states, so three is refused as a whole.
	resp := replacePageAt(api, owner, id, 1, cardSlot(card), emptySlot(), emptySlot())
	assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())

	// A Page past the last Leaf.
	resp = replacePageAt(api, owner, id, 3, emptySlots(4)...)
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	// Another user's binder.
	other := api.newUserToken(t)
	resp = replacePageAt(api, other, id, 1, emptySlots(4)...)
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	// A plain Collection has no Pages.
	plain := api.Post("/collections", bearer(owner), map[string]any{"title": "Box"})
	require.Equal(t, http.StatusCreated, plain.Code, plain.Body.String())
	var box collectionEnvelope
	require.NoError(t, json.Unmarshal(plain.Body.Bytes(), &box))
	resp = replacePageAt(api, owner, box.Data.ID, 1, emptySlots(4)...)
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	resp = replacePageAt(api, "", id, 1, emptySlots(4)...)
	assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())

	after := getPage(t, api, owner, id, 1)
	assert.Equal(t, before.Data, after.Data)
	assert.Equal(t, 1, masterQuantity(t, api, owner, card))
	requireEntriesMatchSlots(t, api, id)
}
