package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/core/config"
)

// The "__Secure-" cookie name prefix is only valid on a cookie that also
// carries the Secure attribute (RFC 6265bis) — a mismatch here means
// browsers (and modern curl) silently refuse to even store the fingerprint
// cookie, breaking every authenticated request. See transport.go's constant
// doc comment.
func TestFingerprintCookieName(t *testing.T) {
	secure := NewTransport(config.Auth{CookieSecure: true})
	if got := secure.FingerprintCookieName(); got != "__Secure-Fgp" {
		t.Errorf("FingerprintCookieName() with CookieSecure=true = %q, want %q", got, "__Secure-Fgp")
	}

	insecure := NewTransport(config.Auth{CookieSecure: false})
	if got := insecure.FingerprintCookieName(); got != "Fgp" {
		t.Errorf("FingerprintCookieName() with CookieSecure=false = %q, want %q", got, "Fgp")
	}
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
		if cookie, ok := findCookie(set, wantName); !ok {
			t.Errorf("SetCookies() missing cookie %q (CookieSecure=%v)", wantName, cookieSecure)
		} else if cookie.Secure != cookieSecure {
			// The whole point of deriving the name from CookieSecure is that
			// the two never drift apart: a "__Secure-" name always carries
			// Secure, a plain name never falsely claims it.
			t.Errorf("cookie %q has Secure=%v, want %v", wantName, cookie.Secure, cookieSecure)
		}

		if _, ok := findCookie(cleared, wantName); !ok {
			t.Errorf("ClearCookies() missing cookie %q (CookieSecure=%v)", wantName, cookieSecure)
		}
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
