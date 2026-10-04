package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var t0 = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

func TestLimiter_AllowsTheBurstThenRefuses(t *testing.T) {
	l := NewLimiter(60, 3)

	for i := range 3 {
		ok, _ := l.Allow("a", t0)
		assert.True(t, ok, "request %d", i+1)
	}

	ok, retryAfter := l.Allow("a", t0)
	assert.False(t, ok)
	assert.Equal(t, time.Second, retryAfter, "60 a minute refills one token a second")
}

func TestLimiter_RetryAfterIsTheTimeUntilOneTokenIsBack(t *testing.T) {
	l := NewLimiter(30, 1)
	l.Allow("a", t0)

	_, retryAfter := l.Allow("a", t0.Add(500*time.Millisecond))
	assert.Equal(t, 1500*time.Millisecond, retryAfter)
}

func TestLimiter_ARefusedRequestTakesNothing(t *testing.T) {
	l := NewLimiter(60, 1)
	l.Allow("a", t0)

	for range 5 {
		_, retryAfter := l.Allow("a", t0)
		assert.Equal(t, time.Second, retryAfter, "hammering must not push the retry further out")
	}

	ok, _ := l.Allow("a", t0.Add(time.Second))
	assert.True(t, ok)
}

func TestLimiter_RecoversAtTheRefillRate(t *testing.T) {
	l := NewLimiter(60, 2)
	l.Allow("a", t0)
	l.Allow("a", t0)

	ok, _ := l.Allow("a", t0.Add(999*time.Millisecond))
	assert.False(t, ok, "not a whole token yet")

	ok, _ = l.Allow("a", t0.Add(time.Second))
	assert.True(t, ok)
	ok, _ = l.Allow("a", t0.Add(time.Second))
	assert.False(t, ok, "only one token came back")

	now := t0.Add(time.Hour)
	for i := range 2 {
		ok, _ = l.Allow("a", now)
		assert.True(t, ok, "a long pause refills to the burst, request %d", i+1)
	}
	ok, _ = l.Allow("a", now)
	assert.False(t, ok, "and never past it")
}

func TestLimiter_KeysHaveTheirOwnBuckets(t *testing.T) {
	l := NewLimiter(60, 1)

	ok, _ := l.Allow("a", t0)
	assert.True(t, ok)
	ok, _ = l.Allow("a", t0)
	assert.False(t, ok)

	ok, _ = l.Allow("b", t0)
	assert.True(t, ok, "a's traffic must not spend b's allowance")
}

func TestLimiter_ForgetsBucketsThatRefilled(t *testing.T) {
	l := NewLimiter(60, 2)
	l.Allow("idle", t0)
	l.Allow("busy", t0)

	l.Allow("busy", t0.Add(sweepInterval))
	assert.Contains(t, l.buckets, "busy", "still below its burst")
	assert.NotContains(t, l.buckets, "idle", "a full bucket is the same as a new one")
}
