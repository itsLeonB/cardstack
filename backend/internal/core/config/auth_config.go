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
	CookieSameSite  string        `split_words:"true" default:"Lax"`
}

func (Auth) Prefix() string { return "AUTH" }
