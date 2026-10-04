package provider

import (
	"errors"

	"github.com/google/wire"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
)

// AuthSet is the wire provider set for token verification.
var AuthSet = wire.NewSet(ProvideTokenVerifier)

// ProvideTokenVerifier builds the Clerk verifier. It refuses to boot without
// the Clerk settings or the frontend origins: with either missing every token
// would fail, which is better reported here than as a 401 for every user.
func ProvideTokenVerifier() (auth.TokenVerifier, error) {
	cfg := config.Global
	if cfg.Clerk.SecretKey == "" || cfg.Clerk.Issuer == "" {
		return nil, errors.New("CLERK_SECRET_KEY and CLERK_ISSUER are required")
	}
	if len(cfg.ClientUrls) == 0 {
		return nil, errors.New("APP_CLIENT_URLS must list the frontend origins Clerk issues tokens for")
	}

	return auth.NewClerkVerifier(cfg.Clerk.Issuer, cfg.ClientUrls, auth.NewClerkKeys(cfg.Clerk.SecretKey)), nil
}
