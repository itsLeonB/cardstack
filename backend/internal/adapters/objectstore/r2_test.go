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

func TestR2Store_Get_ReadsObject(t *testing.T) {
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte("png-bytes"))
	}))
	t.Cleanup(server.Close)

	body, err := NewR2Store(r2TestConfig(server.URL)).Get(context.Background(), "expansion-sets/abc")

	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/images/expansion-sets/abc", gotPath)
	assert.Equal(t, []byte("png-bytes"), body)
}

func TestR2Store_Get_ReturnsErrorOnMissingObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	_, err := NewR2Store(r2TestConfig(server.URL)).Get(context.Background(), "expansion-sets/abc")

	assert.Error(t, err)
}

func TestR2Store_Get_RejectsObjectOverSizeCap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, maxGetBytes+1))
	}))
	t.Cleanup(server.Close)

	_, err := NewR2Store(r2TestConfig(server.URL)).Get(context.Background(), "expansion-sets/huge")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "larger than")
}
