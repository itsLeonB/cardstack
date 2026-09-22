package auth

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

type contextKey int

const (
	userIDKey contextKey = iota
	sessionIDKey
	emailKey
	profileIDKey
)

// EmailClaim is the JWT claim key the auth provider's ClaimsBuilder hook
// adds to every issued access token, carrying the user's email so /auth/me
// can answer from the verified token's claims alone — AuthKit has no public
// "get user by ID" method, only the unexported UserStore it holds
// internally, so this is how the handler learns the email without a second
// store dependency of its own.
const EmailClaim = "email"

// WithClaims stashes the userID/sessionID/email SessionGuard extracted from
// a verified access token, plus the profileID it resolved afterward via
// ProfileLookup (see FindProfileIDByUserID's doc comment — profile_id is
// never itself a JWT claim), into ctx, so handlers downstream of the guard
// can read them via UserID/SessionID/Email/ProfileID.
func WithClaims(ctx huma.Context, userID, sessionID, email, profileID string) huma.Context {
	ctx = huma.WithValue(ctx, userIDKey, userID)
	ctx = huma.WithValue(ctx, sessionIDKey, sessionID)
	ctx = huma.WithValue(ctx, emailKey, email)
	ctx = huma.WithValue(ctx, profileIDKey, profileID)
	return ctx
}

// UserID returns the authenticated user's ID stashed by SessionGuard, and
// whether one was present.
func UserID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	return v, ok
}

// SessionID returns the current session's ID stashed by SessionGuard, and
// whether one was present.
func SessionID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(sessionIDKey).(string)
	return v, ok
}

// Email returns the authenticated user's email stashed by SessionGuard (see
// EmailClaim), and whether one was present.
func Email(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(emailKey).(string)
	return v, ok
}

// ProfileID returns the authenticated user's user_profiles row ID, resolved
// by SessionGuard via ProfileLookup after verifying the access token (see
// WithClaims), and whether one was present.
func ProfileID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(profileIDKey).(string)
	return v, ok
}
