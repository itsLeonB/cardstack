package handler

import (
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestInventoryHandler(t *testing.T, guarded bool) (*mocks.MockInventoryService, humatest.TestAPI, uuid.UUID) {
	t.Helper()

	svc := mocks.NewMockInventoryService(t)
	profileID := uuid.New()
	_, api := humatest.New(t, httpapi.NewConfig())

	var mw []func(huma.Context, func(huma.Context))
	if guarded {
		mw = append(mw, func(ctx huma.Context, next func(huma.Context)) {
			next(authpkg.WithClaims(ctx, "user", "session", "e@example.com", profileID.String()))
		})
	}
	endpoint.RegisterAll(api, NewInventoryHandler(svc).Routes(), mw...)

	return svc, api, profileID
}

func TestInventoryHandler_BulkUpdate(t *testing.T) {
	svc, api, profileID := newTestInventoryHandler(t, true)
	collectionID, cardID := uuid.New(), uuid.New()
	svc.EXPECT().
		BulkUpdate(mock.Anything, dto.InventoryBulkUpdateRequest{
			ProfileID: profileID, CollectionID: collectionID,
			Items: []dto.InventoryQuantityChange{{CardID: cardID, Quantity: 0}},
		}).
		Return([]dto.InventoryChangeResult{{CardID: cardID, Status: dto.InventoryStatusDeclined, Reason: dto.InventoryReasonCapacityExceeded, Message: "m"}}, nil)

	resp := api.Patch("/collections/"+collectionID.String()+"/entries", map[string]any{"items": []map[string]any{{"cardId": cardID, "quantity": 0}}})
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.Contains(t, resp.Body.String(), `"reason":"`+dto.InventoryReasonCapacityExceeded+`"`)
}

// TestInventoryHandler_BulkUpdate_BatchBounds pins the documented 1..100 item
// bounds; other schema rules are Huma's own.
func TestInventoryHandler_BulkUpdate_BatchBounds(t *testing.T) {
	_, api, _ := newTestInventoryHandler(t, true)
	path := "/collections/" + uuid.NewString() + "/entries"

	over := make([]map[string]any, 101)
	for i := range over {
		over[i] = map[string]any{"cardId": uuid.New(), "quantity": 1}
	}

	assert.Equal(t, http.StatusUnprocessableEntity, api.Patch(path, map[string]any{"items": []any{}}).Code)
	assert.Equal(t, http.StatusUnprocessableEntity, api.Patch(path, map[string]any{"items": over}).Code)
}

func TestInventoryHandler_BulkUpdate_MissingSession(t *testing.T) {
	_, api, _ := newTestInventoryHandler(t, false)

	resp := api.Patch("/collections/"+uuid.NewString()+"/entries", map[string]any{"items": []map[string]any{{"cardId": uuid.New(), "quantity": 1}}})
	assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
}

func TestInventoryHandler_List_CardIDs(t *testing.T) {
	svc, api, profileID := newTestInventoryHandler(t, true)
	collectionID, a, b := uuid.New(), uuid.New(), uuid.New()
	svc.EXPECT().
		List(mock.Anything, dto.InventoryListRequest{
			ProfileID: profileID, CollectionID: collectionID,
			Filter: dto.CardFilter{CardIDs: []uuid.UUID{a, b}, Page: 1, Limit: 24},
		}).
		Return([]dto.InventoryItem{}, dto.PaginationMeta{Page: 1, Limit: 24}, nil)

	resp := api.Get("/collections/" + collectionID.String() + "/entries?cardId=" + a.String() + "&cardId=" + b.String())
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
}

func TestInventoryHandler_List_CardIDs_Invalid(t *testing.T) {
	_, api, _ := newTestInventoryHandler(t, true)
	path := "/collections/" + uuid.NewString() + "/entries"

	assert.Equal(t, http.StatusBadRequest, api.Get(path+"?cardId=nope").Code)

	q := ""
	for i := 0; i < 101; i++ {
		q += "&cardId=" + uuid.NewString()
	}
	assert.Equal(t, http.StatusUnprocessableEntity, api.Get(path+"?"+q[1:]).Code)
}

func TestInventoryHandler_ListCardHoldings(t *testing.T) {
	svc, api, profileID := newTestInventoryHandler(t, true)
	cardID, collectionID := uuid.New(), uuid.New()
	svc.EXPECT().
		ListCardHoldings(mock.Anything, dto.CardHoldingsRequest{ProfileID: profileID, CardID: cardID}).
		Return([]dto.CardHolding{{Collection: dto.HoldingCollection{ID: collectionID, Name: "Binder"}, Quantity: 2}}, nil)

	resp := api.Get("/inventory/cards/" + cardID.String() + "/holdings")
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.JSONEq(t, `{"data":[{"collection":{"id":"`+collectionID.String()+`","name":"Binder"},"quantity":2}]}`, resp.Body.String())
}

func TestInventoryHandler_ListCardHoldings_MissingSession(t *testing.T) {
	_, api, _ := newTestInventoryHandler(t, false)

	assert.Equal(t, http.StatusUnauthorized, api.Get("/inventory/cards/"+uuid.NewString()+"/holdings").Code)
}

func TestInventoryHandler_ListMaster(t *testing.T) {
	svc, api, profileID := newTestInventoryHandler(t, true)
	svc.EXPECT().
		ListMasterInventory(mock.Anything, dto.MasterInventoryRequest{ProfileID: profileID, Filter: dto.CardFilter{Page: 2, Limit: 5}}).
		Return([]dto.InventoryItem{{Quantity: 7}}, dto.PaginationMeta{Total: 6, Page: 2, Limit: 5}, nil)

	resp := api.Get("/inventory/cards?page=2&limit=5")
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.Contains(t, resp.Body.String(), `"quantity":7`)
	assert.Contains(t, resp.Body.String(), `"meta":{"total":6,"page":2,"limit":5}`)
}

func TestInventoryHandler_ListMaster_MissingSession(t *testing.T) {
	_, api, _ := newTestInventoryHandler(t, false)

	assert.Equal(t, http.StatusUnauthorized, api.Get("/inventory/cards").Code)
}

func TestInventoryHandler_ListMaster_Filters(t *testing.T) {
	svc, api, profileID := newTestInventoryHandler(t, true)
	cardID := uuid.New()
	svc.EXPECT().
		ListMasterInventory(mock.Anything, dto.MasterInventoryRequest{
			ProfileID: profileID,
			Filter:    dto.CardFilter{Name: "pika", Categories: []string{"Pokemon"}, CardIDs: []uuid.UUID{cardID}, Page: 1, Limit: 24},
		}).
		Return([]dto.InventoryItem{}, dto.PaginationMeta{Page: 1, Limit: 24}, nil)

	resp := api.Get("/inventory/cards?name=pika&category=Pokemon&cardId=" + cardID.String())
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.Equal(t, http.StatusBadRequest, api.Get("/inventory/cards?cardId=nope").Code)
}

func TestInventoryHandler_ListMasterFacets(t *testing.T) {
	svc, api, profileID := newTestInventoryHandler(t, true)
	svc.EXPECT().
		ListMasterFacets(mock.Anything, dto.MasterInventoryRequest{ProfileID: profileID, Filter: dto.CardFilter{Name: "pika"}}).
		Return(dto.CatalogFacets{}, nil)

	resp := api.Get("/inventory/cards/facets?name=pika")
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
}

func TestInventoryHandler_ListMasterFacets_MissingSession(t *testing.T) {
	_, api, _ := newTestInventoryHandler(t, false)

	assert.Equal(t, http.StatusUnauthorized, api.Get("/inventory/cards/facets").Code)
}
