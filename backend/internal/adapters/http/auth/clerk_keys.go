package auth

import (
	"context"
	"sync"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwks"
	"github.com/itsLeonB/ungerr"
)

const (
	// keyTTL bounds how long a key Clerk has since retired stays trusted.
	keyTTL = time.Hour
	// keyRefetchInterval spaces fetches so tokens with forged kids cannot make
	// every request call Clerk.
	keyRefetchInterval = time.Minute
	// keyFetchTimeout bounds how long FindKey holds its lock on a Clerk call.
	keyFetchTimeout = 5 * time.Second
)

// ClerkKeys is the KeySource backed by the instance's JWKS, fetched with the
// secret key and cached so almost no request calls Clerk.
type ClerkKeys struct {
	client *jwks.Client
	now    func() time.Time

	mu          sync.Mutex
	keys        map[string]*clerk.JSONWebKey
	fetchedAt   time.Time
	attemptedAt time.Time
	lastErr     error
}

func NewClerkKeys(secretKey string) *ClerkKeys {
	config := &clerk.ClientConfig{}
	config.Key = clerk.String(secretKey)

	return &ClerkKeys{client: jwks.NewClient(config), now: time.Now}
}

func (c *ClerkKeys) FindKey(ctx context.Context, keyID string) (*clerk.JSONWebKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if key, ok := c.keys[keyID]; ok && now.Sub(c.fetchedAt) < keyTTL {
		return key, nil
	}
	// A failed fetch is throttled like a successful one, and keeps failing
	// meanwhile: during a Clerk outage requests must not each call Clerk, and an
	// outage must not look like a forged key id.
	if !c.attemptedAt.IsZero() && now.Sub(c.attemptedAt) < keyRefetchInterval {
		return nil, c.lastErr
	}

	c.attemptedAt = now
	// Detached from the caller: one aborted request must not fail the fetch and
	// so poison the throttled error for everyone, and the timeout bounds how long
	// this holds the lock.
	fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), keyFetchTimeout)
	defer cancel()
	set, err := c.client.Get(fetchCtx, &jwks.GetParams{})
	if err != nil {
		c.lastErr = ungerr.Wrap(err, "fetching clerk signing keys")
		return nil, c.lastErr
	}

	c.lastErr = nil
	c.fetchedAt = now
	c.keys = make(map[string]*clerk.JSONWebKey, len(set.Keys))
	for _, key := range set.Keys {
		c.keys[key.KeyID] = key
	}

	return c.keys[keyID], nil
}
