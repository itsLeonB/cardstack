package routes

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/ratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	claims["azp"] = testAdminOrigin
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
	unlistedOnAdminApp := api.Get(adminCheckPath, bearer(api.tokenFor(t, "unlisted@example.com")))
	guest := api.Get(adminCheckPath)

	assert.Equal(t, http.StatusNotFound, signedIn.Code, "a signed-in caller who is not listed")
	assert.Equal(t, http.StatusNotFound, unlistedOnAdminApp.Code, "an unlisted email on the Admin app origin")
	assert.Equal(t, http.StatusNotFound, guest.Code, "a guest")
	assert.JSONEq(t, signedIn.Body.String(), guest.Body.String(), "the two answers are the same, so probing learns nothing")
	assert.JSONEq(t, unlistedOnAdminApp.Body.String(), guest.Body.String(), "an unlisted email on the Admin app origin gets the same answer")
}

// TestAdminRoutes_EveryAdminRouteAnswersAsItDoesToAGuest covers every admin
// route for the two callers that must not reach it: a listed email on the user
// app origin, and an unlisted email on the Admin app origin.
func TestAdminRoutes_EveryAdminRouteAnswersAsItDoesToAGuest(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	id := uuid.NewString()
	listedOnUserApp := bearer(api.userTokenFor(t, adminEmail))
	unlistedOnAdminApp := bearer(api.tokenFor(t, "unlisted@example.com"))

	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, adminCheckPath},
		{http.MethodPost, "/admin/expansion-sets"},
		{http.MethodPut, "/admin/expansion-sets/" + id},
		{http.MethodGet, "/admin/cards"},
		{http.MethodPost, "/admin/cards"},
		{http.MethodPut, "/admin/cards/" + id},
		{http.MethodDelete, "/admin/cards/" + id},
		{http.MethodPost, "/admin/cards/" + id + "/image-upload"},
		{http.MethodPost, "/admin/cards/" + id + "/image-confirm"},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			guest := api.Do(route.method, route.path)
			require.Equal(t, http.StatusNotFound, guest.Code, guest.Body.String())

			for _, caller := range []struct {
				name string
				auth string
			}{
				{"listed email on the user app origin", listedOnUserApp},
				{"unlisted email on the Admin app origin", unlistedOnAdminApp},
			} {
				resp := api.Do(route.method, route.path, caller.auth)

				assert.Equal(t, guest.Code, resp.Code, caller.name)
				assert.JSONEq(t, guest.Body.String(), resp.Body.String(), caller.name)
			}
		})
	}
}

func TestAdminRoutes_TokenFromAnOriginInNeitherListIsRejected(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	claims := validClaims()
	claims["email"] = adminEmail
	claims["azp"] = "https://evil.example"
	token := bearer(api.sign(t, claims))

	for _, path := range []string{"/catalog/series", adminCheckPath} {
		resp := api.Get(path, token)

		assert.Equal(t, http.StatusUnauthorized, resp.Code, "%s: %s", path, resp.Body.String())
	}
}

func TestAdminRoutes_ListedEmailOnTheUserAppOriginGetsNotFound(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})

	userApp := api.Get(adminCheckPath, bearer(api.userTokenFor(t, adminEmail)))
	guest := api.Get(adminCheckPath)

	assert.Equal(t, http.StatusNotFound, userApp.Code, "an Admin's token minted for the user app")
	assert.JSONEq(t, guest.Body.String(), userApp.Body.String(), "the same body a Guest gets, so the Admin app is not revealed")
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
