package http

import (
	"testing"

	"github.com/gin-gonic/gin"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetup_RefusesToBootOnAMalformedAdminEmail(t *testing.T) {
	configs := config.Config{
		Clerk:     config.Clerk{SecretKey: "sk", Issuer: "https://x.clerk.accounts.dev"},
		RateLimit: config.DefaultRateLimit(),
		Match:     config.Match{Threshold: 0.85, Margin: 0.03},
		Admin:     config.Admin{Emails: []string{"owner@example.com", "ownerexample.com"}},
	}

	_, _, err := Setup(configs)

	require.Error(t, err)
	assert.ErrorContains(t, err, `"ownerexample.com"`)
}

func TestSetup_RefusesToBootWithAdminsButNoAdminAppOrigin(t *testing.T) {
	configs := config.Config{
		Clerk:     config.Clerk{SecretKey: "sk", Issuer: "https://x.clerk.accounts.dev"},
		RateLimit: config.DefaultRateLimit(),
		Match:     config.Match{Threshold: 0.85, Margin: 0.03},
		Admin:     config.Admin{Emails: []string{"owner@example.com"}},
	}

	_, _, err := Setup(configs)

	require.Error(t, err)
	assert.ErrorContains(t, err, "ADMIN_CLIENT_URLS")
}

func TestSetup_RefusesToBootWithAnOriginInBothLists(t *testing.T) {
	configs := config.Config{
		Clerk:     config.Clerk{SecretKey: "sk", Issuer: "https://x.clerk.accounts.dev"},
		RateLimit: config.DefaultRateLimit(),
		Match:     config.Match{Threshold: 0.85, Margin: 0.03},
		App:       config.App{ClientUrls: []string{"https://www.example.com"}},
		Admin: config.Admin{
			Emails:          []string{"owner@example.com"},
			AdminClientUrls: []string{"https://www.example.com"},
		},
	}

	_, _, err := Setup(configs)

	require.Error(t, err)
	assert.ErrorContains(t, err, `"https://www.example.com"`)
}

func TestAdminEmails_NeedsTheR2Settings(t *testing.T) {
	r2 := config.R2{AccountID: "a", AccessKeyID: "k", SecretAccessKey: "s", Bucket: "b"}
	admin := config.Admin{Emails: []string{"owner@example.com"}}

	assert.Equal(t, admin.Emails, adminEmails(config.Config{Admin: admin, R2: r2}))
	assert.Empty(t, adminEmails(config.Config{Admin: admin}), "no R2 settings, no admin routes")
	assert.Empty(t, adminEmails(config.Config{R2: r2}), "no allowlist, no admin routes")
}

func TestNoClientOrigins_OnlyWhenBothListsAreEmpty(t *testing.T) {
	assert.True(t, noClientOrigins(config.Config{}), "neither list set")
	assert.False(t, noClientOrigins(config.Config{App: config.App{ClientUrls: []string{"https://www.example.com"}}}), "only the user app origin")
	assert.False(t, noClientOrigins(config.Config{Admin: config.Admin{AdminClientUrls: []string{"https://admin.example.com"}}}), "only the Admin app origin")
}

func TestGinMode_EveryAppEnvIsOneGinAccepts(t *testing.T) {
	t.Cleanup(func() { gin.SetMode(gin.TestMode) })

	for _, env := range []string{gin.DebugMode, gin.ReleaseMode, gin.TestMode, config.PreviewAppEnv} {
		assert.NotPanics(t, func() { gin.SetMode(ginMode(env)) }, env)
	}
	assert.Equal(t, gin.ReleaseMode, ginMode(config.PreviewAppEnv), "a preview runs as production does")
	assert.Empty(t, httpapi.NewServerConfig(ginMode(config.PreviewAppEnv)).DocsPath, "and does not advertise its docs")
}
