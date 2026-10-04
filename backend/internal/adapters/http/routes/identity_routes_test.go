package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	privatePath    = "/collections"
	guestOKPath    = "/catalog/series"
	healthPath     = "/health"
	unauthorizedOK = http.StatusUnauthorized
)

// TestRequestClassification covers the three outcomes on a private route and
// on a guest-allowed one: no header is a Guest (allowed only where declared), a
// valid token is authenticated, and a present-but-bad token is a 401, never a
// Guest.
func TestRequestClassification(t *testing.T) {
	api := newTestAPI(t)

	assert.Equal(t, unauthorizedOK, api.Get(privatePath).Code, "a Guest on a private route")
	assert.Equal(t, http.StatusOK, api.Get(guestOKPath).Code, "a Guest on a guest-allowed route")
	assert.Equal(t, http.StatusOK, api.Get(healthPath).Code)

	valid := api.newUserToken(t)
	assert.Equal(t, http.StatusOK, api.Get(privatePath, bearer(valid)).Code)
	assert.Equal(t, http.StatusOK, api.Get(guestOKPath, bearer(valid)).Code)

	badTokens := map[string]string{
		"garbage":                    "garbage",
		"expired":                    api.sign(t, with(validClaims(), "exp", time.Now().Add(-time.Hour).Unix())),
		"minted for another origin":  api.sign(t, with(validClaims(), "azp", "https://evil.example")),
		"minted by another instance": api.sign(t, with(validClaims(), "iss", "https://other.clerk.accounts.dev")),
	}
	for name, token := range badTokens {
		for _, path := range []string{privatePath, guestOKPath} {
			resp := api.Get(path, bearer(token))
			assert.Equal(t, unauthorizedOK, resp.Code, "%s token on %s: %s", name, path, resp.Body.String())
			assert.NotContains(t, resp.Body.String(), token, "the response must not echo the token")
		}
	}

	assert.Equal(t, unauthorizedOK, api.Get(privatePath, "Authorization: Basic abc").Code)
}

func with(claims map[string]any, key string, value any) map[string]any {
	claims[key] = value
	return claims
}

func (a testAPI) usersWithSubject(t *testing.T, subject string) []entity.User {
	t.Helper()
	var users []entity.User
	require.NoError(t, a.db.Where("auth_provider = ? AND auth_subject = ?", dto.AuthProviderClerk, subject).Find(&users).Error)
	return users
}

func (a testAPI) profilesOf(t *testing.T, user entity.User) []entity.UserProfile {
	t.Helper()
	var profiles []entity.UserProfile
	require.NoError(t, a.db.Where("user_id = ?", user.ID).Find(&profiles).Error)
	return profiles
}

func TestFirstRequestCreatesUserAndProfile(t *testing.T) {
	api := newTestAPI(t)
	claims := validClaims()
	subject, _ := claims["sub"].(string)
	token := api.sign(t, claims)
	require.Empty(t, api.usersWithSubject(t, subject))

	require.Equal(t, http.StatusOK, api.Get(privatePath, bearer(token)).Code)

	users := api.usersWithSubject(t, subject)
	require.Len(t, users, 1)
	assert.Equal(t, claims["email"], users[0].Email)
	profiles := api.profilesOf(t, users[0])
	require.Len(t, profiles, 1)
	assert.Equal(t, "Test User", profiles[0].Name)

	// The same Auth Identity keeps mapping to the same profile.
	require.Equal(t, http.StatusOK, api.Get(privatePath, bearer(token)).Code)
	assert.Len(t, api.usersWithSubject(t, subject), 1)
	assert.Len(t, api.profilesOf(t, users[0]), 1)
}

func TestConcurrentFirstRequestsCreateOneUserAndProfile(t *testing.T) {
	api := newTestAPI(t)
	claims := validClaims()
	subject, _ := claims["sub"].(string)
	token := api.sign(t, claims)

	const callers = 12
	codes := make([]int, callers)
	var wg sync.WaitGroup
	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = api.Get(privatePath, bearer(token)).Code
		}()
	}
	wg.Wait()

	for i, code := range codes {
		assert.Equal(t, http.StatusOK, code, "request %d", i)
	}
	users := api.usersWithSubject(t, subject)
	require.Len(t, users, 1)
	assert.Len(t, api.profilesOf(t, users[0]), 1)
}

func TestProfileNameFallsBackToTheEmailLocalPart(t *testing.T) {
	api := newTestAPI(t)
	claims := validClaims()
	delete(claims, "name")
	claims["email"] = "grace.hopper+" + strings.TrimPrefix(claims["sub"].(string), "user_") + "@example.com"

	require.Equal(t, http.StatusOK, api.Get(privatePath, bearer(api.sign(t, claims))).Code)

	users := api.usersWithSubject(t, claims["sub"].(string))
	require.Len(t, users, 1)
	profiles := api.profilesOf(t, users[0])
	require.Len(t, profiles, 1)
	assert.Equal(t, strings.Split(claims["email"].(string), "@")[0], profiles[0].Name)
}

func TestEmailRefreshesAndNameIsNeverOverwritten(t *testing.T) {
	api := newTestAPI(t)
	claims := validClaims()
	subject, _ := claims["sub"].(string)
	require.Equal(t, http.StatusOK, api.Get(privatePath, bearer(api.sign(t, claims))).Code)

	changed := validClaims()
	changed["sub"] = subject
	changed["email"] = "renamed-" + subject + "@example.com"
	changed["name"] = "A Different Name"
	require.Equal(t, http.StatusOK, api.Get(privatePath, bearer(api.sign(t, changed))).Code)

	users := api.usersWithSubject(t, subject)
	require.Len(t, users, 1)
	assert.Equal(t, changed["email"], users[0].Email, "a changed email claim is reflected")
	profiles := api.profilesOf(t, users[0])
	require.Len(t, profiles, 1)
	assert.Equal(t, "Test User", profiles[0].Name, "the profile name is written once")
}

func TestForeignAndMissingCollectionsLookIdentical(t *testing.T) {
	api := newTestAPI(t)
	owner, other := api.newUserToken(t), api.newUserToken(t)

	created := api.Post(privatePath, bearer(owner), map[string]any{"title": "Mine"})
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	var env collectionEnvelope
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &env))

	foreign := api.Get(privatePath+"/"+env.Data.ID, bearer(other))
	missing := api.Get(privatePath+"/"+uuid.NewString(), bearer(other))

	assert.Equal(t, http.StatusNotFound, foreign.Code)
	assert.Equal(t, http.StatusNotFound, missing.Code)
	assert.Equal(t, missing.Body.String(), foreign.Body.String())
}
