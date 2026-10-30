package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// moveEntries posts a Move of the Cards from one Collection to another.
func moveEntries(api testAPI, token, source, destination string, cards ...uuid.UUID) *httptest.ResponseRecorder {
	return api.Post("/collections/"+source+"/entries/move", bearer(token), map[string]any{"cardIds": cards, "destinationId": destination})
}

// newPlainCollection creates a plain Collection, capped at maxCardCount when
// it is positive.
func newPlainCollection(t *testing.T, api testAPI, token string, maxCardCount int) string {
	t.Helper()
	resp := api.Post("/collections", bearer(token), map[string]any{"title": "Plain", "maxCardCount": maxCardCount})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	var c collectionEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &c))
	return c.Data.ID
}

func addEntry(t *testing.T, api testAPI, token, collectionID string, card uuid.UUID, quantity int) {
	t.Helper()
	resp := api.Post("/collections/"+collectionID+"/entries", bearer(token), map[string]any{"cardId": card, "quantity": quantity})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
}

// entryQuantities maps each Card in the Collection to its quantity.
func entryQuantities(t *testing.T, api testAPI, token, collectionID string) map[string]int {
	t.Helper()
	resp := api.Get("/collections/"+collectionID+"/entries", bearer(token))
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

type moveResultEnvelope struct {
	Data struct {
		Cards  int `json:"cards"`
		Copies int `json:"copies"`
	} `json:"data"`
}

// TestMoveMovesWholeEntriesAndMergesIntoExisting: every copy of each listed
// Card moves, a Card the destination already holds gains the copies, and an
// unlisted Card stays where it was.
func TestMoveMovesWholeEntriesAndMergesIntoExisting(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	cards := newTestCards(t, 3)
	source := newPlainCollection(t, api, owner, 0)
	destination := newPlainCollection(t, api, owner, 0)
	addEntry(t, api, owner, source, cards[0], 3)
	addEntry(t, api, owner, source, cards[1], 2)
	addEntry(t, api, owner, source, cards[2], 1)
	addEntry(t, api, owner, destination, cards[1], 4)

	resp := moveEntries(api, owner, source, destination, cards[0], cards[1])
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var moved moveResultEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &moved))
	assert.Equal(t, 2, moved.Data.Cards)
	assert.Equal(t, 5, moved.Data.Copies)

	assert.Equal(t, map[string]int{cards[2].String(): 1}, entryQuantities(t, api, owner, source))
	assert.Equal(t, map[string]int{cards[0].String(): 3, cards[1].String(): 6}, entryQuantities(t, api, owner, destination))
}

// TestMoveRefusedWhenDestinationWouldPassItsLimit: the whole Move is refused
// with the limit and the excess in the message, and nothing changes. A Move
// that lands exactly on the limit goes through.
func TestMoveRefusedWhenDestinationWouldPassItsLimit(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	cards := newTestCards(t, 3)
	source := newPlainCollection(t, api, owner, 0)
	destination := newPlainCollection(t, api, owner, 5)
	addEntry(t, api, owner, source, cards[1], 2)
	addEntry(t, api, owner, source, cards[2], 1)
	addEntry(t, api, owner, destination, cards[0], 3)

	resp := moveEntries(api, owner, source, destination, cards[1], cards[2])
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	assert.Contains(t, decodeError(t, resp.Body.Bytes()).Detail, "limit of 5 cards by 1 copy")
	assert.Equal(t, map[string]int{cards[1].String(): 2, cards[2].String(): 1}, entryQuantities(t, api, owner, source))
	assert.Equal(t, map[string]int{cards[0].String(): 3}, entryQuantities(t, api, owner, destination))

	resp = moveEntries(api, owner, source, destination, cards[1])
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.Equal(t, map[string]int{cards[0].String(): 3, cards[1].String(): 2}, entryQuantities(t, api, owner, destination))
}

// TestMoveRefusesBinderDestination: a binder takes no Move, since a Move
// chooses no Slot, and the refusal carries its own code.
func TestMoveRefusesBinderDestination(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	cards := newTestCards(t, 1)
	source := newPlainCollection(t, api, owner, 0)
	binder := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, binder, 2, 2, 1)
	addEntry(t, api, owner, source, cards[0], 1)

	resp := moveEntries(api, owner, source, binder, cards[0])
	assert.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
	assert.Equal(t, "move_to_binder", decodeError(t, resp.Body.Bytes()).Code)
	assert.Equal(t, map[string]int{cards[0].String(): 1}, entryQuantities(t, api, owner, source))
	assert.Empty(t, entryQuantities(t, api, owner, binder))
}

