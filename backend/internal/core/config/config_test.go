package config

import (
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClerkEnvVarNames(t *testing.T) {
	t.Setenv("CLERK_SECRET_KEY", "sk_test_x")
	t.Setenv("CLERK_ISSUER", "https://example.clerk.accounts.dev")

	var clerk Clerk
	err := envconfig.Process(clerk.Prefix(), &clerk)

	assert.NoError(t, err)
	assert.Equal(t, "sk_test_x", clerk.SecretKey)
	assert.Equal(t, "https://example.clerk.accounts.dev", clerk.Issuer)
}

func TestClerk_ValidateClerk_NeedsSecretKeyAndIssuer(t *testing.T) {
	assert.NoError(t, Clerk{SecretKey: "sk", Issuer: "https://x.clerk.accounts.dev"}.ValidateClerk())
	assert.Error(t, Clerk{Issuer: "https://x.clerk.accounts.dev"}.ValidateClerk())
	assert.Error(t, Clerk{SecretKey: "sk"}.ValidateClerk())
	assert.Error(t, Clerk{}.ValidateClerk())
}

func TestImageAndR2EnvVarNames(t *testing.T) {
	t.Setenv("IMAGE_BASE_URL", "https://img.example.test")
	t.Setenv("R2_ACCOUNT_ID", "acct")
	t.Setenv("R2_ACCESS_KEY_ID", "key")
	t.Setenv("R2_SECRET_ACCESS_KEY", "secret")
	t.Setenv("R2_BUCKET", "images")

	var image Image
	require.NoError(t, envconfig.Process(image.Prefix(), &image))
	var r2 R2
	require.NoError(t, envconfig.Process(r2.Prefix(), &r2))

	assert.Equal(t, "https://img.example.test", image.BaseURL)
	assert.Equal(t, R2{AccountID: "acct", AccessKeyID: "key", SecretAccessKey: "secret", Bucket: "images"}, r2)
	assert.True(t, r2.Configured())
}

func TestR2_Configured_NeedsEveryField(t *testing.T) {
	full := R2{AccountID: "a", AccessKeyID: "k", SecretAccessKey: "s", Bucket: "b"}
	assert.True(t, full.Configured())

	for name, r2 := range map[string]R2{
		"no account id": {AccessKeyID: "k", SecretAccessKey: "s", Bucket: "b"},
		"no key id":     {AccountID: "a", SecretAccessKey: "s", Bucket: "b"},
		"no secret":     {AccountID: "a", AccessKeyID: "k", Bucket: "b"},
		"no bucket":     {AccountID: "a", AccessKeyID: "k", SecretAccessKey: "s"},
		"nothing":       {},
	} {
		assert.False(t, r2.Configured(), name)
	}
}

func TestRateLimitDefaultsAndEnvVarNames(t *testing.T) {
	var defaults RateLimit
	require.NoError(t, envconfig.Process(defaults.Prefix(), &defaults))
	assert.Equal(t, RateLimit{UserPerMinute: 300, UserBurst: 100, SearchPerMinute: 60, SearchBurst: 30, FacetsPerMinute: 30, FacetsBurst: 15}, defaults)
	assert.NoError(t, defaults.ValidateRateLimit())

	t.Setenv("RATE_LIMIT_USER_PER_MINUTE", "1")
	t.Setenv("RATE_LIMIT_USER_BURST", "2")
	t.Setenv("RATE_LIMIT_SEARCH_PER_MINUTE", "3")
	t.Setenv("RATE_LIMIT_SEARCH_BURST", "4")
	t.Setenv("RATE_LIMIT_FACETS_PER_MINUTE", "5")
	t.Setenv("RATE_LIMIT_FACETS_BURST", "6")

	var set RateLimit
	require.NoError(t, envconfig.Process(set.Prefix(), &set))
	assert.Equal(t, RateLimit{UserPerMinute: 1, UserBurst: 2, SearchPerMinute: 3, SearchBurst: 4, FacetsPerMinute: 5, FacetsBurst: 6}, set)
}

func TestRateLimit_ValidateRateLimit_NeedsEveryLimitPositive(t *testing.T) {
	valid := RateLimit{UserPerMinute: 1, UserBurst: 1, SearchPerMinute: 1, SearchBurst: 1, FacetsPerMinute: 1, FacetsBurst: 1}
	assert.NoError(t, valid.ValidateRateLimit())

	zero := valid
	zero.SearchBurst = 0
	assert.Error(t, zero.ValidateRateLimit())

	negative := valid
	negative.UserPerMinute = -1
	assert.Error(t, negative.ValidateRateLimit())
}
