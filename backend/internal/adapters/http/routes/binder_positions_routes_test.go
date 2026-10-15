package routes

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type entryListEnvelope struct {
	Data []struct {
		Card      struct{ ID string }      `json:"card"`
		Quantity  int                      `json:"quantity"`
		Positions []dto.BinderSlotPosition `json:"positions"`
	} `json:"data"`
}

func listEntryRows(t *testing.T, api testAPI, token, id string) entryListEnvelope {
	t.Helper()
	resp := api.Get("/collections/"+id+"/entries", bearer(token))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var out entryListEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
	return out
}

func entryRow(t *testing.T, rows entryListEnvelope, card uuid.UUID) (quantity int, positions []dto.BinderSlotPosition) {
	t.Helper()
	for _, row := range rows.Data {
		if row.Card.ID == card.String() {
			return row.Quantity, row.Positions
		}
	}
	require.FailNow(t, "card missing from the entry list", card.String())
	return 0, nil
}

// TestBinderEntryListPositions: a Card lists every Slot it holds as a Page,
// row and column, an Unidentified Slot lists under no Card, and adding a Leaf
// before the first one renumbers the positions with the Pages.
func TestBinderEntryListPositions(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 1, 2, 2)
	cards := newTestCards(t, 2)
	a, b := cards[0], cards[1]

	// Pages 1 and 2 are the first Leaf; 3 and 4 the second.
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, a))
	decodeBinderPage(t, placeCardAt(api, owner, id, 2, 1, 1, a))
	decodeBinderPage(t, placeCardAt(api, owner, id, 2, 1, 2, b))
	decodeBinderPage(t, markUnidentifiedAt(api, owner, id, 4, 1, 2))

	rows := listEntryRows(t, api, owner, id)
	quantity, positions := entryRow(t, rows, a)
	assert.Equal(t, 2, quantity)
	assert.Equal(t, []dto.BinderSlotPosition{{Page: 1, Row: 1, Column: 1}, {Page: 2, Row: 1, Column: 1}}, positions)
	_, positions = entryRow(t, rows, b)
	assert.Equal(t, []dto.BinderSlotPosition{{Page: 2, Row: 1, Column: 2}}, positions)
	assert.Len(t, rows.Data, 2, "the Unidentified Slot adds no entry")

	// A Leaf before the first one moves every existing Page two numbers on.
	resp := api.Post(binderPath(id)+"/leaves", bearer(owner), map[string]any{"leaf": 1, "placement": "before"})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

	rows = listEntryRows(t, api, owner, id)
	_, positions = entryRow(t, rows, a)
	assert.Equal(t, []dto.BinderSlotPosition{{Page: 3, Row: 1, Column: 1}, {Page: 4, Row: 1, Column: 1}}, positions)
	_, positions = entryRow(t, rows, b)
	assert.Equal(t, []dto.BinderSlotPosition{{Page: 4, Row: 1, Column: 2}}, positions)
}

// TestPlainCollectionEntryListHasNoPositions: a Collection without a Layout
// keeps its entry list as it was, with no positions field at all.
func TestPlainCollectionEntryListHasNoPositions(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	card := newTestCards(t, 1)[0]

	resp := api.Post("/collections/"+id+"/entries", bearer(owner), map[string]any{"cardId": card, "quantity": 2})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

	list := api.Get("/collections/"+id+"/entries", bearer(owner))
	require.Equal(t, http.StatusOK, list.Code, list.Body.String())
	assert.NotContains(t, list.Body.String(), "positions")

	quantity, positions := entryRow(t, listEntryRows(t, api, owner, id), card)
	assert.Equal(t, 2, quantity)
	assert.Empty(t, positions)
}