// TestMoveRefusesBinderSource: a binder's quantities change only by placing
// or clearing Slots, so its Cards cannot be moved out as entries.
func TestMoveOutOfBinderEmptiesTheSlotsHoldingTheCards(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	cards := newTestCards(t, 3)
	binder := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, binder, 2, 2, 1)
	placeCardAt(api, owner, binder, 1, 1, 1, cards[0])
	placeCardAt(api, owner, binder, 1, 1, 2, cards[0])
	placeCardAt(api, owner, binder, 1, 2, 1, cards[1])
	placeCardAt(api, owner, binder, 2, 1, 1, cards[1])
	placeCardAt(api, owner, binder, 2, 2, 2, cards[2])
	markUnidentifiedAt(api, owner, binder, 2, 1, 2)
	destination := newPlainCollection(t, api, owner, 0)
	addEntry(t, api, owner, destination, cards[1], 4)

	resp := moveEntries(api, owner, binder, destination, cards[0], cards[1])
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var moved moveResultEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &moved))
	assert.Equal(t, 2, moved.Data.Cards)
	assert.Equal(t, 4, moved.Data.Copies)

	assert.Equal(t, map[string]int{cards[0].String(): 2, cards[1].String(): 6}, entryQuantities(t, api, owner, destination))
	assert.Equal(t, map[string]int{cards[2].String(): 1}, entryQuantities(t, api, owner, binder))
	requireEntriesMatchSlots(t, api, binder)

	assert.Equal(t, []string{"empty", "empty", "empty", "empty"}, slotStates(getPage(t, api, owner, binder, 1)))
	second := getPage(t, api, owner, binder, 2)
	assert.Equal(t, []string{"empty", "unidentified", "empty", "card"}, slotStates(second))
	assert.Equal(t, cards[2].String(), second.Data.Slots[3].Card.ID)
}

func TestMoveOutOfBinderRefusedWhenDestinationWouldPassItsLimitChangesNothing(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	cards := newTestCards(t, 2)
	binder := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, binder, 2, 2, 1)
	placeCardAt(api, owner, binder, 1, 1, 1, cards[0])
	placeCardAt(api, owner, binder, 1, 1, 2, cards[0])
	placeCardAt(api, owner, binder, 1, 2, 1, cards[1])
	destination := newPlainCollection(t, api, owner, 2)
	addEntry(t, api, owner, destination, cards[1], 1)

	resp := moveEntries(api, owner, binder, destination, cards[0])
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	assert.Equal(t, map[string]int{cards[1].String(): 1}, entryQuantities(t, api, owner, destination))
	assert.Equal(t, map[string]int{cards[0].String(): 2, cards[1].String(): 1}, entryQuantities(t, api, owner, binder))
	requireEntriesMatchSlots(t, api, binder)
	assert.Equal(t, []string{"card", "card", "card", "empty"}, slotStates(getPage(t, api, owner, binder, 1)))
}

func TestMoveOutOfBinderRefusesBinderDestinationChangesNothing(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	cards := newTestCards(t, 1)
	source := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, source, 2, 2, 1)
	placeCardAt(api, owner, source, 1, 1, 1, cards[0])
	destination := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, destination, 2, 2, 1)

	resp := moveEntries(api, owner, source, destination, cards[0])
	assert.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
	assert.Equal(t, "move_to_binder", decodeError(t, resp.Body.Bytes()).Code)
	assert.Equal(t, map[string]int{cards[0].String(): 1}, entryQuantities(t, api, owner, source))
	assert.Empty(t, entryQuantities(t, api, owner, destination))
	requireEntriesMatchSlots(t, api, source)
	assert.Equal(t, []string{"card", "empty", "empty", "empty"}, slotStates(getPage(t, api, owner, source, 1)))
}

// slotStates lists a page's Slot states row by row, so each index is one Slot.
func slotStates(env binderPageEnvelope) []string {
	states := make([]string, 0, len(env.Data.Slots))
	for _, slot := range env.Data.Slots {
		states = append(states, slot.State)
	}
	return states
}

// TestMoveRefusesBadRequests: the same Collection on both sides, a Card named
// twice, and a Card the source does not hold all refuse the Move. The last
// one refuses it whole, so the Card that is held does not move either.
func TestMoveRefusesBadRequests(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	cards := newTestCards(t, 2)
	source := newPlainCollection(t, api, owner, 0)
	destination := newPlainCollection(t, api, owner, 0)
	addEntry(t, api, owner, source, cards[0], 1)

	resp := moveEntries(api, owner, source, source, cards[0])
	assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())

	resp = moveEntries(api, owner, source, destination, cards[0], cards[0])
	assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())

	resp = moveEntries(api, owner, source, destination, cards[0], cards[1])
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	assert.Equal(t, map[string]int{cards[0].String(): 1}, entryQuantities(t, api, owner, source))
	assert.Empty(t, entryQuantities(t, api, owner, destination))
}

// TestMoveRefusesAnotherProfilesCollection: a Collection that is not the
// caller's is "not found" on either side, and nothing moves.
func TestMoveRefusesAnotherProfilesCollection(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	other := api.newUserToken(t)
	cards := newTestCards(t, 1)
	source := newPlainCollection(t, api, owner, 0)
	destination := newPlainCollection(t, api, owner, 0)
	otherSource := newPlainCollection(t, api, other, 0)
	addEntry(t, api, owner, source, cards[0], 1)

	resp := moveEntries(api, owner, source, otherSource, cards[0])
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	resp = moveEntries(api, other, source, destination, cards[0])
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	resp = moveEntries(api, owner, otherSource, destination, cards[0])
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	assert.Equal(t, map[string]int{cards[0].String(): 1}, entryQuantities(t, api, owner, source))
	assert.Empty(t, entryQuantities(t, api, owner, destination))
}
