package routes

import (
	"cmp"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/db/postgres/migrations"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// authTestServices builds a *provider.Services backed by a real local
// Postgres instance (per docs/adr/0005: feature tests run against the full
// real stack, not mocks) rather than through provider.InitializeProviders,
// so the test needs no AUTH_*/config.Load() env at all — it sets
// config.Global directly, matching register_routes_test.go's existing
// pattern of hand-building *provider.Services instead of the full wire
// graph. See internal/adapters/repository/repository_test_helper_test.go
// for how to start a matching Postgres 18 locally.
func authTestServices(t *testing.T) *provider.Services {
	t.Helper()

	config.Global = &config.Config{
		Auth: config.Auth{
			JWTSecret:       "test-secret-for-feature-tests",
			JWTIssuer:       "cardstack-test",
			JWTDuration:     15 * time.Minute,
			RefreshTokenTTL: 24 * time.Hour,
			CookieSecure:    false,
			CookieSamesite:  "Lax",
		},
	}

	dsn := "host=" + envOr("DB_HOST", "localhost") +
		" port=" + envOr("DB_PORT", "5432") +
		" user=" + envOr("DB_USER", "cardstack") +
		" password=" + envOr("DB_PASSWORD", "cardstack") +
		" dbname=" + envOr("DB_NAME", "cardstack") +
		" sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("connecting to test postgres: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("getting sql.DB: %v", err)
	}
	migrateTestDB(t, sqlDB)

	// Deliberately no table truncation here: this database is shared with
	// internal/adapters/repository's tests, which `go test ./...` runs
	// concurrently in a separate process — a truncate here could wipe rows
	// that package is mid-assertion on. TestAuthFlow below uses a
	// uuid-suffixed email instead, so it never collides with leftover data
	// from a previous run either.

	users := provider.ProvideUserStore(&provider.DataSources{Gorm: db, SQL: sqlDB})
	profiles := provider.ProvideProfileLookup(&provider.DataSources{Gorm: db, SQL: sqlDB})
	sessions := provider.ProvideSessionStore(&provider.DataSources{Gorm: db, SQL: sqlDB})
	refresh := provider.ProvideRefreshTokenStore(&provider.DataSources{Gorm: db, SQL: sqlDB})
	tx := provider.ProvideTransactor(&provider.DataSources{Gorm: db, SQL: sqlDB})
	cache := provider.ProvideSessionCache()
	t.Cleanup(func() { _ = cache.Shutdown() })

	kit, cleanup, err := provider.ProvideAuthKit(config.Global.Auth, users, sessions, refresh, tx, cache)
	if err != nil {
		t.Fatalf("ProvideAuthKit: %v", err)
	}
	t.Cleanup(cleanup)

	return provider.ProvideServices(kit, profiles)
}

func migrateTestDB(t *testing.T, sqlDB *sql.DB) {
	t.Helper()

	goose.SetBaseFS(migrations.Migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("setting goose dialect: %v", err)
	}
	if err := goose.Up(sqlDB, "."); err != nil {
		t.Fatalf("running migrations: %v", err)
	}
}

func envOr(key, fallback string) string {
	return cmp.Or(os.Getenv(key), fallback)
}

type authEnvelope struct {
	Data struct {
		Message   string `json:"message"`
		CSRFToken string `json:"csrfToken"`
	} `json:"data"`
}

