package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/go-jose/go-jose/v3"
	josejwt "github.com/go-jose/go-jose/v3/jwt"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	"github.com/itsLeonB/ezutil/v2"
	"github.com/itsLeonB/ungerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testIssuer = "https://cardstack.clerk.accounts.dev"
	testOrigin = "https://cardstack.example"
	testKeyID  = "ins_test"
)

func newRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func publicKey(key *rsa.PrivateKey) *clerk.JSONWebKey {
	return &clerk.JSONWebKey{Key: &key.PublicKey, KeyID: testKeyID, Algorithm: "RS256", Use: "sig"}
}

// validClaims is a token Clerk would mint for the test instance and origin.
func validClaims() map[string]any {
	now := time.Now()
	return map[string]any{
		"iss":   testIssuer,
		"sub":   "user_123",
		"azp":   testOrigin,
		"sid":   "sess_1",
		"iat":   now.Add(-time.Minute).Unix(),
		"nbf":   now.Add(-time.Minute).Unix(),
		"exp":   now.Add(time.Minute).Unix(),
		"email": "ada@example.com",
		"name":  "Ada Lovelace",
	}
}

func signToken(t *testing.T, key any, alg jose.SignatureAlgorithm, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: alg, Key: jose.JSONWebKey{Key: key, KeyID: testKeyID}},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", testKeyID),
	)
	require.NoError(t, err)
	token, err := josejwt.Signed(signer).Claims(claims).CompactSerialize()
	require.NoError(t, err)
	return token
}

func newTestVerifier(t *testing.T, key *rsa.PrivateKey) *ClerkVerifier {
	t.Helper()
	keys := mocks.NewMockKeySource(t)
	keys.EXPECT().FindKey(mock.Anything, testKeyID).Return(publicKey(key), nil).Maybe()
	return NewClerkVerifier(testIssuer, []string{testOrigin, "http://localhost:3000"}, keys)
}

func requireUnauthorized(t *testing.T, err error) {
	t.Helper()
	appErr, ok := errors.AsType[ungerr.AppError](err)
	require.True(t, ok, "expected an AppError, got %v", err)
	assert.Equal(t, http.StatusUnauthorized, appErr.HttpStatus())
	assert.Equal(t, invalidTokenMsg, appErr.Details())
}

func TestClerkVerifier_AcceptsAValidToken(t *testing.T) {
	key := newRSAKey(t)

	identity, err := newTestVerifier(t, key).Verify(context.Background(), signToken(t, key, jose.RS256, validClaims()))

	require.NoError(t, err)
	assert.Equal(t, dto.AuthIdentity{Provider: "clerk", Subject: "user_123", Email: "ada@example.com", Name: "Ada Lovelace"}, identity)
}

func TestClerkVerifier_AcceptsAnyConfiguredOriginAndAMissingName(t *testing.T) {
	key := newRSAKey(t)
	claims := validClaims()
	claims["azp"] = "http://localhost:3000"
	delete(claims, "name")

	identity, err := newTestVerifier(t, key).Verify(context.Background(), signToken(t, key, jose.RS256, claims))

	require.NoError(t, err)
	assert.Empty(t, identity.Name)
}

func TestClerkVerifier_RejectsBadTokens(t *testing.T) {
	key := newRSAKey(t)
	now := time.Now()

	cases := map[string]func(claims map[string]any){
		"expired":                  func(c map[string]any) { c["exp"] = now.Add(-time.Hour).Unix() },
		"no expiry":                func(c map[string]any) { delete(c, "exp") },
		"not yet valid":            func(c map[string]any) { c["nbf"] = now.Add(time.Hour).Unix() },
		"another issuer":           func(c map[string]any) { c["iss"] = "https://other.clerk.accounts.dev" },
		"another authorized party": func(c map[string]any) { c["azp"] = "https://evil.example" },
		"no authorized party":      func(c map[string]any) { delete(c, "azp") },
		"no subject":               func(c map[string]any) { delete(c, "sub") },
		"no email claim":           func(c map[string]any) { delete(c, "email") },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			claims := validClaims()
			mutate(claims)

			_, err := newTestVerifier(t, key).Verify(context.Background(), signToken(t, key, jose.RS256, claims))

			requireUnauthorized(t, err)
		})
	}

	t.Run("signed by another key", func(t *testing.T) {
		_, err := newTestVerifier(t, key).Verify(context.Background(), signToken(t, newRSAKey(t), jose.RS256, validClaims()))

		requireUnauthorized(t, err)
	})

	t.Run("HMAC-signed with the public key as the secret", func(t *testing.T) {
		secret := []byte(key.N.String())

		_, err := newTestVerifier(t, key).Verify(context.Background(), signToken(t, secret, jose.HS256, validClaims()))

		requireUnauthorized(t, err)
	})

	t.Run("not a token", func(t *testing.T) {
		_, err := newTestVerifier(t, key).Verify(context.Background(), "garbage")

		requireUnauthorized(t, err)
	})
}

// recordingLogger captures Warnf so a test can see why a token was refused.
type recordingLogger struct {
	ezutil.Logger
	warnings []string
}

func (r *recordingLogger) Warnf(format string, args ...any) {
	r.warnings = append(r.warnings, fmt.Sprintf(format, args...))
}

// A 401 with a valid-looking token is otherwise undiagnosable in production:
// the client only ever sees the generic message, so the reason must be logged.
func TestClerkVerifier_LogsWhyATokenWasRefused(t *testing.T) {
	key := newRSAKey(t)
	claims := validClaims()
	claims["azp"] = "https://www.cardstack.example"
	recorder := &recordingLogger{Logger: logger.Global}
	t.Cleanup(func(previous ezutil.Logger) func() { return func() { logger.Global = previous } }(logger.Global))
	logger.Global = recorder

	_, err := newTestVerifier(t, key).Verify(context.Background(), signToken(t, key, jose.RS256, claims))

	requireUnauthorized(t, err)
	require.Len(t, recorder.warnings, 1)
	assert.Contains(t, recorder.warnings[0], "authorized party")
	assert.Contains(t, recorder.warnings[0], "https://www.cardstack.example")
}

func TestClerkVerifier_RejectsAnUnknownKeyID(t *testing.T) {
	key := newRSAKey(t)
	keys := mocks.NewMockKeySource(t)
	keys.EXPECT().FindKey(mock.Anything, testKeyID).Return(nil, nil)
	verifier := NewClerkVerifier(testIssuer, []string{testOrigin}, keys)

	_, err := verifier.Verify(context.Background(), signToken(t, key, jose.RS256, validClaims()))

	requireUnauthorized(t, err)
}

func TestClerkVerifier_FailedKeyFetchIsNotAnInvalidToken(t *testing.T) {
	key := newRSAKey(t)
	fetchErr := errors.New("clerk unreachable")
	keys := mocks.NewMockKeySource(t)
	keys.EXPECT().FindKey(mock.Anything, testKeyID).Return(nil, fetchErr)
	verifier := NewClerkVerifier(testIssuer, []string{testOrigin}, keys)

	_, err := verifier.Verify(context.Background(), signToken(t, key, jose.RS256, validClaims()))

	assert.ErrorIs(t, err, fetchErr)
	_, isAppErr := errors.AsType[ungerr.AppError](err)
	assert.False(t, isAppErr, "an outage must not look like a bad token, or clients would drop their sessions")
}
