package http

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/kroma-labs/sentinel-go/httpserver"
	sentinelGin "github.com/kroma-labs/sentinel-go/httpserver/adapters/gin"
	"github.com/rs/zerolog"
)

func setupSentinel(router *gin.Engine, skipPaths []string, logger zerolog.Logger) error {
	corsCfg := httpserver.CORSConfig{
		AllowedOrigins: config.Global.ClientUrls,
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Cache-Control",
			"Referer",
			"User-Agent",
			"range",
			"DNT",
			"sec-ch-ua",
			"sec-ch-ua-platform",
			"sec-ch-ua-mobile",
		},
		ExposedHeaders:   []string{"Content-Length", "X-Total-Count"},
		AllowCredentials: true,
	}

	if len(corsCfg.AllowedOrigins) < 1 {
		corsCfg.AllowedOrigins = []string{"*"}
	}

	metricsCfg := httpserver.DefaultMetricsConfig()
	metricsCfg.SkipPaths = skipPaths

	metrics, err := httpserver.NewMetrics(metricsCfg)
	if err != nil {
		return fmt.Errorf("error setting up metrics config: %w", err)
	}

	tracingCfg := httpserver.DefaultTracingConfig()
	tracingCfg.SkipPaths = skipPaths

	router.Use(
		realIP(),
		securityHeaders(),
		sentinelGin.Recovery(logger),
		sentinelGin.Tracing(tracingCfg),
		sentinelGin.CORS(corsCfg),
		sentinelGin.Timeout(config.Global.Timeout),
		sentinelGin.Metrics(metrics),
		sentinelGin.RateLimit(httpserver.RateLimitConfig{
			// ponytail: per-IP global limit. 100 req/s burst 200 is generous for
			// legitimate users but stops single-IP floods. Ceiling: doesn't help
			// distributed DDoS. Upgrade path: Redis-backed limiter + WAF.
			Limit:   100,
			Burst:   200,
			KeyFunc: httpserver.KeyFuncByIP(),
		}),
	)

	return nil
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("Content-Security-Policy", "frame-ancestors 'none'")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	}
}

// ponytail: overwrites RemoteAddr with the real client IP from the
// deployment platform's proxy header, so sentinel's logger and rate limiter
// see the true IP.
func realIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		if ip := c.GetHeader("X-Real-IP"); ip != "" {
			c.Request.RemoteAddr = ip
		}
		c.Next()
	}
}
