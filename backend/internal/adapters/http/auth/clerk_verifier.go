package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/ungerr"
)

// KeySource finds the public key a token's kid header names.
type KeySource interface {
	// FindKey returns nil with a nil error for a key the provider does not
	// publish, so the caller can tell a forged kid from a failed fetch.
	FindKey(ctx context.Context, keyID string) (*clerk.JSONWebKey, error)
}

var errInvalidToken = errors.New("invalid token")

// customClaims are the two claims the Clerk session token is configured to
// carry (see scripts/clerk-setup.sh). They are trusted because the token is
// signed.
type customClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// ClerkVerifier verifies Clerk session tokens: RS256 signature against the
// instance's published keys, expiry and not-before, the instance's issuer, and
// that the token was minted for one of our frontend origins (azp).
type ClerkVerifier struct {
	issuer            string
	authorizedParties []string
	keys              KeySource
}

func NewClerkVerifier(issuer string, authorizedParties []string, keys KeySource) *ClerkVerifier {
	return &ClerkVerifier{issuer: issuer, authorizedParties: authorizedParties, keys: keys}
}

func (v *ClerkVerifier) Verify(ctx context.Context, token string) (dto.AuthIdentity, error) {
	identity, err := v.verify(ctx, token)
	if errors.Is(err, errInvalidToken) {
		return dto.AuthIdentity{}, ungerr.UnauthorizedError("invalid or expired token")
	}

	return identity, err
}

func (v *ClerkVerifier) verify(ctx context.Context, token string) (dto.AuthIdentity, error) {
	// Decode only reads the kid; nothing it returns is trusted before Verify.
	unverified, err := jwt.Decode(ctx, &jwt.DecodeParams{Token: token})
	if err != nil {
		return dto.AuthIdentity{}, fmt.Errorf("%w: decoding: %w", errInvalidToken, err)
	}

	key, err := v.keys.FindKey(ctx, unverified.KeyID)
	if err != nil {
		return dto.AuthIdentity{}, err
	}
	if key == nil {
		return dto.AuthIdentity{}, fmt.Errorf("%w: unknown key id", errInvalidToken)
	}

	claims, err := jwt.Verify(ctx, &jwt.VerifyParams{
		Token:                   token,
		JWK:                     key,
		CustomClaimsConstructor: func(context.Context) any { return &customClaims{} },
	})
	if err != nil {
		return dto.AuthIdentity{}, fmt.Errorf("%w: verifying: %w", errInvalidToken, err)
	}

	// Verify only checks that the issuer looks like a Clerk one, not that it is
	// ours.
	if claims.Issuer != v.issuer {
		return dto.AuthIdentity{}, fmt.Errorf("%w: issuer", errInvalidToken)
	}
	// A token without azp was not minted for any origin, so it is rejected
	// rather than skipped.
	if !slices.Contains(v.authorizedParties, claims.AuthorizedParty) {
		return dto.AuthIdentity{}, fmt.Errorf("%w: authorized party", errInvalidToken)
	}

	custom, _ := claims.Custom.(*customClaims)
	if claims.Subject == "" || custom == nil || custom.Email == "" {
		return dto.AuthIdentity{}, fmt.Errorf("%w: missing subject or email claim", errInvalidToken)
	}

	return dto.AuthIdentity{
		Provider: dto.AuthProviderClerk,
		Subject:  claims.Subject,
		Email:    custom.Email,
		Name:     custom.Name,
	}, nil
}
