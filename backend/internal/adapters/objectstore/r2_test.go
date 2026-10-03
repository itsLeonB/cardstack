package objectstore

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func r2TestConfig(endpoint string) config.R2 {
	return config.R2{AccountID: "acct", AccessKeyID: "AKIDEXAMPLE", SecretAccessKey: "secret", Bucket: "images", Endpoint: endpoint}
}

func TestR2Store_Put_WritesObjectWithContentTypeAndCacheControl(t *testing.T) {
	var gotMethod, gotPath, gotType, gotCache, gotAuth string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotType, gotCache, gotAuth = r.Header.Get("Content-Type"), r.Header.Get("Cache-Control"), r.Header.Get("Authorization")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	err := NewR2Store(r2TestConfig(server.URL)).Put(context.Background(), "cards/abc", "image/png", []byte("png-bytes"))

	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "/images/cards/abc", gotPath)
	assert.Equal(t, "image/png", gotType)
	assert.Equal(t, immutableCacheControl, gotCache)
	assert.Contains(t, gotAuth, "AKIDEXAMPLE")
	assert.Equal(t, []byte("png-bytes"), gotBody)
}

func TestR2Store_Put_ReturnsErrorOnFailureStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	err := NewR2Store(r2TestConfig(server.URL)).Put(context.Background(), "cards/abc", "image/png", []byte("x"))

	assert.Error(t, err)
}
