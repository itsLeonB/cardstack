package http

import (
	"crypto/sha256"
	"crypto/subtle"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	edgeSecretHeader = "X-Edge-Secret"
	cfClientIPHeader = "CF-Connecting-IP"
	healthPath       = "/health"
)

// edgeGuard makes Cloudflare the only way in. A request without the secret
// that Cloudflare adds is rejected, except the platform health check, and the
// client address then comes from Cloudflare's header because the secret proves
// the request passed through it. Client-supplied address headers are dropped so
// neither sentinel's limiter nor its logger can be pointed at a spoofed bucket.
func edgeGuard(secret string) gin.HandlerFunc {
	want := sha256.Sum256([]byte(secret))

	return func(c *gin.Context) {
		if c.Request.URL.Path == healthPath {
			c.Next()
			return
		}

		// Hashing first keeps the comparison constant-time regardless of length.
		got := sha256.Sum256([]byte(c.GetHeader(edgeSecretHeader)))
		if subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		c.Request.Header.Del("X-Forwarded-For")
		c.Request.Header.Del("X-Real-IP")
		if ip := net.ParseIP(c.GetHeader(cfClientIPHeader)); ip != nil {
			c.Request.RemoteAddr = ip.String()
		}
		c.Next()
	}
}
