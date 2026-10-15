package routes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func slotPath(id string, page, row, column int) string {
	return binderPath(id) + "/pages/" + strconv.Itoa(page) + "/slots/" + strconv.Itoa(row) + "/" + strconv.Itoa(column)
}

func placeCardAt(api testAPI, token, id string, page, row, column int, card uuid.UUID) *httptest.ResponseRecorder {
	return api.Put(slotPath(id, page, row, column), bearer(token), map[string]any{"cardId": card})
}

func markUnidentifiedAt(api testAPI, token, id string, page, row, column int) *httptest.ResponseRecorder {
	return api.Put(slotPath(id, page, row, column)+"/unidentified", bearer(token))
}

func clearSlotAt(api testAPI, token, id string, page, row, column int) *httptest.ResponseRecorder {
	return api.Delete(slotPath(id, page, row, column), bearer(token))
}

func decodeBinderPage(t *testing.T, resp *httptest.ResponseRecorder) binderPageEnvelope {
	t.Helper()
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var env binderPageEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &env))
	return env
}

// requireEntriesMatchSlots is the quantity invariant (ADR-0020): every Card's
// Inventory Entry equals the number of its Slots in the Collection, and a Card
// with no Slots has no entry. It reads the database, so it also catches a write
// the HTTP response does not show.
func requireEntriesMatchSlots(t *testing.T, api testAPI, collectionID string) {
	t.Helper()
	cid := uuid.MustParse(collectionID)

	var counts []struct {
		CardID    uuid.UUID
		SlotCount int
	}
	require.NoError(t, api.db.Model(&entity.BinderSlot{}).
		Select("card_id, COUNT(*) AS slot_count").
		Where("collection_id = ? AND card_id IS NOT NULL", cid).
		Group("card_id").
		Scan(&counts).Error)

	var entries []entity.InventoryEntry
	require.NoError(t, api.db.Where("collection_id = ?", cid).Find(&entries).Error)

	want := map[uuid.UUID]int{}
	for _, c := range counts {
		want[c.CardID] = c.SlotCount
	}
	got := map[uuid.UUID]int{}
	for _, e := range entries {
		got[e.CardID] = e.Quantity
	}
	assert.Equal(t, want, got, "each Inventory Entry must equal the Card's Slot count")
}

