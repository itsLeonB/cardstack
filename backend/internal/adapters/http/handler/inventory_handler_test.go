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
		Return([]dto.InventoryChangeResult{{CardID: cardID, Status: "declined", Reason: "capacity_exceeded", Message: "m"}}, nil)

	resp := api.Patch("/collections/"+collectionID.String()+"/entries", map[string]any{"items": []map[string]any{{"cardId": cardID, "quantity": 0}}})
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.Contains(t, resp.Body.String(), `"reason":"capacity_exceeded"`)
}

func TestInventoryHandler_BulkUpdate_Validation(t *testing.T) {
	_, api, _ := newTestInventoryHandler(t, true)
	path := "/collections/" + uuid.NewString() + "/entries"
	item := func() map[string]any { return map[string]any{"cardId": uuid.New(), "quantity": 1} }

	over := make([]map[string]any, 101)
	for i := range over {
		over[i] = item()
	}

	bodies := map[string]any{
		"empty list":  map[string]any{"items": []any{}},
		"no items":    map[string]any{},
		"oversized":   map[string]any{"items": over},
		"negative":    map[string]any{"items": []map[string]any{{"cardId": uuid.New(), "quantity": -1}}},
		"bad card id": map[string]any{"items": []map[string]any{{"cardId": "nope", "quantity": 1}}},
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, http.StatusUnprocessableEntity, api.Patch(path, body).Code)
		})
	}
}

func TestInventoryHandler_BulkUpdate_MissingSession(t *testing.T) {
	_, api, _ := newTestInventoryHandler(t, false)

	resp := api.Patch("/collections/"+uuid.NewString()+"/entries", map[string]any{"items": []map[string]any{{"cardId": uuid.New(), "quantity": 1}}})
	assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
}
