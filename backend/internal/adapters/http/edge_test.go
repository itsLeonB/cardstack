package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kroma-labs/sentinel-go/httpserver"
	sentinelGin "github.com/kroma-labs/sentinel-go/httpserver/adapters/gin"
	"github.com/stretchr/testify/assert"
)

const testSecret = "s3cret"

// newEdgeRouter wires the production clientAddress middleware in front of
// routes that echo the bucket key the per-IP limiter would use.
func newEdgeRouter(edgeSecret string, limiter ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(clientAddress(edgeSecret))
	r.Use(limiter...)
	key := func(c *gin.Context) { c.String(http.StatusOK, httpserver.KeyFuncByIP()(c.Request)) }
	r.GET("/health", key)
	r.GET("/catalog/cards", key)
	return r
}

func do(r *gin.Engine, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = "10.0.0.1:4321"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestEdgeGuard_SecretCorrect(t *testing.T) {
	w := do(newEdgeRouter(testSecret), "/catalog/cards", map[string]string{
		edgeSecretHeader: testSecret,
		cfClientIPHeader: "203.0.113.7",
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "203.0.113.7", w.Body.String())
}

func TestEdgeGuard_SecretMissingOrWrong(t *testing.T) {
	r := newEdgeRouter(testSecret)

	for name, headers := range map[string]map[string]string{
		"missing": {cfClientIPHeader: "203.0.113.7"},
		"wrong":   {edgeSecretHeader: "nope", cfClientIPHeader: "203.0.113.7"},
		"prefix":  {edgeSecretHeader: testSecret[:3]},
	} {
		t.Run(name, func(t *testing.T) {
			w := do(r, "/catalog/cards", headers)

			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.JSONEq(t, `{"error":"forbidden"}`, w.Body.String())
		})
	}
}

func TestEdgeGuard_HealthExempt(t *testing.T) {
	w := do(newEdgeRouter(testSecret), "/health", nil)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEdgeGuard_SpoofedAddressHeadersIgnored(t *testing.T) {
	w := do(newEdgeRouter(testSecret), "/catalog/cards", map[string]string{
		edgeSecretHeader:  testSecret,
		cfClientIPHeader:  "203.0.113.7",
		"X-Forwarded-For": "198.51.100.1",
		"X-Real-IP":       "198.51.100.2",
	})

	assert.Equal(t, "203.0.113.7", w.Body.String())
}

func TestEdgeGuard_InvalidConnectingIPKeepsPeerAddress(t *testing.T) {
	w := do(newEdgeRouter(testSecret), "/catalog/cards", map[string]string{
		edgeSecretHeader:  testSecret,
		cfClientIPHeader:  "not-an-ip",
		"X-Forwarded-For": "198.51.100.1",
	})

	assert.Equal(t, "10.0.0.1:4321", w.Body.String())
}

func TestEdgeGuard_SecretUnset(t *testing.T) {
	r := newEdgeRouter("")

	t.Run("no edge check", func(t *testing.T) {
		w := do(r, "/catalog/cards", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "10.0.0.1:4321", w.Body.String())
	})

	t.Run("address logic unchanged", func(t *testing.T) {
		w := do(r, "/catalog/cards", map[string]string{
			"X-Real-IP":      "198.51.100.2",
			cfClientIPHeader: "203.0.113.7",
		})

		assert.Equal(t, "198.51.100.2", w.Body.String())
	})
}

// A spoofed X-Forwarded-For must not buy a fresh bucket: both requests share
// one Cloudflare address, so the second is limited whatever the header says.
func TestEdgeGuard_LimiterKeysOnConnectingIP(t *testing.T) {
	limiter := sentinelGin.RateLimit(httpserver.RateLimitConfig{
		Limit:   0.001,
		Burst:   1,
		KeyFunc: httpserver.KeyFuncByIP(),
	})
	r := newEdgeRouter(testSecret, limiter)
	headers := func(xff string) map[string]string {
		return map[string]string{
			edgeSecretHeader:  testSecret,
			cfClientIPHeader:  "203.0.113.7",
			"X-Forwarded-For": xff,
		}
	}

	assert.Equal(t, http.StatusOK, do(r, "/catalog/cards", headers("198.51.100.1")).Code)
	assert.Equal(t, http.StatusTooManyRequests, do(r, "/catalog/cards", headers("198.51.100.99")).Code)

	other := headers("198.51.100.1")
	other[cfClientIPHeader] = "203.0.113.8"
	assert.Equal(t, http.StatusOK, do(r, "/catalog/cards", other).Code)
}
