package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/ungerr"
)

// Guard classifies each request. No Authorization header is a Guest: it
// passes when allowGuests is set and is a 401 otherwise. A valid bearer token
// is authenticated: its user and profile (created on first use) are stashed
// for CallerFrom. A header that is present but not a valid token is always a
// 401, never a Guest, so a client with an expired token refreshes it instead
// of silently losing access.
func Guard(api huma.API, verifier TokenVerifier, users service.UserService, allowGuests bool) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		header := ctx.Header("Authorization")
		if header == "" {
			if allowGuests {
				next(ctx)
				return
			}
			writeErr(api, ctx, ungerr.UnauthorizedError("authentication required"))
			return
		}

		scheme, token, _ := strings.Cut(header, " ")
		if !strings.EqualFold(scheme, "Bearer") || token == "" {
			writeErr(api, ctx, ungerr.UnauthorizedError(invalidTokenMsg))
			return
		}

		identity, err := verifier.Verify(ctx.Context(), token)
		if err != nil {
			writeErr(api, ctx, err)
			return
		}

		caller, err := users.ResolveCaller(ctx.Context(), identity)
		if err != nil {
			writeErr(api, ctx, err)
			return
		}

		next(WithCaller(ctx, Caller{UserID: caller.UserID, ProfileID: caller.ProfileID}))
	}
}

// writeErr answers through the Huma error seam (ADR-0013): an AppError keeps
// its status and safe message, anything else becomes a redacted 500.
func writeErr(api huma.API, ctx huma.Context, err error) {
	status := http.StatusInternalServerError
	if appErr, ok := errors.AsType[ungerr.AppError](err); ok {
		status = appErr.HttpStatus()
	}

	if writeFailure := huma.WriteErr(api, ctx, status, http.StatusText(status), err); writeFailure != nil {
		logger.Errorf("writing auth error response: %v", writeFailure)
	}
}
