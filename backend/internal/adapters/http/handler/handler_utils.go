package handler

import (
	"context"

	"github.com/google/uuid"
	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/ungerr"
)

// requireProfileID returns the authenticated caller's profile ID, which
// authpkg.Guard stashed in ctx; a Guest here is unreachable behind a private
// route's guard.
func requireProfileID(ctx context.Context) (uuid.UUID, error) {
	caller := authpkg.CallerFrom(ctx)
	if caller.IsGuest() {
		return uuid.Nil, ungerr.UnauthorizedError("authentication required")
	}

	return caller.ProfileID, nil
}
