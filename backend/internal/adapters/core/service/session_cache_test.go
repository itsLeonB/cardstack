package service

import (
	"testing"
	"time"
)

func TestSessionCache_GetCachesLoaderResult(t *testing.T) {
	c := &SessionCache{stop: make(chan struct{})}
	defer c.Shutdown() //nolint:errcheck

	calls := 0
	loader := func(string) (string, bool) {
		calls++
		return "user-1", true
	}

	userID, hit := c.Get("session-1", loader)
	if !hit || userID != "user-1" {
		t.Fatalf("expected hit with user-1, got hit=%v userID=%q", hit, userID)
	}
	if calls != 1 {
		t.Fatalf("expected loader called once, got %d", calls)
	}

	// Second call should hit the cache, not the loader.
	userID, hit = c.Get("session-1", loader)
	if !hit || userID != "user-1" {
		t.Fatalf("expected cached hit with user-1, got hit=%v userID=%q", hit, userID)
	}
	if calls != 1 {
		t.Fatalf("expected loader still called once after a cache hit, got %d", calls)
	}
}

func TestSessionCache_GetLoaderMiss(t *testing.T) {
	c := &SessionCache{stop: make(chan struct{})}
	defer c.Shutdown() //nolint:errcheck

	userID, hit := c.Get("missing-session", func(string) (string, bool) { return "", false })
	if hit || userID != "" {
		t.Fatalf("expected a miss, got hit=%v userID=%q", hit, userID)
	}
}

func TestSessionCache_Delete(t *testing.T) {
	c := &SessionCache{stop: make(chan struct{})}
	defer c.Shutdown() //nolint:errcheck

	calls := 0
	loader := func(string) (string, bool) {
		calls++
		return "user-1", true
	}

	if _, hit := c.Get("session-1", loader); !hit {
		t.Fatal("expected initial hit")
	}

	c.Delete("session-1")

	if _, hit := c.Get("session-1", loader); !hit {
		t.Fatal("expected hit after re-loading a deleted session")
	}
	if calls != 2 {
		t.Fatalf("expected loader called again after Delete, got %d calls", calls)
	}
}

func TestSessionCache_ExpiredEntryReloads(t *testing.T) {
	c := &SessionCache{stop: make(chan struct{})}
	defer c.Shutdown() //nolint:errcheck

	// Seed an already-expired entry directly, bypassing the TTL constant so
	// the test doesn't need to sleep for real minutes.
	c.entries.Store("session-1", cacheEntry{userID: "stale-user", expiresAt: time.Now().Add(-time.Second)})

	calls := 0
	userID, hit := c.Get("session-1", func(string) (string, bool) {
		calls++
		return "fresh-user", true
	})
	if !hit || userID != "fresh-user" {
		t.Fatalf("expected a reload past expiry with fresh-user, got hit=%v userID=%q", hit, userID)
	}
	if calls != 1 {
		t.Fatalf("expected loader called once for the expired entry, got %d", calls)
	}
}

func TestSessionCache_ShutdownStopsSweep(t *testing.T) {
	c := &SessionCache{stop: make(chan struct{})}
	go c.sweep()

	if err := c.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	// Calling Shutdown twice must not panic (close on a closed channel).
	if err := c.Shutdown(); err != nil {
		t.Fatalf("second Shutdown: %v", err)
	}
}
