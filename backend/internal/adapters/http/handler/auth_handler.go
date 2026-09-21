package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	authkit "github.com/itsLeonB/go-authkit"
)

// generateCSRFToken mirrors authgin.Handler's own setCSRFCookie: 16 random
// bytes, hex-encoded. Returned to the client both as a cookie (via
// Transport.SetCookies) and in the response body, matching authgin's
// {"message", "csrfToken"} shape — the frontend reads csrf_token from the
// cookie for the double-submit header, but returning it in the body too
// costs nothing and matches the reference handler.
func generateCSRFToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// AuthHandler holds *authkit.AuthKit directly and calls
// Register/Login/Logout/RefreshToken/VerifyToken on it, mapping authkit's
// sentinel errors to this project's HTTP error conventions inline (see the
// ticket 02 plan's "Wiring style" and "Error mapping" decisions) — there is
// no domain-service indirection for auth the way HealthService has one.
type AuthHandler struct {
	kit       *authkit.AuthKit
	transport *authpkg.Transport
}

// NewAuthHandler builds an AuthHandler. It reads config.Global.Auth directly
// (matching setup_sentinel.go's existing convention of reading config.Global
// in the http layer) rather than threading cookie config through the wire
// graph.
func NewAuthHandler(kit *authkit.AuthKit) *AuthHandler {
	return &AuthHandler{
		kit:       kit,
		transport: authpkg.NewTransport(config.Global.Auth),
	}
}

type authMessage struct {
	Message   string `json:"message"`
	CSRFToken string `json:"csrfToken,omitempty"`
}

// --- Register ---

type registerInput struct {
	Body struct {
		Email                string `json:"email" required:"true" format:"email"`
		Password             string `json:"password" required:"true" minLength:"8"`
		PasswordConfirmation string `json:"passwordConfirmation" required:"true"`
	}
}

func (h *AuthHandler) register(ctx context.Context, in registerInput) (authMessage, error) {
	if in.Body.Password != in.Body.PasswordConfirmation {
		return authMessage{}, huma.Error400BadRequest("password and passwordConfirmation do not match")
	}

	verified, err := h.kit.Register(ctx, in.Body.Email, in.Body.Password, "")
	if err != nil {
		return authMessage{}, mapAuthError(err)
	}

	msg := "check your email to confirm your registration"
	if verified {
		msg = "success registering, please login"
	}

	return authMessage{Message: msg}, nil
}

// --- Login ---

type loginInput struct {
	Body struct {
		Email    string `json:"email" required:"true" format:"email"`
		Password string `json:"password" required:"true"`
	}
}

type cookieOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      httpapi.Envelope[authMessage]
}

// cookieResponse builds the Set-Cookie headers + {message, csrfToken} body
// shared by login and refresh's successful responses.
func (h *AuthHandler) cookieResponse(tokens authkit.TokenSet) (*cookieOutput, error) {
	csrfToken, err := generateCSRFToken()
	if err != nil {
		return nil, huma.Error500InternalServerError("error generating csrf token")
	}

	return &cookieOutput{
		SetCookie: h.transport.SetCookies(tokens.AccessToken, tokens.RefreshToken, tokens.Fingerprint, csrfToken),
		Body:      httpapi.NewEnvelope(authMessage{Message: "ok", CSRFToken: csrfToken}),
	}, nil
}

func (h *AuthHandler) registerLogin(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID:   "login",
		Method:        http.MethodPost,
		Path:          "/auth/login",
		Summary:       "Log in with email and password",
		Tags:          []string{"auth"},
		DefaultStatus: http.StatusOK,
		Middlewares:   mw,
	}, func(ctx context.Context, in *loginInput) (*cookieOutput, error) {
		tokens, err := h.kit.Login(ctx, in.Body.Email, in.Body.Password)
		if err != nil {
			return nil, mapAuthError(err)
		}

		return h.cookieResponse(tokens)
	})
}

// --- Logout ---

type logoutInput struct{}

type logoutOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
}

func (h *AuthHandler) registerLogout(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID:   "logout",
		Method:        http.MethodPost,
		Path:          "/auth/logout",
		Summary:       "Log out and clear the current session",
		Tags:          []string{"auth"},
		DefaultStatus: http.StatusNoContent,
		Security:      endpoint.CookieAuthSecurity,
		Middlewares:   mw,
	}, func(ctx context.Context, _ *logoutInput) (*logoutOutput, error) {
		sessionID, ok := authpkg.SessionID(ctx)
		if !ok || sessionID == "" {
			return nil, huma.Error401Unauthorized("missing session")
		}

		if err := h.kit.Logout(ctx, sessionID); err != nil {
			return nil, mapAuthError(err)
		}

		return &logoutOutput{SetCookie: h.transport.ClearCookies()}, nil
	})
}

// --- Refresh ---

type refreshInput struct {
	RefreshToken string `cookie:"refresh_token"`
}

