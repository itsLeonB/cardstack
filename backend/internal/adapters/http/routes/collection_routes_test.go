package routes

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type collectionEnvelope struct {
	Data struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		MaxCardCount *int   `json:"maxCardCount"`
	} `json:"data"`
}

type collectionListEnvelope struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// registerAndLogin registers and logs in a fresh test user, returning the
// session cookies a subsequent request can pass to cookieHeader.
func registerAndLogin(t *testing.T, api humatest.TestAPI, email, password string) []*http.Cookie {
	t.Helper()

	regResp := api.Post("/auth/register", map[string]string{
		"email":                email,
		"password":             password,
		"passwordConfirmation": password,
	})
	require.Equal(t, http.StatusCreated, regResp.Code, regResp.Body.String())

	loginResp := api.Post("/auth/login", map[string]string{"email": email, "password": password})
	require.Equal(t, http.StatusOK, loginResp.Code, loginResp.Body.String())

	return loginResp.Result().Cookies()
}

// TestCollectionsFlow covers the CRUD happy path plus the unauthenticated
// and cross-user failures; branch-level cases live in the unit tests.
func TestCollectionsFlow(t *testing.T) {
	services := authTestServices(t)
	_, api := humatest.New(t, httpapi.NewConfig())
	RegisterRoutes(api, services)

	ownerCookies := registerAndLogin(t, api, uuid.NewString()+"@example.com", "correct-horse-battery-staple")
	otherCookies := registerAndLogin(t, api, uuid.NewString()+"@example.com", "correct-horse-battery-staple")

	// Unauthenticated requests are rejected.
	resp := api.Get("/collections")
	assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())

	limit := 100
	createResp := api.Post("/collections", cookieHeader(ownerCookies), map[string]any{
		"title":        "Base Set Binder",
		"description":  "My original cards",
		"maxCardCount": limit,
	})
	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	var created collectionEnvelope
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &created))
	assert.Equal(t, "Base Set Binder", created.Data.Title)
	require.NotNil(t, created.Data.MaxCardCount)
	assert.Equal(t, limit, *created.Data.MaxCardCount)
	id := created.Data.ID

	// The owner's list includes it.
	listResp := api.Get("/collections", cookieHeader(ownerCookies))
	require.Equal(t, http.StatusOK, listResp.Code, listResp.Body.String())
	var ownerList collectionListEnvelope
	require.NoError(t, json.Unmarshal(listResp.Body.Bytes(), &ownerList))
	var ownerIDs []string
	for _, c := range ownerList.Data {
		ownerIDs = append(ownerIDs, c.ID)
	}
	assert.Contains(t, ownerIDs, id)

	// The other user's list does not include it.
	otherListResp := api.Get("/collections", cookieHeader(otherCookies))
	require.Equal(t, http.StatusOK, otherListResp.Code, otherListResp.Body.String())
	var otherList collectionListEnvelope
	require.NoError(t, json.Unmarshal(otherListResp.Body.Bytes(), &otherList))
	for _, c := range otherList.Data {
		assert.NotEqual(t, id, c.ID)
	}

	// The owner can view it.
	resp = api.Get("/collections/"+id, cookieHeader(ownerCookies))
	assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

	// The other user gets 404, not 403, on every verb: existence isn't leaked.
	resp = api.Get("/collections/"+id, cookieHeader(otherCookies))
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	resp = api.Put("/collections/"+id, cookieHeader(otherCookies), map[string]any{"title": "Hijacked"})
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	resp = api.Delete("/collections/"+id, cookieHeader(otherCookies))
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())

	// The owner can edit it.
	updateResp := api.Put("/collections/"+id, cookieHeader(ownerCookies), map[string]any{
		"title":       "Renamed Binder",
		"description": "updated description",
	})
	require.Equal(t, http.StatusOK, updateResp.Code, updateResp.Body.String())
	var updated collectionEnvelope
	require.NoError(t, json.Unmarshal(updateResp.Body.Bytes(), &updated))
	assert.Equal(t, "Renamed Binder", updated.Data.Title)
	assert.Equal(t, "updated description", updated.Data.Description)
	assert.Nil(t, updated.Data.MaxCardCount, "the update should clear the limit")

	// The owner can delete it (hard delete, no undo).
	resp = api.Delete("/collections/"+id, cookieHeader(ownerCookies))
	assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())

	// It's gone for good.
	resp = api.Get("/collections/"+id, cookieHeader(ownerCookies))
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
}
