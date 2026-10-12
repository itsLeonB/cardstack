package http

import (
	"testing"

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

func TestAdminEmails_NeedsTheR2Settings(t *testing.T) {
	r2 := config.R2{AccountID: "a", AccessKeyID: "k", SecretAccessKey: "s", Bucket: "b"}
	admin := config.Admin{Emails: []string{"owner@example.com"}}

	assert.Equal(t, admin.Emails, adminEmails(config.Config{Admin: admin, R2: r2}))
	assert.Empty(t, adminEmails(config.Config{Admin: admin}), "no R2 settings, no admin routes")
	assert.Empty(t, adminEmails(config.Config{R2: r2}), "no allowlist, no admin routes")
}
