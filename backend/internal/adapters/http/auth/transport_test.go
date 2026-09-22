package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/stretchr/testify/assert"
)

// The "__Secure-" cookie name prefix is only valid on a cookie that also
// carries the Secure attribute (RFC 6265bis) — a mismatch here means
// browsers (and modern curl) silently refuse to even store the fingerprint
// cookie, breaking every authenticated request. See transport.go's constant
// doc comment.
func TestFingerprintCookieName(t *testing.T) {
	secure := NewTransport(config.Auth{CookieSecure: true})
	assert.Equal(t, "__Secure-Fgp", secure.FingerprintCookieName())

	insecure := NewTransport(config.Auth{CookieSecure: false})
	assert.Equal(t, "Fgp", insecure.FingerprintCookieName())
}

func TestSetCookiesAndClearCookiesUseMatchingFingerprintName(t *testing.T) {
	for _, cookieSecure := range []bool{true, false} {
		transport := NewTransport(config.Auth{
			CookieSecure:    cookieSecure,
			JWTDuration:     15 * time.Minute,
			RefreshTokenTTL: time.Hour,
		})

		set := transport.SetCookies("access", "refresh", "fingerprint", "csrf")
		cleared := transport.ClearCookies()

		wantName := transport.FingerprintCookieName()
		if cookie, ok := findCookie(set, wantName); assert.Truef(t, ok, "SetCookies() missing cookie %q (CookieSecure=%v)", wantName, cookieSecure) {
			// The whole point of deriving the name from CookieSecure is that
			// the two never drift apart: a "__Secure-" name always carries
			// Secure, a plain name never falsely claims it.
			assert.Equal(t, cookieSecure, cookie.Secure)
		}

		_, ok := findCookie(cleared, wantName)
		assert.Truef(t, ok, "ClearCookies() missing cookie %q (CookieSecure=%v)", wantName, cookieSecure)
	}
}

func findCookie(cookies []http.Cookie, name string) (http.Cookie, bool) {
	for _, c := range cookies {
		if c.Name == name {
			return c, true
		}
	}
	return http.Cookie{}, false
}
