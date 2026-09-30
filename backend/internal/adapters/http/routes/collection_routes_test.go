package routes

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
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
	if regResp.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", regResp.Code, regResp.Body.String())
	}

	loginResp := api.Post("/auth/login", map[string]string{"email": email, "password": password})
	if loginResp.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", loginResp.Code, loginResp.Body.String())
	}

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
	if resp := api.Get("/collections"); resp.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated list: expected 401, got %d: %s", resp.Code, resp.Body.String())
	}

	limit := 100
	createResp := api.Post("/collections", cookieHeader(ownerCookies), map[string]any{
		"title":        "Base Set Binder",
		"description":  "My original cards",
		"maxCardCount": limit,
	})
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", createResp.Code, createResp.Body.String())
	}
	var created collectionEnvelope
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding create response: %v", err)
	}
	if created.Data.Title != "Base Set Binder" || created.Data.MaxCardCount == nil || *created.Data.MaxCardCount != limit {
		t.Fatalf("unexpected create response: %+v", created.Data)
	}
	id := created.Data.ID

	// The owner's list includes it.
	listResp := api.Get("/collections", cookieHeader(ownerCookies))
	if listResp.Code != http.StatusOK {
		t.Fatalf("owner list: expected 200, got %d: %s", listResp.Code, listResp.Body.String())
	}
	var ownerList collectionListEnvelope
	if err := json.Unmarshal(listResp.Body.Bytes(), &ownerList); err != nil {
		t.Fatalf("decoding owner list: %v", err)
	}
	found := false
	for _, c := range ownerList.Data {
		if c.ID == id {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected owner's list to include %s, got %+v", id, ownerList.Data)
	}

	// The other user's list does not include it.
	otherListResp := api.Get("/collections", cookieHeader(otherCookies))
	if otherListResp.Code != http.StatusOK {
		t.Fatalf("other user list: expected 200, got %d: %s", otherListResp.Code, otherListResp.Body.String())
	}
	var otherList collectionListEnvelope
	if err := json.Unmarshal(otherListResp.Body.Bytes(), &otherList); err != nil {
		t.Fatalf("decoding other user list: %v", err)
	}
	for _, c := range otherList.Data {
		if c.ID == id {
			t.Fatalf("expected other user's list to exclude %s, got %+v", id, otherList.Data)
		}
	}

	// The owner can view it.
	if resp := api.Get("/collections/"+id, cookieHeader(ownerCookies)); resp.Code != http.StatusOK {
		t.Fatalf("owner get: expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	// The other user gets 404, not 403: existence isn't leaked.
	if resp := api.Get("/collections/"+id, cookieHeader(otherCookies)); resp.Code != http.StatusNotFound {
		t.Fatalf("other user get: expected 404, got %d: %s", resp.Code, resp.Body.String())
	}

	// The owner can edit it.
	updateResp := api.Put("/collections/"+id, cookieHeader(ownerCookies), map[string]any{
		"title":       "Renamed Binder",
		"description": "updated description",
	})
	if updateResp.Code != http.StatusOK {
		t.Fatalf("owner update: expected 200, got %d: %s", updateResp.Code, updateResp.Body.String())
	}
	var updated collectionEnvelope
	if err := json.Unmarshal(updateResp.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decoding update response: %v", err)
	}
	if updated.Data.Title != "Renamed Binder" || updated.Data.Description != "updated description" || updated.Data.MaxCardCount != nil {
		t.Fatalf("expected the limit to be cleared by the update, got %+v", updated.Data)
	}

	// The owner can delete it (hard delete, no undo).
	if resp := api.Delete("/collections/"+id, cookieHeader(ownerCookies)); resp.Code != http.StatusNoContent {
		t.Fatalf("owner delete: expected 204, got %d: %s", resp.Code, resp.Body.String())
	}

	// It's gone for good.
	if resp := api.Get("/collections/"+id, cookieHeader(ownerCookies)); resp.Code != http.StatusNotFound {
		t.Fatalf("get after delete: expected 404, got %d: %s", resp.Code, resp.Body.String())
	}
}