type meEnvelope struct {
	Data struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"data"`
}

func cookieHeader(cookies []*http.Cookie) string {
	header := ""
	for i, c := range cookies {
		if i > 0 {
			header += "; "
		}
		header += c.Name + "=" + c.Value
	}
	return "Cookie: " + header
}

func csrfFrom(cookies []*http.Cookie) string {
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			return c.Value
		}
	}
	return ""
}

// TestAuthFlow exercises the full register -> login -> /auth/me -> logout ->
// refresh flow end to end (ticket 02's testing decision), plus the
// unauthenticated-rejection and CSRF checks the plan calls out explicitly.
func TestAuthFlow(t *testing.T) {
	services := authTestServices(t)
	_, api := humatest.New(t, httpapi.NewConfig())
	RegisterRoutes(api, services)

	email := uuid.NewString() + "@example.com"
	const password = "correct-horse-battery-staple"

	// Register.
	regResp := api.Post("/auth/register", map[string]string{
		"email":                email,
		"password":             password,
		"passwordConfirmation": password,
	})
	if regResp.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", regResp.Code, regResp.Body.String())
	}

	// Registering the same email again is rejected.
	dupResp := api.Post("/auth/register", map[string]string{
		"email":                email,
		"password":             password,
		"passwordConfirmation": password,
	})
	if dupResp.Code != http.StatusConflict {
		t.Fatalf("duplicate register: expected 409, got %d: %s", dupResp.Code, dupResp.Body.String())
	}

	// Mismatched passwordConfirmation is rejected.
	mismatchResp := api.Post("/auth/register", map[string]string{
		"email":                "someone-else@example.com",
		"password":             password,
		"passwordConfirmation": "does-not-match",
	})
	if mismatchResp.Code != http.StatusBadRequest {
		t.Fatalf("mismatched confirmation: expected 400, got %d: %s", mismatchResp.Code, mismatchResp.Body.String())
	}

	// Wrong password is rejected.
	badLoginResp := api.Post("/auth/login", map[string]string{"email": email, "password": "wrong-password"})
	if badLoginResp.Code != http.StatusUnauthorized {
		t.Fatalf("bad login: expected 401, got %d: %s", badLoginResp.Code, badLoginResp.Body.String())
	}

	// GET /auth/me without any session cookies is rejected — the ticket
	// checklist item covering the Secured:true guard.
	if resp := api.Get("/auth/me"); resp.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /auth/me: expected 401, got %d: %s", resp.Code, resp.Body.String())
	}

	// Login.
	loginResp := api.Post("/auth/login", map[string]string{"email": email, "password": password})
	if loginResp.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", loginResp.Code, loginResp.Body.String())
	}
	var loginBody authEnvelope
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginBody); err != nil {
		t.Fatalf("decoding login body: %v", err)
	}
	cookies := loginResp.Result().Cookies()
	if len(cookies) != 4 {
		t.Fatalf("expected 4 cookies from login, got %d: %+v", len(cookies), cookies)
	}
	if loginBody.Data.CSRFToken != csrfFrom(cookies) {
		t.Fatalf("csrfToken in body (%q) doesn't match csrf_token cookie (%q)", loginBody.Data.CSRFToken, csrfFrom(cookies))
	}

	// GET /auth/me with the session cookies set by login.
	meResp := api.Get("/auth/me", cookieHeader(cookies))
	if meResp.Code != http.StatusOK {
		t.Fatalf("/auth/me: expected 200, got %d: %s", meResp.Code, meResp.Body.String())
	}
	var me meEnvelope
	if err := json.Unmarshal(meResp.Body.Bytes(), &me); err != nil {
		t.Fatalf("decoding /auth/me body: %v", err)
	}
	if me.Data.Email != email {
		t.Fatalf("expected /auth/me email %q, got %q", email, me.Data.Email)
	}

	// Refresh: rotates cookies and still resolves to the same user.
	refreshResp := api.Post("/auth/refresh", cookieHeader(cookies), "X-CSRF-Token: "+csrfFrom(cookies))
	if refreshResp.Code != http.StatusOK {
		t.Fatalf("refresh: expected 200, got %d: %s", refreshResp.Code, refreshResp.Body.String())
	}
	rotatedCookies := refreshResp.Result().Cookies()
	if len(rotatedCookies) != 4 {
		t.Fatalf("expected 4 cookies from refresh, got %d: %+v", len(rotatedCookies), rotatedCookies)
	}

	meAfterRefresh := api.Get("/auth/me", cookieHeader(rotatedCookies))
	if meAfterRefresh.Code != http.StatusOK {
		t.Fatalf("/auth/me after refresh: expected 200, got %d: %s", meAfterRefresh.Code, meAfterRefresh.Body.String())
	}

	// Logout without a CSRF header is rejected.
	noCSRFResp := api.Post("/auth/logout", cookieHeader(rotatedCookies))
	if noCSRFResp.Code != http.StatusForbidden {
		t.Fatalf("logout without CSRF header: expected 403, got %d: %s", noCSRFResp.Code, noCSRFResp.Body.String())
	}

	// Logout.
	logoutResp := api.Post("/auth/logout", cookieHeader(rotatedCookies), "X-CSRF-Token: "+csrfFrom(rotatedCookies))
	if logoutResp.Code != http.StatusNoContent {
		t.Fatalf("logout: expected 204, got %d: %s", logoutResp.Code, logoutResp.Body.String())
	}

	// The access token is revoked immediately (SessionCache.Delete on logout).
	meAfterLogout := api.Get("/auth/me", cookieHeader(rotatedCookies))
	if meAfterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("/auth/me after logout: expected 401, got %d: %s", meAfterLogout.Code, meAfterLogout.Body.String())
	}

	// The refresh token was deleted on logout too.
	refreshAfterLogout := api.Post("/auth/refresh", cookieHeader(rotatedCookies), "X-CSRF-Token: "+csrfFrom(rotatedCookies))
	if refreshAfterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout: expected 401, got %d: %s", refreshAfterLogout.Code, refreshAfterLogout.Body.String())
	}
}
