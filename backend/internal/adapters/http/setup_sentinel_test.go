package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityHeadersAreOnEveryResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(securityHeaders())
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })

	want := map[string]string{
		"Strict-Transport-Security": "max-age=63072000; includeSubDomains",
		"Referrer-Policy":           "no-referrer",
		"X-Frame-Options":           "DENY",
		"Content-Security-Policy":   "frame-ancestors 'none'",
		"X-Content-Type-Options":    "nosniff",
	}

	for path, status := range map[string]int{"/ok": http.StatusOK, "/missing": http.StatusNotFound} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))

		require.Equal(t, status, w.Code, path)
		for header, value := range want {
			assert.Equal(t, value, w.Header().Get(header), path+" "+header)
		}
	}
}

func TestSetupSentinel_AllowsTheOriginsOfBothLists(t *testing.T) {
	previous := config.Global
	t.Cleanup(func() { config.Global = previous })
	config.Global = &config.Config{
		App:   config.App{ClientUrls: []string{"https://www.example.com"}, Timeout: time.Second},
		Admin: config.Admin{AdminClientUrls: []string{"https://admin.example.com"}},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	require.NoError(t, setupSentinel(r, nil, zerolog.Nop()))
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, origin := range []string{"https://www.example.com", "https://admin.example.com"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/ok", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		r.ServeHTTP(w, req)

		assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"), origin)
	}
}
