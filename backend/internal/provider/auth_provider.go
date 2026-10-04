package provider

import (
	"errors"

	"github.com/google/wire"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
)

// AuthSet is the wire provider set for token verification.
var AuthSet = wire.NewSet(ProvideTokenVerifier)

// ProvideTokenVerifier builds the Clerk verifier. It refuses to boot without
// the Clerk settings, which would otherwise show up only as a 401 for every
// user. With no frontend origins it boots and rejects every token (a preview
// has none until its frontend is deployed), and logs why.
func ProvideTokenVerifier() (auth.TokenVerifier, error) {
	cfg := config.Global
	if cfg.SecretKey == "" || cfg.Issuer == "" {
		return nil, errors.New("CLERK_SECRET_KEY and CLERK_ISSUER are required")
	}
	if len(cfg.ClientUrls) == 0 {
		logger.Error("APP_CLIENT_URLS is empty: every token will be rejected, since none can come from a configured frontend origin")
	}

	return auth.NewClerkVerifier(cfg.Issuer, cfg.ClientUrls, auth.NewClerkKeys(cfg.SecretKey)), nil
}
