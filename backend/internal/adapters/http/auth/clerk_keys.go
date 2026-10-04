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
)

// ClerkKeys is the KeySource backed by the instance's JWKS, fetched with the
// secret key and cached so almost no request calls Clerk.
type ClerkKeys struct {
	client *jwks.Client
	now    func() time.Time

	mu        sync.Mutex
	keys      map[string]*clerk.JSONWebKey
	fetchedAt time.Time
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
	age := now.Sub(c.fetchedAt)
	if key, ok := c.keys[keyID]; ok && age < keyTTL {
		return key, nil
	}
	if !c.fetchedAt.IsZero() && age < keyRefetchInterval {
		return nil, nil
	}

	set, err := c.client.Get(ctx, &jwks.GetParams{})
	if err != nil {
		return nil, ungerr.Wrap(err, "fetching clerk signing keys")
	}

	c.keys = make(map[string]*clerk.JSONWebKey, len(set.Keys))
	for _, key := range set.Keys {
		c.keys[key.KeyID] = key
	}
	c.fetchedAt = now

	return c.keys[keyID], nil
}
