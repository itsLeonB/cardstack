// Package auth classifies each request as a Guest or an authenticated caller
// from its bearer token. The API holds no credentials, sessions or cookies:
// the identity provider (Clerk, ADR-0015) signs a short-lived token and this
// package only verifies it.
package auth

import (
	"context"

	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
)

// TokenVerifier turns a bearer token into the Auth Identity it carries. A token
// that is malformed, expired, badly signed or minted for another application
// fails with an ungerr.UnauthorizedError; any other error means verification
// itself broke (for example the signing keys were unreachable).
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (dto.AuthIdentity, error)
}
