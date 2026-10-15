package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type binderMoveEnvelope struct {
	Data struct {
		From binderPageBody `json:"from"`
		To   binderPageBody `json:"to"`
	} `json:"data"`
}

func moveSlotAt(api testAPI, token, id string, fromPage, fromRow, fromColumn, toPage, toRow, toColumn int) *httptest.ResponseRecorder {
	target := func(page, row, column int) map[string]any {
		return map[string]any{"page": page, "row": row, "column": column}
	}
	return api.Post(binderPath(id)+"/slots/move", bearer(token), map[string]any{
		"from": target(fromPage, fromRow, fromColumn),
		"to":   target(toPage, toRow, toColumn),
	})
}

func decodeBinderMove(t *testing.T, resp *httptest.ResponseRecorder) binderMoveEnvelope {
	t.Helper()
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var env binderMoveEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &env))
	return env
}

// slotState reads one Slot's state and card id from a Page read.
func slotState(page binderPageBody, row, column int) (state, cardID string) {
	for _, s := range page.Slots {
		if s.Row == row && s.Column == column {
			if s.Card != nil {
				cardID = s.Card.ID
			}
			return s.State, cardID
		}
	}
	return "", ""
}

func TestBinderMoveToEmptySlotOnAnotherPage(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	card := newTestCards(t, 1)[0]
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, card))

	env := decodeBinderMove(t, moveSlotAt(api, owner, id, 1, 1, 1, 2, 2, 2))
	assert.Equal(t, 1, env.Data.From.Page)
	assert.Equal(t, 2, env.Data.To.Page)
	state, _ := slotState(env.Data.From, 1, 1)
	assert.Equal(t, "empty", state, "the source Slot is empty after the move")
	state, cardID := slotState(env.Data.To, 2, 2)
	assert.Equal(t, "card", state)
	assert.Equal(t, card.String(), cardID)

	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, card), "a move changes no quantity")
}

func TestBinderMoveSwapsWithFilledSlot(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	cards := newTestCards(t, 2)
	a, b := cards[0], cards[1]
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, a))
	decodeBinderPage(t, placeCardAt(api, owner, id, 2, 1, 1, b))
	requireEntriesMatchSlots(t, api, id)

	env := decodeBinderMove(t, moveSlotAt(api, owner, id, 1, 1, 1, 2, 1, 1))
	_, cardID := slotState(env.Data.From, 1, 1)
	assert.Equal(t, b.String(), cardID, "the source now holds the card that was on the target")
	_, cardID = slotState(env.Data.To, 1, 1)
	assert.Equal(t, a.String(), cardID, "the target now holds the moved card")

	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, a))
	assert.Equal(t, 1, masterQuantity(t, api, owner, b))
}

func TestBinderMoveSwapsUnidentifiedLikeAnySlot(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	card := newTestCards(t, 1)[0]
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, card))
	decodeBinderPage(t, markUnidentifiedAt(api, owner, id, 1, 1, 2))
	requireEntriesMatchSlots(t, api, id)

	// The card swaps with the Unidentified Slot beside it on the same Page.
	env := decodeBinderMove(t, moveSlotAt(api, owner, id, 1, 1, 1, 1, 1, 2))
	state, _ := slotState(env.Data.From, 1, 1)
	assert.Equal(t, "unidentified", state)
	state, cardID := slotState(env.Data.To, 1, 2)
	assert.Equal(t, "card", state)
	assert.Equal(t, card.String(), cardID)
	assert.Equal(t, 1, env.Data.To.UnidentifiedCount, "the Unidentified mark moved, it was not lost")
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, card))
}

func TestBinderMoveUnidentifiedToEmptySlot(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	decodeBinderPage(t, markUnidentifiedAt(api, owner, id, 1, 1, 1))

	env := decodeBinderMove(t, moveSlotAt(api, owner, id, 1, 1, 1, 2, 2, 1))
	state, _ := slotState(env.Data.To, 2, 1)
	assert.Equal(t, "unidentified", state)
	assert.Equal(t, 1, env.Data.To.UnidentifiedCount)
	state, _ = slotState(env.Data.From, 1, 1)
	assert.Equal(t, "empty", state, "the source Slot is empty after the move")
	requireEntriesMatchSlots(t, api, id)
}

func TestBinderMoveOntoItselfChangesNothing(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	card := newTestCards(t, 1)[0]
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, card))

	env := decodeBinderMove(t, moveSlotAt(api, owner, id, 1, 1, 1, 1, 1, 1))
	state, cardID := slotState(env.Data.To, 1, 1)
	assert.Equal(t, "card", state)
	assert.Equal(t, card.String(), cardID)
	assert.Equal(t, 1, env.Data.To.CardTotal)
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, card))
}

func TestBinderMoveKeepsEntryWhenOneOfTwoCopiesMoves(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	card := newTestCards(t, 1)[0]
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, card))
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 2, card))
	requireEntriesMatchSlots(t, api, id)

	decodeBinderMove(t, moveSlotAt(api, owner, id, 1, 1, 2, 2, 1, 1))
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 2, masterQuantity(t, api, owner, card), "one copy moved, two remain")
}

func TestBinderMoveRefusals(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	card := newTestCards(t, 1)[0]
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, card))

	// An empty source has nothing to move.
	resp := moveSlotAt(api, owner, id, 1, 2, 2, 2, 1, 1)
	assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())

	// A target outside the Layout's rows and columns.
	resp = moveSlotAt(api, owner, id, 1, 1, 1, 1, 3, 1)
	assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
	resp = moveSlotAt(api, owner, id, 1, 1, 1, 1, 1, 3)
	assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())

	// A Page past the last Leaf.
	resp = moveSlotAt(api, owner, id, 1, 1, 1, 3, 1, 1)
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	// A Slot in another user's binder cannot be reached.
	other := api.newUserToken(t)
	resp = moveSlotAt(api, other, id, 1, 1, 1, 1, 1, 2)
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	// A plain Collection has no Pages.
	plain := api.Post("/collections", bearer(owner), map[string]any{"title": "Box"})
	require.Equal(t, http.StatusCreated, plain.Code, plain.Body.String())
	var box collectionEnvelope
	require.NoError(t, json.Unmarshal(plain.Body.Bytes(), &box))
	resp = moveSlotAt(api, owner, box.Data.ID, 1, 1, 1, 1, 1, 2)
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	resp = moveSlotAt(api, "", id, 1, 1, 1, 1, 1, 2)
	assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())

	// Every refusal left the binder as it was.
	page := getPage(t, api, owner, id, 1)
	state, cardID := slotState(page.Data, 1, 1)
	assert.Equal(t, "card", state)
	assert.Equal(t, card.String(), cardID)
	assert.Equal(t, 1, page.Data.CardTotal)
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, card))
}

func TestBinderMoveRefusesSourceOutsideTheBinder(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	card := newTestCards(t, 1)[0]
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, card))

	// A source row past the Layout's rows.
	resp := moveSlotAt(api, owner, id, 1, 3, 1, 1, 1, 2)
	assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())

	// A source on a Page past the last Leaf.
	resp = moveSlotAt(api, owner, id, 3, 1, 1, 1, 1, 2)
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	page := getPage(t, api, owner, id, 1)
	state, cardID := slotState(page.Data, 1, 1)
	assert.Equal(t, "card", state)
	assert.Equal(t, card.String(), cardID)
	requireEntriesMatchSlots(t, api, id)
}
