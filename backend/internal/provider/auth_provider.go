package provider

import (
	"github.com/google/wire"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
)

// AuthSet is the wire provider set for token verification.
var AuthSet = wire.NewSet(ProvideTokenVerifier)

// ProvideTokenVerifier builds the Clerk verifier. It does not check the Clerk
// settings: the job and the ingesters build this same graph and never verify a
// token, so only the API refuses to boot without them (see Clerk.Validate).
func ProvideTokenVerifier() auth.TokenVerifier {
	cfg := config.Global

	return auth.NewClerkVerifier(cfg.Issuer, cfg.ClientUrls, auth.NewClerkKeys(cfg.SecretKey))
}
