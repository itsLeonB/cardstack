package config

import "time"

// Auth holds go-authkit's stateful-mode configuration (JWT signing, refresh
// token lifetime) plus the cookie parameters the http/auth package's
// Transport uses to build Set-Cookie headers. See docs/adr/0003 and the
// ticket 02 plan's "Auth config" decision.
type Auth struct {
	JWTSecret       string        `split_words:"true" required:"true"`
	JWTIssuer       string        `split_words:"true" default:"cardstack"`
	JWTDuration     time.Duration `split_words:"true" default:"15m"`
	RefreshTokenTTL time.Duration `split_words:"true" default:"168h"`
	CookieDomain    string        `split_words:"true"`
	CookieSecure    bool          `split_words:"true" default:"true"`
	// CookieSamesite defaults to "None" because the deployed frontend
	// (Vercel) and backend (Railway) are on different sites: browsers
	// exclude SameSite=Lax cookies from cross-site fetch/XHR requests even
	// with credentials: "include", so Lax would silently break every
	// authenticated request in production. None requires Secure, which
	// CookieSecure already defaults to.
	//
	// Field is named "Samesite" (not "SameSite") so split_words' camelCase
	// splitter produces AUTH_COOKIE_SAMESITE — the env var .env.example and
	// every deploy config actually use — instead of AUTH_COOKIE_SAME_SITE.
	CookieSamesite string `split_words:"true" default:"None"`
}

func (Auth) Prefix() string { return "AUTH" }
