package config

import (
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
)

// Regression test for a real bug: envconfig's split_words camelCase
// splitter treats "SameSite" as two words, so split_words:"true" on
// CookieSameSite would derive AUTH_COOKIE_SAME_SITE — not
// AUTH_COOKIE_SAMESITE, the name .env.example/railway.ts/this whole
// codebase actually uses. Confirmed live: a real server booted with
// AUTH_COOKIE_SAMESITE=Lax set kept using the "None" default regardless,
// because envconfig was never actually reading that env var. The fix is
// an explicit `envconfig:"COOKIE_SAMESITE"` tag; this proves it works.
func TestAuthCookieSameSiteEnvVarName(t *testing.T) {
	t.Setenv("AUTH_JWT_SECRET", "test-secret")
	t.Setenv("AUTH_COOKIE_SAMESITE", "Strict")

	var auth Auth
	err := envconfig.Process(auth.Prefix(), &auth)

	assert.NoError(t, err)
	assert.Equal(t, "Strict", auth.CookieSamesite)
}

// SameSite=None without Secure is a combination browsers reject outright
// (every auth cookie would silently never get stored) — Load must catch it
// at boot rather than let it ship. Environment variables drive Load(), so
// these exercise the pure helper the same check runs, not Load() itself.
func TestValidateAuthCookiePolicy(t *testing.T) {
	cases := []struct {
		name       string
		sameSite   string
		secure     bool
		wantErrMsg string
	}{
		{name: "None without Secure is rejected", sameSite: "None", secure: false, wantErrMsg: "AUTH_COOKIE_SAMESITE=None requires AUTH_COOKIE_SECURE=true"},
		{name: "none (any case) without Secure is rejected", sameSite: "none", secure: false, wantErrMsg: "AUTH_COOKIE_SAMESITE=None requires AUTH_COOKIE_SECURE=true"},
		{name: "None with Secure is fine", sameSite: "None", secure: true},
		{name: "Lax without Secure is fine (non-TLS local/preview deploys)", sameSite: "Lax", secure: false},
		{name: "Strict without Secure is fine", sameSite: "Strict", secure: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAuthCookiePolicy(Auth{CookieSamesite: tc.sameSite, CookieSecure: tc.secure})
			if tc.wantErrMsg == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tc.wantErrMsg)
		})
	}
}
