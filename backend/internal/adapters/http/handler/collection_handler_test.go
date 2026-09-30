package handler

import (
	"math"
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
	"github.com/itsLeonB/ungerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// newTestCollectionHandler registers the routes behind a stub guard that
// stashes a fixed profile ID, standing in for SessionGuard (which needs a
// real AuthKit session). With guarded false, no claims are stashed.
func newTestCollectionHandler(t *testing.T, guarded bool) (*mocks.MockCollectionService, humatest.TestAPI, uuid.UUID) {
	t.Helper()

	svc := mocks.NewMockCollectionService(t)
	profileID := uuid.New()
	_, api := humatest.New(t, httpapi.NewConfig())

	var mw []func(huma.Context, func(huma.Context))
	if guarded {
		mw = append(mw, func(ctx huma.Context, next func(huma.Context)) {
			next(authpkg.WithClaims(ctx, "user", "session", "e@example.com", profileID.String()))
		})
	}
	endpoint.RegisterAll(api, NewCollectionHandler(svc).Routes(), mw...)

	return svc, api, profileID
}

func TestCollectionHandler_MissingSession(t *testing.T) {
	_, api, _ := newTestCollectionHandler(t, false)

	resp := api.Get("/collections")
	assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
}

func TestCollectionHandler_BlankTitle(t *testing.T) {
	_, api, _ := newTestCollectionHandler(t, true)

	resp := api.Post("/collections", map[string]any{"title": " \t "})
	assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())

	resp = api.Put("/collections/"+uuid.NewString(), map[string]any{"title": " "})
	assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
}

func TestCollectionHandler_InvalidCollectionID(t *testing.T) {
	_, api, _ := newTestCollectionHandler(t, true)

	assert.Equal(t, http.StatusUnprocessableEntity, api.Get("/collections/nope").Code)
	assert.Equal(t, http.StatusUnprocessableEntity, api.Put("/collections/nope", map[string]any{"title": "T"}).Code)
	assert.Equal(t, http.StatusUnprocessableEntity, api.Delete("/collections/nope").Code)
}

func TestCollectionHandler_MaxCardCountOverflow(t *testing.T) {
	_, api, _ := newTestCollectionHandler(t, true)

	resp := api.Post("/collections", map[string]any{"title": "T", "maxCardCount": math.MaxInt32 + 1})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
}

func TestCollectionHandler_Create(t *testing.T) {
	svc, api, profileID := newTestCollectionHandler(t, true)
	limit := 10
	svc.EXPECT().
		Create(mock.Anything, dto.CollectionRequest{ProfileID: profileID, Title: "Binder", Description: "d", MaxCardCount: limit}).
		Return(dto.CollectionSummary{Title: "Binder"}, nil)

	resp := api.Post("/collections", map[string]any{"title": "Binder", "description": "d", "maxCardCount": limit})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	assert.Contains(t, resp.Body.String(), `"title":"Binder"`)
}

func TestCollectionHandler_List(t *testing.T) {
	svc, api, profileID := newTestCollectionHandler(t, true)
	svc.EXPECT().List(mock.Anything, dto.CollectionListRequest{ProfileID: profileID}).Return([]dto.CollectionSummary{{Title: "A"}}, nil)

	resp := api.Get("/collections")
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.Contains(t, resp.Body.String(), `"title":"A"`)
}

func TestCollectionHandler_NotFoundPassesThrough(t *testing.T) {
	svc, api, profileID := newTestCollectionHandler(t, true)
	id := uuid.New()
	svc.EXPECT().Get(mock.Anything, dto.CollectionLookup{ProfileID: profileID, ID: id}).Return(dto.CollectionSummary{}, ungerr.NotFoundError("collection not found"))
	svc.EXPECT().Update(mock.Anything, dto.CollectionRequest{ProfileID: profileID, ID: id, Title: "T"}).Return(dto.CollectionSummary{}, ungerr.NotFoundError("collection not found"))
	svc.EXPECT().Delete(mock.Anything, dto.CollectionLookup{ProfileID: profileID, ID: id}).Return(ungerr.NotFoundError("collection not found"))

	path := "/collections/" + id.String()
	assert.Equal(t, http.StatusNotFound, api.Get(path).Code)
	assert.Equal(t, http.StatusNotFound, api.Put(path, map[string]any{"title": "T"}).Code)
	assert.Equal(t, http.StatusNotFound, api.Delete(path).Code)
}

func TestCollectionHandler_DeleteSuccess(t *testing.T) {
	svc, api, profileID := newTestCollectionHandler(t, true)
	id := uuid.New()
	svc.EXPECT().Delete(mock.Anything, dto.CollectionLookup{ProfileID: profileID, ID: id}).Return(nil)

	resp := api.Delete("/collections/" + id.String())
	assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
}
