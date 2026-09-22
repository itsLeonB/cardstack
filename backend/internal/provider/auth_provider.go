package provider

import (
	"context"

	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	authkit "github.com/itsLeonB/go-authkit"

	"github.com/google/wire"
)

// AuthSet is the wire provider set for the *authkit.AuthKit instance
// auth_handler.go calls directly (see the ticket 02 plan's "Wiring style"
// decision — no domain-service indirection for auth).
var AuthSet = wire.NewSet(ProvideAuthKit, ProvideAuthConfig)

// ProvideAuthKit builds authkit.Config from config.Global.Auth (mirroring
// ProvideDataSource's existing convention of reading config.Global directly
// at the wire-provider level) and authkit.Deps from the repository
// providers + session cache, then constructs the AuthKit in stateful mode
// (Stateless is the zero value, per docs/adr/0003). VerificationURL and
// ResetPasswordURL are left empty and Deps.Mail is left nil — per
// docs/adr/0004, that's what makes authkit.Register mark new users verified
// immediately instead of emailing a verification link.
//
// The Hooks.ClaimsBuilder embeds the user's email into every issued access
// token: AuthKit has no public "get user by ID" method, so GET /auth/me
// answers from the verified token's own claims (see internal/adapters/http/
// auth/claims.go) rather than a second store lookup.
func ProvideAuthKit(
	cfg config.Auth,
	users authkit.UserStore,
	sessions authkit.SessionStore,
	refresh authkit.RefreshTokenStore,
	tx authkit.Transactor,
	cache authkit.SessionCache,
) (*authkit.AuthKit, func(), error) {
	kit, err := authkit.New(
		authkit.Config{
			JWTSecret:       cfg.JWTSecret,
			JWTIssuer:       cfg.JWTIssuer,
			JWTDuration:     cfg.JWTDuration,
			RefreshTokenTTL: cfg.RefreshTokenTTL,
		},
		authkit.Deps{
			Tx:       tx,
			Users:    users,
			Sessions: sessions,
			Refresh:  refresh,
			Cache:    cache,
		},
		authkit.Hooks{
			ClaimsBuilder: func(ctx context.Context, userID string, baseClaims map[string]any) (map[string]any, error) {
				user, err := users.FindByID(ctx, userID)
				if err != nil {
					return nil, err
				}
				baseClaims[authpkg.EmailClaim] = user.Email
				return baseClaims, nil
			},
		},
	)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		_ = kit.Shutdown()
	}

	return kit, cleanup, nil
}

// ProvideAuthConfig exposes config.Global.Auth as a wire-injectable value.
func ProvideAuthConfig() config.Auth {
	return config.Global.Auth
}
