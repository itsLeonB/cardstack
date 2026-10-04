package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testIdentity = dto.AuthIdentity{Provider: "clerk", Subject: "user_1", Email: "a@example.com"}

func TestIdentityCache_HitsUntilTheTTLElapses(t *testing.T) {
	now := time.Now()
	cache := NewIdentityCache()
	cache.now = func() time.Time { return now }
	caller := dto.CallerIdentity{UserID: uuid.New(), ProfileID: uuid.New()}

	_, ok := cache.Get(testIdentity)
	assert.False(t, ok)

	cache.Set(testIdentity, caller)
	got, ok := cache.Get(testIdentity)
	require.True(t, ok)
	assert.Equal(t, caller, got)

	now = now.Add(identityCacheTTL)
	_, ok = cache.Get(testIdentity)
	assert.False(t, ok, "an entry is stale once its TTL has elapsed")
}

func TestIdentityCache_ChangedEmailMisses(t *testing.T) {
	cache := NewIdentityCache()
	cache.Set(testIdentity, dto.CallerIdentity{UserID: uuid.New(), ProfileID: uuid.New()})

	changed := testIdentity
	changed.Email = "new@example.com"
	_, ok := cache.Get(changed)

	assert.False(t, ok, "a changed email must reach the update path")
}

func TestIdentityCache_EmptiesWhenFull(t *testing.T) {
	cache := NewIdentityCache()
	caller := dto.CallerIdentity{UserID: uuid.New(), ProfileID: uuid.New()}
	first := dto.AuthIdentity{Provider: "clerk", Subject: "first"}

	cache.Set(first, caller)
	for i := 1; i < identityCacheMax; i++ {
		cache.Set(dto.AuthIdentity{Provider: "clerk", Subject: uuid.NewString()}, caller)
	}
	overflow := dto.AuthIdentity{Provider: "clerk", Subject: "overflow"}
	cache.Set(overflow, caller)

	_, ok := cache.Get(first)
	assert.False(t, ok)
	_, ok = cache.Get(overflow)
	assert.True(t, ok)
}
