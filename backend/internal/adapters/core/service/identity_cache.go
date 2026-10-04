package service

import (
	"sync"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
)

// identityCacheTTL bounds how stale a cached mapping can be, for example after
// the user or profile row is deleted.
const identityCacheTTL = time.Minute

// identityCacheMax caps the map; a full cache is emptied rather than evicted
// entry by entry, since a miss only costs one database round trip.
const identityCacheMax = 10_000

type identityKey struct{ provider, subject string }

type cachedCaller struct {
	caller    dto.CallerIdentity
	email     string
	expiresAt time.Time
}

// IdentityCache is the in-process service.IdentityCache. The MVP runs a single
// API replica, so there is no cross-instance consistency to keep; revisit if
// the backend ever scales beyond one.
type IdentityCache struct {
	now     func() time.Time
	mu      sync.Mutex
	entries map[identityKey]cachedCaller
}

func NewIdentityCache() *IdentityCache {
	return &IdentityCache{now: time.Now, entries: map[identityKey]cachedCaller{}}
}

func (c *IdentityCache) Get(identity dto.AuthIdentity) (dto.CallerIdentity, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[identityKey{identity.Provider, identity.Subject}]
	if !ok || entry.email != identity.Email || !c.now().Before(entry.expiresAt) {
		return dto.CallerIdentity{}, false
	}

	return entry.caller, true
}

func (c *IdentityCache) Set(identity dto.AuthIdentity, caller dto.CallerIdentity) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= identityCacheMax {
		clear(c.entries)
	}
	c.entries[identityKey{identity.Provider, identity.Subject}] = cachedCaller{
		caller:    caller,
		email:     identity.Email,
		expiresAt: c.now().Add(identityCacheTTL),
	}
}
