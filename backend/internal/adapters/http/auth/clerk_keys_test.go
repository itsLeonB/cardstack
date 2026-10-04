package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwks"
	"github.com/go-jose/go-jose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestKeys serves one JWKS from a local server standing in for Clerk's
// Backend API and counts the fetches.
func newTestKeys(t *testing.T, status int) (*ClerkKeys, *atomic.Int32, *time.Time) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"keys": []jose.JSONWebKey{
		{Key: &newRSAKey(t).PublicKey, KeyID: testKeyID, Algorithm: "RS256", Use: "sig"},
	}})
	require.NoError(t, err)

	var fetches atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches.Add(1)
		assert.Equal(t, "Bearer sk_test_x", r.Header.Get("Authorization"))
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	config := &clerk.ClientConfig{}
	config.Key = clerk.String("sk_test_x")
	config.URL = clerk.String(server.URL)
	now := time.Now()

	return &ClerkKeys{client: jwks.NewClient(config), now: func() time.Time { return now }}, &fetches, &now
}

func TestClerkKeys_CachesTheKeySet(t *testing.T) {
	keys, fetches, _ := newTestKeys(t, http.StatusOK)

	for range 3 {
		key, err := keys.FindKey(context.Background(), testKeyID)
		require.NoError(t, err)
		require.NotNil(t, key)
		assert.Equal(t, testKeyID, key.KeyID)
	}

	assert.Equal(t, int32(1), fetches.Load())
}

func TestClerkKeys_UnknownKeyDoesNotRefetchWithinTheInterval(t *testing.T) {
	keys, fetches, now := newTestKeys(t, http.StatusOK)

	key, err := keys.FindKey(context.Background(), "forged")
	require.NoError(t, err)
	assert.Nil(t, key)

	key, err = keys.FindKey(context.Background(), "forged-again")
	require.NoError(t, err)
	assert.Nil(t, key)
	assert.Equal(t, int32(1), fetches.Load(), "forged key ids must not make every request call Clerk")

	*now = now.Add(keyRefetchInterval)
	_, err = keys.FindKey(context.Background(), "forged")
	require.NoError(t, err)
	assert.Equal(t, int32(2), fetches.Load())
}

func TestClerkKeys_RefetchesAfterTheKeyTTL(t *testing.T) {
	keys, fetches, now := newTestKeys(t, http.StatusOK)

	_, err := keys.FindKey(context.Background(), testKeyID)
	require.NoError(t, err)
	*now = now.Add(keyTTL)
	key, err := keys.FindKey(context.Background(), testKeyID)

	require.NoError(t, err)
	assert.NotNil(t, key)
	assert.Equal(t, int32(2), fetches.Load(), "a key Clerk retired must not stay trusted forever")
}

func TestClerkKeys_FailedFetchIsAnErrorAndRetriedNextTime(t *testing.T) {
	keys, fetches, _ := newTestKeys(t, http.StatusInternalServerError)

	_, err := keys.FindKey(context.Background(), testKeyID)
	assert.Error(t, err)
	_, err = keys.FindKey(context.Background(), testKeyID)
	assert.Error(t, err)

	assert.Equal(t, int32(2), fetches.Load())
}