func (h *AuthHandler) registerRefresh(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID:   "refresh-token",
		Method:        http.MethodPost,
		Path:          "/auth/refresh",
		Summary:       "Rotate the refresh token and issue a new access token",
		Tags:          []string{"auth"},
		DefaultStatus: http.StatusOK,
		// No Security: the access token guard doesn't apply here — the whole
		// point of this route is that the access token may be expired. The
		// refresh cookie itself is authkit's real check (kit.RefreshToken
		// returns ErrTokenInvalid/ErrTokenExpired otherwise). CSRFGuard still
		// runs, via mw.
		Middlewares: mw,
	}, func(ctx context.Context, in *refreshInput) (*cookieOutput, error) {
		if in.RefreshToken == "" {
			return nil, huma.Error401Unauthorized("missing refresh token")
		}

		tokens, err := h.kit.RefreshToken(ctx, in.RefreshToken)
		if err != nil {
			return nil, mapAuthError(err)
		}

		return h.cookieResponse(tokens)
	})
}

// --- Me ---

type meInput struct{}

type meResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// me answers from the verified access token's own claims (userID + email,
// see authpkg.EmailClaim) rather than a fresh store lookup — AuthKit has no
// public "get user by ID" method to call here.
func (h *AuthHandler) me(ctx context.Context, _ meInput) (meResponse, error) {
	userID, ok := authpkg.UserID(ctx)
	if !ok || userID == "" {
		return meResponse{}, huma.Error401Unauthorized("missing session")
	}
	email, _ := authpkg.Email(ctx)

	return meResponse{ID: userID, Email: email}, nil
}

// registrableFunc adapts a plain registration function to
// endpoint.Registrable, for routes (login/logout/refresh) that need a
// custom Output shape endpoint.Endpoint can't express (a Set-Cookie response
// header) and so register directly with huma.Register instead of going
// through endpoint.New.
type registrableFunc func(api huma.API, mw ...func(huma.Context, func(huma.Context)))

func (f registrableFunc) Register(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
	f(api, mw...)
}

// withGuards appends route-specific guard middlewares after the shared ones
// RegisterAll passes in, mirroring endpoint.go's own unexported
// mergeMiddlewares — needed here too since logout/refresh/me build their
// guards (which need the live huma.API) at Register-call time rather than
// at Routes()-build time.
func withGuards(shared []func(huma.Context, func(huma.Context)), guards ...func(huma.Context, func(huma.Context))) []func(huma.Context, func(huma.Context)) {
	merged := make([]func(huma.Context, func(huma.Context)), 0, len(shared)+len(guards))
	merged = append(merged, shared...)
	merged = append(merged, guards...)
	return merged
}

// Routes returns every route AuthHandler exposes, for registration via
// endpoint.RegisterAll. Mirrors health_handler.go's shape; see the ticket 02
// plan's "Route surface" section for each route's Secured/Middlewares.
func (h *AuthHandler) Routes() []endpoint.Registrable {
	sessionGuard := func(api huma.API) func(huma.Context, func(huma.Context)) {
		return authpkg.SessionGuard(api, h.kit)
	}

	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[registerInput, authMessage]{
			OperationID: "register",
			Method:      http.MethodPost,
			Path:        "/auth/register",
			Summary:     "Register with email and password",
			Tags:        []string{"auth"},
			SuccessCode: http.StatusCreated,
			Secured:     false,
			HandlerFunc: h.register,
		}),
		registrableFunc(h.registerLogin),
		registrableFunc(func(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
			h.registerLogout(api, withGuards(mw, sessionGuard(api), authpkg.CSRFGuard(api))...)
		}),
		registrableFunc(func(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
			h.registerRefresh(api, withGuards(mw, authpkg.CSRFGuard(api))...)
		}),
		// /auth/me goes through a registrableFunc rather than endpoint.New
		// directly, for the same reason logout/refresh do: Secured:true only
		// sets the OpenAPI security metadata (see endpoint.Register), it
		// doesn't attach SessionGuard as an enforced middleware — and
		// SessionGuard needs the live huma.API (for huma.WriteErr), which
		// Routes() doesn't have until Register is actually called.
		registrableFunc(func(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
			endpoint.Register(api, endpoint.Endpoint[meInput, meResponse]{
				OperationID: "get-current-user",
				Method:      http.MethodGet,
				Path:        "/auth/me",
				Summary:     "Get the current session's user",
				Tags:        []string{"auth"},
				SuccessCode: http.StatusOK,
				Secured:     true,
				HandlerFunc: h.me,
			}, withGuards(mw, sessionGuard(api))...)
		}),
	}
}

func mapAuthError(err error) error {
	switch {
	case errors.Is(err, authkit.ErrUserExists):
		return huma.Error409Conflict(err.Error())
	case errors.Is(err, authkit.ErrInvalidCredentials):
		return huma.Error401Unauthorized(err.Error())
	case errors.Is(err, authkit.ErrUserNotFound):
		return huma.Error404NotFound(err.Error())
	case errors.Is(err, authkit.ErrSessionNotFound):
		return huma.Error401Unauthorized(err.Error())
	case errors.Is(err, authkit.ErrTokenInvalid),
		errors.Is(err, authkit.ErrTokenExpired),
		errors.Is(err, authkit.ErrTokenNotFound):
		return huma.Error401Unauthorized(err.Error())
	case errors.Is(err, authkit.ErrNotVerified):
		return huma.Error403Forbidden(err.Error())
	case errors.Is(err, authkit.ErrTooManyRequests):
		return huma.Error429TooManyRequests(err.Error())
	default:
		return huma.Error500InternalServerError("internal error")
	}
}
