// Package auth holds the cookie transport and Huma middleware for
// go-authkit's stateful mode, ported from authgin's Gin-based equivalents
// (see docs/adr/0003 and the ticket 02 plan's "Session guard middleware" /
// "CSRF" / "Cookie transport" decisions).
package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/core/config"
)

const (
	accessTokenCookie  = "access_token"
	refreshTokenCookie = "refresh_token"
	csrfTokenCookie    = "csrf_token"
	fingerprintCookie  = "__Secure-Fgp"
)

// Transport builds the Set-Cookie values go-authkit's stateful flow needs.
//
// This is a hand-rolled equivalent of authgin.CookieTransport rather than a
// direct reuse of it, for two reasons found only by reading authgin's
// actual source (see the plan's fallback note on this decision):
//
//  1. authgin.CookieTransport hardcodes its cookie Paths to "/api" and
//     "/api/v1/auth" — not configurable via CookieConfig — which doesn't
//     match this API's routes (mounted at the engine root, e.g.
//     "/auth/login", not under "/api"). Reusing it as-is would scope the
//     access-token cookie to a path this API never serves, so it would
//     never actually be sent back by the browser.
//  2. Its SetTokens/ClearTokens methods are shaped around
//     http.ResponseWriter, built for authgin's own Gin handlers. Huma
//     supports a native `[]http.Cookie` response-header field (see
//     auth_handler.go's login/logout/refresh Outputs), which is the more
//     idiomatic fit for custom Huma handlers and needs no
//     ResponseWriter/gin.Context unwrapping at all.
//
// Transport only builds http.Cookie values; nothing here writes to a
// ResponseWriter directly.
type Transport struct {
	cfg config.Auth
}

// NewTransport builds a Transport from the given auth config.
func NewTransport(cfg config.Auth) *Transport {
	return &Transport{cfg: cfg}
}

func (t *Transport) sameSite() http.SameSite {
	switch strings.ToLower(t.cfg.CookieSameSite) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

// SetCookies builds the cookies a successful login/refresh sets: access
// token, refresh token, fingerprint, and a JS-readable CSRF token (the
// double-submit cookie CSRFGuard checks against the X-CSRF-Token header).
// The refresh token cookie is scoped to "/auth" only, mirroring authgin's
// own intent of exposing it solely to the auth routes that need it.
func (t *Transport) SetCookies(access, refresh, fingerprint, csrfToken string) []http.Cookie {
	return []http.Cookie{
		t.cookie(accessTokenCookie, access, "/", true, t.cfg.JWTDuration),
		t.cookie(refreshTokenCookie, refresh, "/auth", true, t.cfg.RefreshTokenTTL),
		t.cookie(fingerprintCookie, fingerprint, "/", true, t.cfg.RefreshTokenTTL),
		t.cookie(csrfTokenCookie, csrfToken, "/", false, t.cfg.JWTDuration),
	}
}

// ClearCookies builds the Set-Cookie values that expire every cookie
// SetCookies sets, for logout.
func (t *Transport) ClearCookies() []http.Cookie {
	return []http.Cookie{
		t.expired(accessTokenCookie, "/", true),
		t.expired(refreshTokenCookie, "/auth", true),
		t.expired(fingerprintCookie, "/", true),
		t.expired(csrfTokenCookie, "/", false),
	}
}

func (t *Transport) cookie(name, value, path string, httpOnly bool, ttl time.Duration) http.Cookie {
	return http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   t.cfg.CookieDomain,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: httpOnly,
		Secure:   t.cfg.CookieSecure,
		SameSite: t.sameSite(),
	}
}

func (t *Transport) expired(name, path string, httpOnly bool) http.Cookie {
	return http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		Domain:   t.cfg.CookieDomain,
		MaxAge:   -1,
		HttpOnly: httpOnly,
		Secure:   t.cfg.CookieSecure,
		SameSite: t.sameSite(),
	}
}
