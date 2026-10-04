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
