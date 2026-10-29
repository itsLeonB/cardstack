package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryAfterSeconds_RoundsUpToAWholeSecond(t *testing.T) {
	assert.Equal(t, 1, retryAfterSeconds(time.Millisecond))
	assert.Equal(t, 1, retryAfterSeconds(time.Second))
	assert.Equal(t, 2, retryAfterSeconds(1500*time.Millisecond))
}

func TestClientKey_DropsThePortSoOneIPSharesABucket(t *testing.T) {
	l := NewLimiter(60, 1)

	ok, _ := l.Allow(clientKey("203.0.113.7:50001"), t0)
	assert.True(t, ok)
	ok, _ = l.Allow(clientKey("203.0.113.7:50002"), t0)
	assert.False(t, ok, "a second connection from the same IP must spend the same allowance")
}

func TestClientKey_LeavesANonHostPortValueUnchanged(t *testing.T) {
	assert.Equal(t, "203.0.113.7", clientKey("203.0.113.7"), "the proxy-set bare IP")
	assert.Equal(t, "2001:db8::1", clientKey("2001:db8::1"), "a bare IPv6 address has colons but no port")
	assert.Equal(t, "2001:db8::1", clientKey("[2001:db8::1]:443"), "a bracketed IPv6 host:port drops the port")
}
