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
