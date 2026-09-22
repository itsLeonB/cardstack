package auth

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	authkit "github.com/itsLeonB/go-authkit"
)

// ProfileLookup resolves a user_profiles row's ID for a user_id. SessionGuard
// calls it after verifying the access token, rather than reading profile_id
// out of the token itself: users is the auth table, user_profiles is domain
// data, and there's deliberately no FK between them (see
// entity/user_profile.go's doc comment) — so profile_id is looked up fresh
// per request instead of being baked into the JWT.
type ProfileLookup interface {
	FindProfileIDByUserID(ctx context.Context, userID string) (string, error)
}

// SessionGuard ports authgin.AuthMiddleware's access-token check to Huma:
// read the access-token and fingerprint cookies, verify them against kit,
// resolve profile_id via ProfileLookup, and stash the resulting
// userID/sessionID/email/profileID claims into the request context for
// downstream handlers (see claims.go). A request without a valid session,
// or a valid session with no resolvable profile, is rejected with 401 and
// never reaches the handler.
//
// It reads the fingerprint cookie by transport.FingerprintCookieName()
// rather than a fixed name, so it always matches whichever name SetCookies
// actually wrote (plain vs. "__Secure-" prefixed, per CookieSecure).
func SessionGuard(api huma.API, kit *authkit.AuthKit, transport *Transport, profiles ProfileLookup) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		token, err := readCookie(ctx, accessTokenCookie)
		if err != nil || token == "" {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "missing access token")
			return
		}

		fingerprint, _ := readCookie(ctx, transport.FingerprintCookieName())

		claims, err := kit.VerifyToken(ctx.Context(), token, fingerprint)
		if err != nil {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "invalid or expired session")
			return
		}

		userID, _ := claims[authkit.ClaimUserID].(string)
		sessionID, _ := claims[authkit.ClaimSessionID].(string)
		if userID == "" || sessionID == "" {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "invalid session claims")
			return
		}
		email, _ := claims[EmailClaim].(string)

		profileID, err := profiles.FindProfileIDByUserID(ctx.Context(), userID)
		if err != nil {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "invalid session claims")
			return
		}

		next(WithClaims(ctx, userID, sessionID, email, profileID))
	}
}

// CSRFGuard ports authgin.CSRFMiddleware's double-submit check to Huma: on
// any request past GET/HEAD/OPTIONS, the csrf_token cookie must match the
// X-CSRF-Token header. Applied only to logout/refresh — register/login are
// what create the CSRF cookie in the first place, so there's nothing to
// double-submit against yet on that first call (see the plan's "CSRF"
// decision).
func CSRFGuard(api huma.API) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		switch ctx.Method() {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next(ctx)
			return
		}

		cookieToken, err := readCookie(ctx, csrfTokenCookie)
		if err != nil || cookieToken == "" {
			_ = huma.WriteErr(api, ctx, http.StatusForbidden, "missing CSRF token")
			return
		}

		headerToken := ctx.Header("X-CSRF-Token")
		if headerToken == "" || headerToken != cookieToken {
			_ = huma.WriteErr(api, ctx, http.StatusForbidden, "invalid CSRF token")
			return
		}

		next(ctx)
	}
}

func readCookie(ctx huma.Context, name string) (string, error) {
	c, err := huma.ReadCookie(ctx, name)
	if err != nil {
		return "", err
	}
	return c.Value, nil
}
