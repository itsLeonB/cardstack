package routes

import (
	"net/http"
	"testing"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/ratelimit"
	"github.com/stretchr/testify/assert"
)

// adminCheckPath is the one admin route today: it answers 204 to an Admin and
// 404 to everyone else, which is how the /admin page learns whether to render.
// Later admin routes join the same group and gate.
const adminCheckPath = "/admin/check"

const adminEmail = "owner@example.com"

// tokenFor is a valid token whose email claim is email.
func (a testAPI) tokenFor(t *testing.T, email string) string {
	t.Helper()
	claims := validClaims()
	claims["email"] = email
	return a.sign(t, claims)
}

func TestAdminRoutes_ListedCallerPasses(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})

	resp := api.Get(adminCheckPath, bearer(api.tokenFor(t, adminEmail)))

	assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
}

func TestAdminRoutes_EmailIsComparedCaseInsensitively(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{"Owner@Example.com"})

	resp := api.Get(adminCheckPath, bearer(api.tokenFor(t, "OWNER@example.COM")))

	assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
}

func TestAdminRoutes_UnlistedCallerAndGuestGetNotFound(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})

	signedIn := api.Get(adminCheckPath, bearer(api.newUserToken(t)))
	guest := api.Get(adminCheckPath)

	assert.Equal(t, http.StatusNotFound, signedIn.Code, "a signed-in caller who is not listed")
	assert.Equal(t, http.StatusNotFound, guest.Code, "a guest")
	assert.JSONEq(t, signedIn.Body.String(), guest.Body.String(), "the two answers are the same, so probing learns nothing")
}

func TestAdminRoutes_AbsentWhenNoAdminIsConfigured(t *testing.T) {
	off := newTestAPIWithAdmins(t, nil)
	on := newTestAPIWithAdmins(t, []string{adminEmail})

	assert.NotContains(t, off.OpenAPI().Paths, adminCheckPath, "no allowlist or no R2 settings means no route")
	assert.Contains(t, on.OpenAPI().Paths, adminCheckPath)
	assert.Equal(t, http.StatusNotFound, off.Get(adminCheckPath, bearer(off.tokenFor(t, adminEmail))).Code)
}

func TestAdminRoutes_ShareTheRateLimitsOfTheRestOfTheAPI(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)}
	api := newTestAPIWith(t, ratelimit.NewLimits(tightLimits, clock.Now), []string{adminEmail})
	auth := bearer(api.tokenFor(t, adminEmail))

	for i := range tightLimits.User.Burst {
		assert.Equal(t, http.StatusNoContent, api.Get(adminCheckPath, auth).Code, "request %d is under the limit", i+1)
	}

	assertRateLimited(t, api.Get(adminCheckPath, auth), "1")
}
