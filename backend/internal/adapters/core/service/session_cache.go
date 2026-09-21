package service

import (
	"sync"
	"time"

	authkit "github.com/itsLeonB/go-authkit"
)

// sessionCacheTTL bounds how long a session's userID stays cached before
// the next lookup re-validates it against the SessionStore. A revoked
// session (logout) is evicted immediately via Delete regardless of this
// TTL — it only bounds staleness between explicit invalidations.
const sessionCacheTTL = 5 * time.Minute

// sweepInterval is how often expired entries are purged in the background.
const sweepInterval = time.Minute

type cacheEntry struct {
	userID    string
	expiresAt time.Time
}

// SessionCache is a minimal in-process implementation of
// authkit.SessionCache, backed by sync.Map with TTL eviction. MVP runs a
// single Railway instance, so there is no multi-instance cache-consistency
// requirement yet that would justify a shared cache like Redis — see the
// ticket 02 plan's "Session cache" decision. Revisit if the backend ever
// scales beyond one instance.
type SessionCache struct {
	entries sync.Map // sessionID (string) -> cacheEntry
	stop    chan struct{}
	once    sync.Once
}

// NewSessionCache creates a SessionCache and starts its background sweeper.
// Call Shutdown to stop the sweeper and release resources.
func NewSessionCache() authkit.SessionCache {
	c := &SessionCache{stop: make(chan struct{})}
	go c.sweep()
	return c
}

// Get returns the cached userID for sessionID. On a cache miss (or an
// expired entry), it calls loader and caches a successful result.
func (c *SessionCache) Get(sessionID string, loader func(string) (string, bool)) (string, bool) {
	if v, ok := c.entries.Load(sessionID); ok {
		if entry, ok := v.(cacheEntry); ok && time.Now().Before(entry.expiresAt) {
			return entry.userID, true
		}
		c.entries.Delete(sessionID)
	}

	userID, ok := loader(sessionID)
	if !ok {
		return "", false
	}

	c.entries.Store(sessionID, cacheEntry{userID: userID, expiresAt: time.Now().Add(sessionCacheTTL)})
	return userID, true
}

// Delete evicts a session from the cache (used on logout).
func (c *SessionCache) Delete(sessionID string) {
	c.entries.Delete(sessionID)
}

// Shutdown stops the background eviction sweeper. Safe to call more than
// once.
func (c *SessionCache) Shutdown() error {
	c.once.Do(func() { close(c.stop) })
	return nil
}

func (c *SessionCache) sweep() {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			return
		case now := <-ticker.C:
			c.entries.Range(func(key, value any) bool {
				if entry, ok := value.(cacheEntry); ok && now.After(entry.expiresAt) {
					c.entries.Delete(key)
				}
				return true
			})
		}
	}
}
