package handler

import (
	"context"

	"github.com/google/uuid"
	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/ungerr"
)

// requireProfileID returns the profile ID SessionGuard stashed in ctx; a
// failure here is unreachable behind SessionGuard.
func requireProfileID(ctx context.Context) (uuid.UUID, error) {
	raw, _ := authpkg.ProfileID(ctx)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, ungerr.UnauthorizedError("missing session")
	}

	return id, nil
}