// masterQuantity is the user's Master Inventory quantity for one Card, summed
// over all their Collections; zero when the Card is absent.
func masterQuantity(t *testing.T, api testAPI, token string, card uuid.UUID) int {
	t.Helper()
	resp := api.Get("/inventory/cards", bearer(token))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var out struct {
		Data []struct {
			Card     struct{ ID string } `json:"card"`
			Quantity int                 `json:"quantity"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
	for _, row := range out.Data {
		if row.Card.ID == card.String() {
			return row.Quantity
		}
	}
	return 0
}

func TestBinderSlotWritesKeepEntriesInStep(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	cards := newTestCards(t, 2)
	a, b := cards[0], cards[1]

	resp := placeCardAt(api, owner, id, 1, 1, 1, a)
	page := decodeBinderPage(t, resp)
	assert.Equal(t, "card", page.Data.Slots[0].State)
	require.NotNil(t, page.Data.Slots[0].Card)
	assert.Equal(t, a.String(), page.Data.Slots[0].Card.ID)
	assert.Equal(t, 1, page.Data.CardTotal)
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, a))

	// A second copy takes a second Slot, so the quantity rises to two.
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 2, a))
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 2, masterQuantity(t, api, owner, a))

	// Placing a Card on a filled Slot replaces the one it held.
	page = decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, b))
	assert.Equal(t, b.String(), page.Data.Slots[0].Card.ID)
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, a))
	assert.Equal(t, 1, masterQuantity(t, api, owner, b))

	// Unidentified counts toward the binder total but to no Card, so A's
	// last Slot leaves its entry removed and Master Inventory drops it.
	page = decodeBinderPage(t, markUnidentifiedAt(api, owner, id, 1, 1, 2))
	assert.Equal(t, "unidentified", page.Data.Slots[1].State)
	assert.Equal(t, 2, page.Data.CardTotal)
	assert.Equal(t, 1, page.Data.UnidentifiedCount)
	requireEntriesMatchSlots(t, api, id)
	assert.Zero(t, masterQuantity(t, api, owner, a))

	// An Unidentified Slot can be given a Card.
	page = decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 2, b))
	assert.Equal(t, "card", page.Data.Slots[1].State)
	assert.Zero(t, page.Data.UnidentifiedCount)
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 2, masterQuantity(t, api, owner, b))

	page = decodeBinderPage(t, clearSlotAt(api, owner, id, 1, 1, 1))
	assert.Equal(t, "empty", page.Data.Slots[0].State)
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, b))

	// Clearing an empty Slot changes nothing.
	decodeBinderPage(t, clearSlotAt(api, owner, id, 1, 2, 2))
	requireEntriesMatchSlots(t, api, id)

	page = decodeBinderPage(t, clearSlotAt(api, owner, id, 1, 1, 2))
	assert.Zero(t, page.Data.CardTotal)
	requireEntriesMatchSlots(t, api, id)
	assert.Zero(t, masterQuantity(t, api, owner, b))
}

func TestBinderSlotPlacingSameCardTwiceIsIdempotent(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	card := newTestCards(t, 1)[0]

	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, card))
	requireEntriesMatchSlots(t, api, id)

	page := decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, card))
	assert.Equal(t, card.String(), page.Data.Slots[0].Card.ID)
	assert.Equal(t, 1, page.Data.CardTotal, "placing into an occupied Slot must not add a Slot")
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, card))
}

func TestBinderSlotReplacingLastSlotMovesEntry(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	cards := newTestCards(t, 2)
	a, b := cards[0], cards[1]

	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, a))
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, a))

	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, b))
	requireEntriesMatchSlots(t, api, id)
	assert.Zero(t, masterQuantity(t, api, owner, a), "A's entry goes when its last Slot is replaced")
	assert.Equal(t, 1, masterQuantity(t, api, owner, b), "B's entry is created by the replacement")
}

func TestBinderSlotUnidentifiedDropsOneOfTwoEntries(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	card := newTestCards(t, 1)[0]

	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, card))
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 2, card))
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 2, masterQuantity(t, api, owner, card))

	page := decodeBinderPage(t, markUnidentifiedAt(api, owner, id, 1, 1, 1))
	assert.Equal(t, 1, page.Data.UnidentifiedCount)
	assert.Equal(t, 2, page.Data.CardTotal, "the Unidentified Slot still counts toward the binder total")
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, card), "the entry drops from 2 to 1")
}

// TestBinderWriteSlotJoinsOpenTransaction proves WriteSlot does not commit on
// its own. A caller that opens a transaction, writes a Slot and then fails
// leaves nothing behind, which is how the move and Replace Page tickets reuse it.
func TestBinderWriteSlotJoinsOpenTransaction(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	card := newTestCards(t, 1)[0]

	var collection entity.Collection
	require.NoError(t, api.db.First(&collection, "id = ?", uuid.MustParse(id)).Error)
	binders := provider.ProvideBinderService(&provider.DataSources{Gorm: api.db}, mapper.NewImageHost(""))
	write := dto.BinderSlotWrite{
		ProfileID:    collection.ProfileID,
		CollectionID: collection.ID,
		Target:       dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		State:        dto.BinderSlotCard,
		CardID:       card,
	}
	transactor := crud.NewTransactor(api.db)

	errCallerFailed := errors.New("the caller's own step failed")
	err := transactor.WithinTransaction(context.Background(), func(ctx context.Context) error {
		if err := binders.WriteSlot(ctx, write); err != nil {
			return err
		}
		return errCallerFailed
	})
	require.ErrorIs(t, err, errCallerFailed)
	assert.Zero(t, getPage(t, api, owner, id, 1).Data.CardTotal, "a rolled-back caller must undo WriteSlot")
	requireEntriesMatchSlots(t, api, id)
	assert.Zero(t, masterQuantity(t, api, owner, card))

	err = transactor.WithinTransaction(context.Background(), func(ctx context.Context) error {
		return binders.WriteSlot(ctx, write)
	})
	require.NoError(t, err)
	assert.Equal(t, 1, getPage(t, api, owner, id, 1).Data.CardTotal)
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, card))
}

func TestBinderSlotWriteRefusals(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	card := newTestCards(t, 1)[0]

	resp := placeCardAt(api, owner, id, 1, 3, 1, card)
	assert.Equal(t, http.StatusBadRequest, resp.Code, "row 3 is outside a 2-row Layout: %s", resp.Body.String())
	resp = placeCardAt(api, owner, id, 1, 1, 3, card)
	assert.Equal(t, http.StatusBadRequest, resp.Code, "column 3 is outside a 2-column Layout: %s", resp.Body.String())
	resp = placeCardAt(api, owner, id, 1, 0, 1, card)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, "row 0 fails path validation: %s", resp.Body.String())

	resp = placeCardAt(api, owner, id, 3, 1, 1, card)
	assert.Equal(t, http.StatusNotFound, resp.Code, "page 3 is past the only Leaf: %s", resp.Body.String())

	// An unknown Card is refused and leaves the Slot and entries as they were.
	resp = placeCardAt(api, owner, id, 1, 1, 1, uuid.New())
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	resp = placeCardAt(api, owner, id, 1, 1, 1, uuid.Nil)
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	page := getPage(t, api, owner, id, 1)
	assert.Zero(t, page.Data.CardTotal)
	requireEntriesMatchSlots(t, api, id)

	resp = api.Put(slotPath(id, 1, 1, 1), bearer(owner), map[string]any{})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, "a missing cardId is rejected: %s", resp.Body.String())
}

func TestBinderSlotWriteOwnershipAndPlainCollection(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	other := api.newUserToken(t)
	id := newBinderCollection(t, api, owner)
	createBinder(t, api, owner, id, 2, 2, 1)
	card := newTestCards(t, 1)[0]
	decodeBinderPage(t, placeCardAt(api, owner, id, 1, 1, 1, card))

	resp := placeCardAt(api, other, id, 1, 1, 2, card)
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	resp = markUnidentifiedAt(api, other, id, 1, 1, 1)
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	resp = clearSlotAt(api, other, id, 1, 1, 1)
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	requireEntriesMatchSlots(t, api, id)
	assert.Equal(t, 1, masterQuantity(t, api, owner, card), "a refused write leaves the owner's entry as it was")
	assert.Zero(t, masterQuantity(t, api, other, card))
	page := getPage(t, api, owner, id, 1)
	assert.Equal(t, 1, page.Data.CardTotal)
	assert.Equal(t, "card", page.Data.Slots[0].State)

	resp = placeCardAt(api, "", id, 1, 1, 1, card)
	assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())

	plain := api.Post("/collections", bearer(owner), map[string]any{"title": "Box"})
	require.Equal(t, http.StatusCreated, plain.Code, plain.Body.String())
	var box collectionEnvelope
	require.NoError(t, json.Unmarshal(plain.Body.Bytes(), &box))
	resp = placeCardAt(api, owner, box.Data.ID, 1, 1, 1, card)
	assert.Equal(t, http.StatusNotFound, resp.Code, "a plain Collection has no Pages: %s", resp.Body.String())
	requireEntriesMatchSlots(t, api, box.Data.ID)
	assert.Equal(t, 1, masterQuantity(t, api, owner, card), "the refused plain-Collection write adds no entry")
}
