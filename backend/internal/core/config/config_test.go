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
	t.Setenv("MATCH_EMPTY_THRESHOLD", "0.4")

	var set Match
	require.NoError(t, envconfig.Process(set.Prefix(), &set))
	assert.Equal(t, Match{Threshold: 0.9, Margin: 0.05, EmptyThreshold: 0.4}, set)
}

func TestMatch_ValidateMatch(t *testing.T) {
	assert.NoError(t, Match{Threshold: 1, Margin: 0}.ValidateMatch())
	assert.NoError(t, Match{Threshold: 0.01, Margin: 0.99}.ValidateMatch())
	assert.NoError(t, Match{Threshold: 0.85, Margin: 0.03, EmptyThreshold: 0.84}.ValidateMatch())

	for name, m := range map[string]Match{
		"zero threshold":       {Threshold: 0, Margin: 0.03},
		"negative threshold":   {Threshold: -0.1, Margin: 0.03},
		"threshold over one":   {Threshold: 1.1, Margin: 0.03},
		"negative margin":      {Threshold: 0.85, Margin: -0.01},
		"margin of one":        {Threshold: 0.85, Margin: 1},
		"negative empty":       {Threshold: 0.85, Margin: 0.03, EmptyThreshold: -0.01},
		"empty at threshold":   {Threshold: 0.85, Margin: 0.03, EmptyThreshold: 0.85},
		"empty over threshold": {Threshold: 0.85, Margin: 0.03, EmptyThreshold: 0.9},
	} {
		assert.Error(t, m.ValidateMatch(), name)
	}
	assert.ErrorContains(t, Match{Threshold: 0.85, Margin: 0.03, EmptyThreshold: 0.85}.ValidateMatch(), "MATCH_EMPTY_THRESHOLD")
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

func TestAdminEnvVarName(t *testing.T) {
	t.Setenv("ADMIN_EMAILS", "a@example.com,B@example.com")

	var admin Admin
	require.NoError(t, envconfig.Process(admin.Prefix(), &admin))

	assert.Equal(t, []string{"a@example.com", "B@example.com"}, admin.Emails)
}

func TestAdmin_ValidateAdminEmails_NamesTheBadEntry(t *testing.T) {
	for _, ok := range [][]string{nil, {"a@example.com"}, {"a@example.com", "B.c+d@Example.org"}} {
		assert.NoError(t, Admin{Emails: ok}.ValidateAdminEmails(), "%q", ok)
	}

	for bad, entries := range map[string][]string{
		"":                            {"a@example.com", ""},
		"not-an-email":                {"not-an-email"},
		" a@example.com":              {" a@example.com"},
		"Ann <a@example.com>":         {"Ann <a@example.com>"},
		"a@example.com,b@example.com": {"a@example.com,b@example.com"},
	} {
		assert.ErrorContains(t, Admin{Emails: entries}.ValidateAdminEmails(), fmt.Sprintf("%q", bad))
	}
}

func TestAdminClientUrlsEnvVarName(t *testing.T) {
	t.Setenv("ADMIN_CLIENT_URLS", "https://admin.example.com,http://localhost:3001")

	var admin Admin
	require.NoError(t, envconfig.Process(admin.Prefix(), &admin))

	assert.Equal(t, []string{"https://admin.example.com", "http://localhost:3001"}, admin.AdminClientUrls)
}

func TestAdmin_ValidateAdminClientUrls_RejectsEntriesThatAreNotBareOrigins(t *testing.T) {
	for _, ok := range [][]string{nil, {"http://localhost:3001"}, {"https://admin.example.com"}} {
		assert.NoError(t, Admin{AdminClientUrls: ok}.ValidateAdminClientUrls(), "%q", ok)
	}

	for bad, entries := range map[string][]string{
		"https://admin.example.com/": {"https://admin.example.com/"},
		"admin.example.com":          {"https://www.example.com", "admin.example.com"},
		"https://admin.example.com ": {"https://admin.example.com "},
	} {
		assert.ErrorContains(t, Admin{AdminClientUrls: entries}.ValidateAdminClientUrls(), fmt.Sprintf("%q", bad))
	}
}

func TestConfig_ValidateAdminOrigins_NeedsAnAdminOriginWhenThereAreAdmins(t *testing.T) {
	app := App{ClientUrls: []string{"https://www.example.com"}}
	admin := Admin{Emails: []string{"owner@example.com"}}

	assert.NoError(t, Config{}.ValidateAdminOrigins(), "no admins and no Admin app: the admin routes stay off")
	assert.NoError(t, Config{App: app, Admin: Admin{AdminClientUrls: []string{"https://admin.example.com"}}}.ValidateAdminOrigins())
	assert.ErrorContains(t, Config{App: app, Admin: admin}.ValidateAdminOrigins(), "ADMIN_CLIENT_URLS")
	assert.ErrorContains(t, Config{App: app, Admin: Admin{AdminClientUrls: []string{"https://www.example.com"}}}.ValidateAdminOrigins(), "https://www.example.com", "an origin in both lists, even with no admins")
}

func TestScanPhotoR2EnvVarNames(t *testing.T) {
	t.Setenv("SCAN_PHOTO_R2_ACCOUNT_ID", "acct")
	t.Setenv("SCAN_PHOTO_R2_ACCESS_KEY_ID", "key")
	t.Setenv("SCAN_PHOTO_R2_SECRET_ACCESS_KEY", "secret")
	t.Setenv("SCAN_PHOTO_R2_BUCKET", "photos")

	var photos ScanPhotoR2
	require.NoError(t, envconfig.Process(photos.Prefix(), &photos))

	assert.Equal(t, ScanPhotoR2{AccountID: "acct", AccessKeyID: "key", SecretAccessKey: "secret", Bucket: "photos"}, photos)
	assert.True(t, R2(photos).Configured())
}

func TestConfig_ValidateScanPhotoR2_OnlyWhenConfigured(t *testing.T) {
	public := R2{AccountID: "acct", AccessKeyID: "key", SecretAccessKey: "secret", Bucket: "images"}
	photos := ScanPhotoR2{AccountID: "acct", AccessKeyID: "pkey", SecretAccessKey: "psecret", Bucket: "photos"}

	assert.NoError(t, Config{R2: public}.ValidateScanPhotoR2(), "unset: nothing to check")
	assert.NoError(t, Config{R2: public, ScanPhotoR2: photos}.ValidateScanPhotoR2())
	assert.Error(t, Config{ScanPhotoR2: ScanPhotoR2{Bucket: "photos"}}.ValidateScanPhotoR2(), "a partial set is refused")

	samePublic := photos
	samePublic.Bucket = public.Bucket
	assert.Error(t, Config{R2: public, ScanPhotoR2: samePublic}.ValidateScanPhotoR2(), "the photo bucket must not be the public image bucket")
}
