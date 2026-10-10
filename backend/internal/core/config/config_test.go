package config

import (
	"fmt"
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
	defaults := DefaultRateLimit()
	require.NoError(t, envconfig.Process(defaults.Prefix(), &defaults))
	assert.Equal(t, RateLimit{User: Tier{PerMinute: 300, Burst: 100}, Search: Tier{PerMinute: 60, Burst: 30}, Facets: Tier{PerMinute: 30, Burst: 15}}, defaults)
	assert.NoError(t, defaults.ValidateRateLimit())

	t.Setenv("RATE_LIMIT_USER_PER_MINUTE", "1")
	t.Setenv("RATE_LIMIT_USER_BURST", "2")
	t.Setenv("RATE_LIMIT_SEARCH_PER_MINUTE", "3")
	t.Setenv("RATE_LIMIT_SEARCH_BURST", "4")
	t.Setenv("RATE_LIMIT_FACETS_PER_MINUTE", "5")
	t.Setenv("RATE_LIMIT_FACETS_BURST", "6")

	set := DefaultRateLimit()
	require.NoError(t, envconfig.Process(set.Prefix(), &set))
	assert.Equal(t, RateLimit{User: Tier{PerMinute: 1, Burst: 2}, Search: Tier{PerMinute: 3, Burst: 4}, Facets: Tier{PerMinute: 5, Burst: 6}}, set)
}

func TestMatchDefaultsAndEnvVarNames(t *testing.T) {
	var defaults Match
	require.NoError(t, envconfig.Process(defaults.Prefix(), &defaults))
	assert.Equal(t, Match{Threshold: 0.85, Margin: 0.03}, defaults)
	assert.NoError(t, defaults.ValidateMatch())

	t.Setenv("MATCH_THRESHOLD", "0.9")
	t.Setenv("MATCH_MARGIN", "0.05")

	var set Match
	require.NoError(t, envconfig.Process(set.Prefix(), &set))
	assert.Equal(t, Match{Threshold: 0.9, Margin: 0.05}, set)
}

func TestMatch_ValidateMatch(t *testing.T) {
	assert.NoError(t, Match{Threshold: 1, Margin: 0}.ValidateMatch())
	assert.NoError(t, Match{Threshold: 0.01, Margin: 0.99}.ValidateMatch())

	for name, m := range map[string]Match{
		"zero threshold":     {Threshold: 0, Margin: 0.03},
		"negative threshold": {Threshold: -0.1, Margin: 0.03},
		"threshold over one": {Threshold: 1.1, Margin: 0.03},
		"negative margin":    {Threshold: 0.85, Margin: -0.01},
		"margin of one":      {Threshold: 0.85, Margin: 1},
	} {
		assert.Error(t, m.ValidateMatch(), name)
	}
}

func TestRateLimit_ValidateRateLimit_NeedsEveryLimitPositive(t *testing.T) {
	assert.NoError(t, DefaultRateLimit().ValidateRateLimit())

	zero := DefaultRateLimit()
	zero.Search.Burst = 0
	assert.Error(t, zero.ValidateRateLimit())

	negative := DefaultRateLimit()
	negative.User.PerMinute = -1
	assert.Error(t, negative.ValidateRateLimit())
}

// A token's authorized party is compared to each entry as an exact string, so
// an entry that is not a bare origin makes every sign-in fail with a 401.
func TestApp_ValidateClientUrls_RejectsEntriesThatCanNeverMatchAnOrigin(t *testing.T) {
	for _, ok := range [][]string{
		nil, // a preview has none until its frontend is deployed
		{"http://localhost:3000"},
		{"https://www.cardstack.my.id", "https://cardstack.my.id"},
	} {
		assert.NoError(t, App{ClientUrls: ok}.ValidateClientUrls(), "%q", ok)
	}

	for bad, entries := range map[string][]string{
		"https://cardstack.my.id/":       {"https://cardstack.my.id/"},
		"https://cardstack.my.id/app":    {"https://cardstack.my.id/app"},
		"https://cardstack.my.id?x=1":    {"https://cardstack.my.id?x=1"},
		"https://cardstack.my.id#top":    {"https://cardstack.my.id#top"},
		" https://cardstack.my.id":       {"https://x.example", " https://cardstack.my.id"},
		"https://cardstack.my.id ":       {"https://cardstack.my.id "},
		"":                               {"https://cardstack.my.id", ""},
		"cardstack.my.id":                {"cardstack.my.id"},
		"ftp://cardstack.my.id":          {"ftp://cardstack.my.id"},
		"https://":                       {"https://"},
		"\"https://cardstack.my.id\"":    {"\"https://cardstack.my.id\""},
		"https://user@cardstack.my.id":   {"https://user@cardstack.my.id"},
		"https://cardstack.my.id:99999x": {"https://cardstack.my.id:99999x"},
	} {
		assert.ErrorContains(t, App{ClientUrls: entries}.ValidateClientUrls(), fmt.Sprintf("%q", bad))
	}
}
