package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func postPlacements(api testAPI, token, id string, items ...map[string]any) *httptest.ResponseRecorder {
	return api.Post(binderPath(id)+"/placements", bearer(token), map[string]any{"items": items})
}

func placementItem(card uuid.UUID, quantity int) map[string]any {
	return map[string]any{"cardId": card, "quantity": quantity}
}

func decodePlacements(t *testing.T, resp *httptest.ResponseRecorder) []dto.BinderPlacementResult {
	t.Helper()
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var out struct {
		Data []dto.BinderPlacementResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
	return out.Data
}

// TestBinderPlacementFillsFirstEmptySlotsInPageOrder: copies take the first
// empty Slots counting row by row within a Page, skipping an occupied Slot,
// and a second placement of the same Card takes the next Page.
func TestBinderPlacementFillsFirstEmptySlotsInPageOrder(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	cards := newTestCards(t, 2)
	a, x := cards[0], cards[1]

	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 2, x))

	results := decodePlacements(t, postPlacements(api, owner, id, placementItem(a, 3)))
	assert.Equal(t, []dto.BinderPlacementResult{{CardID: a, Placed: 3}}, results)
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 3, masterQuantity(t, api, owner, a))
	assert.Equal(t, 1, masterQuantity(t, api, owner, x))

	_, positions := entryRow(t, listEntryRows(t, api, owner, id), a)
	assert.Equal(t, []dto.BinderSlotTarget{{Page: 1, Row: 1, Column: 1}, {Page: 1, Row: 2, Column: 1}, {Page: 1, Row: 2, Column: 2}}, positions)

	results = decodePlacements(t, postPlacements(api, owner, id, placementItem(a, 1)))
	assert.Equal(t, []dto.BinderPlacementResult{{CardID: a, Placed: 1}}, results)
	requireEntriesMatchSlots(t, api, id)
	_, positions = entryRow(t, listEntryRows(t, api, owner, id), a)
	assert.Contains(t, positions, dto.BinderSlotTarget{Page: 2, Row: 1, Column: 1})
}

// TestBinderPlacementDeclinesWhatDoesNotFit: copies that find no empty Slot
// stay declined with the reason, and the copies before them still land.
func TestBinderPlacementDeclinesWhatDoesNotFit(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 1, 1, 1)
	cards := newTestCards(t, 3)
	a, b, c := cards[0], cards[1], cards[2]

	results := decodePlacements(t, postPlacements(api, owner, id, placementItem(a, 1), placementItem(b, 3)))
	assert.Equal(t, []dto.BinderPlacementResult{
		{CardID: a, Placed: 1},
		{CardID: b, Placed: 1, Declined: 2, Reason: dto.BinderPlacementReasonNotEnoughEmptySlots, Message: "not enough empty Slots"},
	}, results)
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, b))

	results = decodePlacements(t, postPlacements(api, owner, id, placementItem(c, 1)))
	assert.Equal(t, []dto.BinderPlacementResult{
		{CardID: c, Declined: 1, Reason: dto.BinderPlacementReasonNotEnoughEmptySlots, Message: "not enough empty Slots"},
	}, results)
	requireEntriesMatchSlots(t, api, id)
	assert.Zero(t, masterQuantity(t, api, owner, c))
}

// TestBinderPlacementDeclinesUnknownCards: an unknown Card, or the nil id, is
// declined per item and does not stop the other items.
func TestBinderPlacementDeclinesUnknownCards(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 1, 2, 1)
	cards := newTestCards(t, 1)
	a := cards[0]
	unknown := uuid.New()

	results := decodePlacements(t, postPlacements(api, owner, id,
		placementItem(unknown, 2), placementItem(uuid.Nil, 1), placementItem(a, 1)))
	assert.Equal(t, []dto.BinderPlacementResult{
		{CardID: unknown, Declined: 2, Reason: dto.InventoryReasonCardNotFound, Message: "card not found"},
		{CardID: uuid.Nil, Declined: 1, Reason: dto.InventoryReasonCardNotFound, Message: "card not found"},
		{CardID: a, Placed: 1},
	}, results)
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, a))
}

// TestBinderPlacementRefusals: a plain Collection and another user's binder
// are refused, and a Card named twice in one request is a bad request.
func TestBinderPlacementRefusals(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	other := api.newUserToken(t)
	cards := newTestCards(t, 1)

	plain := newBinderCollection(t, api, owner)
	resp := postPlacements(api, owner, plain, placementItem(cards[0], 1))
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 1, 1, 1)
	resp = postPlacements(api, other, id, placementItem(cards[0], 1))
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	resp = postPlacements(api, owner, id, placementItem(cards[0], 1), placementItem(cards[0], 2))
	assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
	requireEntriesMatchSlots(t, api, id)
}

// TestBinderPlacementSkipsUnidentifiedSlot: a Slot holding an Unidentified card
// is taken, so the copy lands in the next empty Slot and the Unidentified one stays.
func TestBinderPlacementSkipsUnidentifiedSlot(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 1, 2, 1)
	cards := newTestCards(t, 1)
	a := cards[0]

	require.Equal(t, http.StatusOK, markUnidentifiedAt(api, owner, id, 1, 1, 1).Code)

	results := decodePlacements(t, postPlacements(api, owner, id, placementItem(a, 1)))
	assert.Equal(t, []dto.BinderPlacementResult{{CardID: a, Placed: 1}}, results)
	requireEntriesMatchSlots(t, api, id)
	_, positions := entryRow(t, listEntryRows(t, api, owner, id), a)
	assert.Equal(t, []dto.BinderSlotTarget{{Page: 1, Row: 1, Column: 2}}, positions)

	page := getPage(t, api, owner, id, 1)
	assert.Equal(t, 2, page.Data.CardTotal, "the Unidentified Slot still counts toward the binder total")
	assert.Equal(t, "unidentified", page.Data.Slots[0].State)
}

// TestBinderPlacementRequestBounds: a quantity is 1 to 2147483647, as the bulk
// update allows, and a request holds 1 to 100 items. The largest quantity is
// accepted and places only what fits.
func TestBinderPlacementRequestBounds(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 1, 2, 1)
	cards := newTestCards(t, 1)
	a := cards[0]
	const maxQuantity = 2147483647

	for _, quantity := range []int{0, -1, maxQuantity + 1} {
		resp := postPlacements(api, owner, id, placementItem(a, quantity))
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, "quantity %d: %s", quantity, resp.Body.String())
	}

	resp := api.Post(binderPath(id)+"/placements", bearer(owner), map[string]any{"items": []map[string]any{}})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())

	tooMany := make([]map[string]any, 0, 101)
	for range 101 {
		tooMany = append(tooMany, placementItem(uuid.New(), 1))
	}
	resp = postPlacements(api, owner, id, tooMany...)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())

	// The binder has 4 Slots, so the copies beyond them are declined.
	results := decodePlacements(t, postPlacements(api, owner, id, placementItem(a, maxQuantity)))
	assert.Equal(t, []dto.BinderPlacementResult{
		{CardID: a, Placed: 4, Declined: maxQuantity - 4, Reason: dto.BinderPlacementReasonNotEnoughEmptySlots, Message: "not enough empty Slots"},
	}, results)
	requireEntriesMatchSlots(t, api, id)
}
