package config

import (
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
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
