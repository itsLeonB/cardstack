package provider

import (
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvideTokenVerifier_RefusesMissingClerkSettings(t *testing.T) {
	complete := config.Config{
		App:   config.App{ClientUrls: []string{"https://cardstack.example"}},
		Clerk: config.Clerk{SecretKey: "sk_test_x", Issuer: "https://example.clerk.accounts.dev"},
	}
	cases := map[string]func(c *config.Config){
		"no secret key": func(c *config.Config) { c.Clerk.SecretKey = "" },
		"no issuer":     func(c *config.Config) { c.Clerk.Issuer = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := complete
			mutate(&cfg)
			config.Global = &cfg

			_, err := ProvideTokenVerifier()

			assert.Error(t, err)
		})
	}

	config.Global = &complete
	verifier, err := ProvideTokenVerifier()
	require.NoError(t, err)
	assert.NotNil(t, verifier)
}
