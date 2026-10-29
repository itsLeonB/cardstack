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
	Shared *Limiter
	Now    func() time.Time
}

func NewLimits(cfg config.RateLimit, now func() time.Time) Limits {
	return Limits{
		User:   NewLimiter(cfg.User.PerMinute, cfg.User.Burst),
		Search: NewLimiter(cfg.Search.PerMinute, cfg.Search.Burst),
		Facets: NewLimiter(cfg.Facets.PerMinute, cfg.Facets.Burst),
		Shared: NewLimiter(cfg.Shared.PerMinute, cfg.Shared.Burst),
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

		refuse(api, ctx, retryAfter)
	}
}

// PerIP limits every caller of the routes it guards by client address, guest
// or signed in, so a shared page spends one allowance per address. Unlike
// PerUser it does not read the caller, so it needs no auth.Guard before it.
func PerIP(api huma.API, limiter *Limiter, now func() time.Time) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		allowed, retryAfter := limiter.Allow(ctx.RemoteAddr(), now())
		if allowed {
			next(ctx)
			return
		}

		refuse(api, ctx, retryAfter)
	}
}

// refuse writes the 429 both limiters answer with: a whole-second Retry-After
// and a body that says nothing about the caller.
func refuse(api huma.API, ctx huma.Context, retryAfter time.Duration) {
	ctx.SetHeader("Retry-After", strconv.Itoa(retryAfterSeconds(retryAfter)))
	err := ungerr.TooManyRequestsError(tooManyRequestsMsg)
	if writeFailure := huma.WriteErr(api, ctx, http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests), err); writeFailure != nil {
		logger.Errorf("writing rate limit response: %v", writeFailure)
	}
}

// retryAfterSeconds rounds up, since Retry-After takes whole seconds and a
// client retrying early would only be refused again.
func retryAfterSeconds(d time.Duration) int {
	return int(math.Ceil(d.Seconds()))
}
