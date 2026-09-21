package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	authkit "github.com/itsLeonB/go-authkit"
	"github.com/itsLeonB/go-authkit/authkittest"
)

// withImmediateVerification mirrors this project's production config (empty
// VerificationURL/ResetPasswordURL, see docs/adr/0004): without it,
// authkittest.NewKit's defaults would leave every registered user
// unverified, and Login rejects unverified users regardless of password.
func withImmediateVerification() authkittest.Option {
	return func(cfg *authkit.Config, _ *authkit.Deps) {
		cfg.VerificationURL = ""
		cfg.ResetPasswordURL = ""
	}
}

// newTestAuthHandler wires an AuthHandler against authkittest's in-memory
// mock stores rather than a real Postgres instance — the "service
// boundary" layer of the ticket 02 plan's testing decisions, exercising
// this handler's request parsing, error-mapping, and cookie/CSRF wiring in
// isolation from persistence. Repository behavior itself is covered
// separately by internal/adapters/repository's real-Postgres tests, and the
// full stack end to end by internal/adapters/http/routes's feature test.
func newTestAuthHandler(t *testing.T) (*AuthHandler, humatest.TestAPI) {
	t.Helper()

	config.Global = &config.Config{Auth: config.Auth{CookieSameSite: "Lax"}}

	kit := authkittest.NewKit(withImmediateVerification())
	t.Cleanup(func() { _ = kit.Shutdown() })

	h := NewAuthHandler(kit)
	_, api := humatest.New(t, httpapi.NewConfig())
	endpoint.RegisterAll(api, h.Routes())

	return h, api
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	_, api := newTestAuthHandler(t)

	body := map[string]string{
		"email":                "dup@example.com",
		"password":             "correct-horse-battery-staple",
		"passwordConfirmation": "correct-horse-battery-staple",
	}

	first := api.Post("/auth/register", body)
	if first.Code != http.StatusCreated {
		t.Fatalf("first register: expected 201, got %d: %s", first.Code, first.Body.String())
	}

	second := api.Post("/auth/register", body)
	if second.Code != http.StatusConflict {
		t.Fatalf("duplicate register: expected 409, got %d: %s", second.Code, second.Body.String())
	}
}

func TestAuthHandler_Register_PasswordMismatch(t *testing.T) {
	_, api := newTestAuthHandler(t)

	resp := api.Post("/auth/register", map[string]string{
		"email":                "mismatch@example.com",
		"password":             "correct-horse-battery-staple",
		"passwordConfirmation": "something-else",
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	_, api := newTestAuthHandler(t)

	registerResp := api.Post("/auth/register", map[string]string{
		"email":                "login@example.com",
		"password":             "correct-horse-battery-staple",
		"passwordConfirmation": "correct-horse-battery-staple",
	})
	if registerResp.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", registerResp.Code, registerResp.Body.String())
	}

	resp := api.Post("/auth/login", map[string]string{"email": "login@example.com", "password": "wrong"})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body.String())
	}

	var problem struct {
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decoding error body: %v", err)
	}
	if problem.Detail != authkit.ErrInvalidCredentials.Error() {
		t.Fatalf("expected detail %q, got %q", authkit.ErrInvalidCredentials.Error(), problem.Detail)
	}
}

func TestAuthHandler_Login_UnknownEmail(t *testing.T) {
	_, api := newTestAuthHandler(t)

	resp := api.Post("/auth/login", map[string]string{"email": "nobody@example.com", "password": "whatever"})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 (not a 404 — see mapAuthError, ErrUserNotFound is remapped to invalid credentials by authkit.Login itself), got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestAuthHandler_Me_RequiresSession(t *testing.T) {
	_, api := newTestAuthHandler(t)

	resp := api.Get("/auth/me")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for an unauthenticated request to a Secured:true route, got %d: %s", resp.Code, resp.Body.String())
	}
}
