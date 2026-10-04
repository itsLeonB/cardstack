// Package ratelimit limits signed-in callers per user inside the API, on top
// of the per-IP limits at the edge and the loose in-API per-IP backstop.
//
// Ceiling: buckets live in this process's memory, so a deploy or restart
// resets every allowance, and the limits only hold while the API runs as a
// single replica. Scaling out needs a shared store (Redis) behind Limiter.
package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// sweepInterval is how often Allow drops buckets that have refilled, which
// keeps memory proportional to recently active users, not to all users ever.
const sweepInterval = time.Minute

// Limiter is a token bucket per key: each key may spend burst requests at
// once, and gets perMinute of them back every minute, up to burst.
type Limiter struct {
	limit rate.Limit
	burst int

	mu        sync.Mutex
	buckets   map[string]*rate.Limiter
	lastSweep time.Time
}

func NewLimiter(perMinute, burst int) *Limiter {
	return &Limiter{
		limit:   rate.Limit(float64(perMinute) / time.Minute.Seconds()),
		burst:   burst,
		buckets: map[string]*rate.Limiter{},
	}
}

// Allow spends one of key's tokens at now. When none is left it spends
// nothing and returns how long until one is back.
func (l *Limiter) Allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.sweep(now)

	bucket, ok := l.buckets[key]
	if !ok {
		bucket = rate.NewLimiter(l.limit, l.burst)
		l.buckets[key] = bucket
	}

	res := bucket.ReserveN(now, 1)
	if delay := res.DelayFrom(now); delay > 0 {
		// Cancelling returns the token, so a refused request costs nothing and
		// hammering cannot push the retry further out.
		res.CancelAt(now)
		return false, delay
	}
	return true, 0
}

// sweep drops every full bucket: a full bucket behaves exactly like a new one.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < sweepInterval {
		return
	}
	l.lastSweep = now

	for key, bucket := range l.buckets {
		if bucket.TokensAt(now) >= float64(l.burst) {
			delete(l.buckets, key)
		}
	}
}
