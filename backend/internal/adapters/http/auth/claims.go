package auth

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type callerKey struct{}

// Caller is who sent the request. The zero value is a Guest: a request with no
// Authorization header on a route that allows guests.
type Caller struct {
	UserID    uuid.UUID
	ProfileID uuid.UUID
}

func (c Caller) IsGuest() bool { return c.ProfileID == uuid.Nil }

// WithCaller stashes the authenticated caller for the handlers behind Guard.
func WithCaller(ctx huma.Context, caller Caller) huma.Context {
	return huma.WithValue(ctx, callerKey{}, caller)
}

// CallerFrom returns the caller Guard stashed, or the zero-value Guest when
// there is none.
func CallerFrom(ctx context.Context) Caller {
	caller, _ := ctx.Value(callerKey{}).(Caller)
	return caller
}
