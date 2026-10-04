package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/ratelimit"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Each tier has a different burst so a test can tell which bucket answered.
// Refills are one token a second (general) or every two seconds (search,
// facets), so nothing comes back while a test runs unless it moves the clock.
var tightLimits = config.RateLimit{
	UserPerMinute: 60, UserBurst: 4,
	SearchPerMinute: 30, SearchBurst: 2,
	FacetsPerMinute: 30, FacetsBurst: 1,
}

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newRateLimitedAPI(t *testing.T) (testAPI, *fakeClock) {
	t.Helper()
	clock := &fakeClock{now: time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)}
	return newTestAPIWithLimits(t, ratelimit.NewLimits(tightLimits, clock.Now)), clock
}

// assertRateLimited checks the whole 429 contract: the status, a whole-second
// Retry-After hint, and a fixed body that says nothing about the caller.
func assertRateLimited(t *testing.T, resp *httptest.ResponseRecorder, retryAfter string, msgAndArgs ...any) {
	t.Helper()
	require.Equal(t, http.StatusTooManyRequests, resp.Code, append([]any{resp.Body.String()}, msgAndArgs...)...)
	assert.Equal(t, retryAfter, resp.Header().Get("Retry-After"), msgAndArgs...)

	var body struct {
		Status int    `json:"status"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, http.StatusTooManyRequests, body.Status)
	assert.Equal(t, "Too Many Requests", body.Title)
	assert.Equal(t, "too many requests, retry later", body.Detail)
}

func TestRateLimit_SignedInCallerIsLimitedPerUser(t *testing.T) {
	api, clock := newRateLimitedAPI(t)
	auth := bearer(api.newUserToken(t))

	for i := range tightLimits.UserBurst {
		assert.Equal(t, http.StatusOK, api.Get("/collections", auth).Code, "request %d is under the limit", i+1)
	}
	assertRateLimited(t, api.Get("/collections", auth), "1", "over the limit")
	assertRateLimited(t, api.Get("/catalog/series", auth), "1", "the general bucket covers guest-allowed routes too")

	clock.advance(time.Second)
	assert.Equal(t, http.StatusOK, api.Get("/collections", auth).Code, "one token is back after the hinted wait")
	assertRateLimited(t, api.Get("/collections", auth), "1", "and only one")
}

func TestRateLimit_SearchAndFacetsHaveTheirOwnTighterBuckets(t *testing.T) {
	api, clock := newRateLimitedAPI(t)
	auth := bearer(api.newUserToken(t))

	for i := range tightLimits.SearchBurst {
		assert.Equal(t, http.StatusOK, api.Get("/catalog/cards?name=rate", auth).Code, "search %d", i+1)
	}
	assertRateLimited(t, api.Get("/catalog/cards?name=rate", auth), "2", "search is over its own limit")

	for i := range tightLimits.FacetsBurst {
		assert.Equal(t, http.StatusOK, api.Get("/catalog/facets?name=rate", auth).Code, "facets %d", i+1)
	}
	assertRateLimited(t, api.Get("/catalog/facets?name=rate", auth), "2", "facets are over their own limit")

	for i := range tightLimits.UserBurst {
		assert.Equal(t, http.StatusOK, api.Get("/collections", auth).Code, "general request %d: search and facets did not spend it", i+1)
	}

	clock.advance(2 * time.Second)
	assert.Equal(t, http.StatusOK, api.Get("/catalog/cards?name=rate", auth).Code, "search recovers")
	assert.Equal(t, http.StatusOK, api.Get("/catalog/facets?name=rate", auth).Code, "facets recover")
}

func TestRateLimit_UsersDoNotShareAnAllowance(t *testing.T) {
	api, _ := newRateLimitedAPI(t)
	first := bearer(api.newUserToken(t))
	second := bearer(api.newUserToken(t))

	for range tightLimits.UserBurst {
		require.Equal(t, http.StatusOK, api.Get("/collections", first).Code)
	}
	assertRateLimited(t, api.Get("/collections", first), "1")

	assert.Equal(t, http.StatusOK, api.Get("/collections", second).Code, "the first user's traffic must not spend the second's allowance")
}

func TestRateLimit_GuestsAreNotLimitedPerUser(t *testing.T) {
	api, _ := newRateLimitedAPI(t)

	for i := range tightLimits.UserBurst + tightLimits.SearchBurst + 1 {
		assert.Equal(t, http.StatusOK, api.Get("/catalog/cards?name=rate").Code, "guest search %d", i+1)
	}
}
