package ratelimit

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/ungerr"
)

// tooManyRequestsMsg is the whole body detail of a 429: fixed, so it never
// says which bucket ran out or anything about the caller.
const tooManyRequestsMsg = "too many requests, retry later"

// Limits holds one Limiter per tier and the clock they read.
type Limits struct {
	User   *Limiter
	Search *Limiter
	Facets *Limiter
	Now    func() time.Time
}

func NewLimits(cfg config.RateLimit, now func() time.Time) Limits {
	return Limits{
		User:   NewLimiter(cfg.UserPerMinute, cfg.UserBurst),
		Search: NewLimiter(cfg.SearchPerMinute, cfg.SearchBurst),
		Facets: NewLimiter(cfg.FacetsPerMinute, cfg.FacetsBurst),
		Now:    now,
	}
}

// PerUser limits signed-in callers by their verified user id, so one account
// gets one allowance however many addresses it calls from. It must run after
// auth.Guard. An operation listed in byOperation spends only that tier;
// every other operation spends limits.User. Guests pass untouched: the edge
// and the per-IP backstop limit them.
func PerUser(api huma.API, limits Limits, byOperation map[string]*Limiter) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		caller := auth.CallerFrom(ctx.Context())
		if caller.IsGuest() {
			next(ctx)
			return
		}

		limiter, ok := byOperation[ctx.Operation().OperationID]
		if !ok {
			limiter = limits.User
		}

		allowed, retryAfter := limiter.Allow(caller.UserID.String(), limits.Now())
		if allowed {
			next(ctx)
			return
		}

		ctx.SetHeader("Retry-After", strconv.Itoa(retryAfterSeconds(retryAfter)))
		err := ungerr.TooManyRequestsError(tooManyRequestsMsg)
		if writeFailure := huma.WriteErr(api, ctx, http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests), err); writeFailure != nil {
			logger.Errorf("writing rate limit response: %v", writeFailure)
		}
	}
}

// retryAfterSeconds rounds up, since Retry-After takes whole seconds and a
// client retrying early would only be refused again.
func retryAfterSeconds(d time.Duration) int {
	return int(math.Ceil(d.Seconds()))
}
