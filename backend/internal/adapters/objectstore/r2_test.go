package objectstore

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	corestore "github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
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

func TestR2Store_PresignPut_SignsKeyContentTypeAndExpiry(t *testing.T) {
	store := NewR2Store(r2TestConfig("https://acct.r2.test"))

	rawURL, expiresAt, err := store.PresignPut(context.Background(), "cards/abc", "image/webp", 5*time.Minute)

	require.NoError(t, err)
	u, err := url.Parse(rawURL)
	require.NoError(t, err)
	assert.Equal(t, "acct.r2.test", u.Host)
	assert.Equal(t, "/images/cards/abc", u.Path)
	assert.Equal(t, "300", u.Query().Get("X-Amz-Expires"))
	assert.Contains(t, u.Query().Get("X-Amz-SignedHeaders"), "content-type", "the content type is part of the signature")
	assert.NotEmpty(t, u.Query().Get("X-Amz-Signature"))
	assert.WithinDuration(t, time.Now().Add(5*time.Minute), expiresAt, 5*time.Second)
}

func TestR2Store_ReadHead_RequestsOnlyTheFirstBytes(t *testing.T) {
	var gotRange string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRange = r.Header.Get("Range")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("RIFFxxxxWEBP"))
	}))
	t.Cleanup(server.Close)

	head, err := NewR2Store(r2TestConfig(server.URL)).ReadHead(context.Background(), "cards/abc", 12)

	require.NoError(t, err)
	assert.Equal(t, "bytes=0-11", gotRange)
	assert.Equal(t, []byte("RIFFxxxxWEBP"), head)
}

func TestR2Store_ReadHead_ReportsAMissingObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<?xml version="1.0"?><Error><Code>NoSuchKey</Code><Message>gone</Message></Error>`))
	}))
	t.Cleanup(server.Close)

	_, err := NewR2Store(r2TestConfig(server.URL)).ReadHead(context.Background(), "cards/abc", 12)

	assert.ErrorIs(t, err, corestore.ErrNotFound)
}

func TestR2Store_Delete_RemovesTheObject(t *testing.T) {
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	err := NewR2Store(r2TestConfig(server.URL)).Delete(context.Background(), "cards/abc")

	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/images/cards/abc", gotPath)
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

func TestR2Store_Size_ReportsTheObjectLength(t *testing.T) {
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Header().Set("Content-Length", "1234")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	size, err := NewR2Store(r2TestConfig(server.URL)).Size(context.Background(), "scan/abc")

	require.NoError(t, err)
	assert.Equal(t, http.MethodHead, gotMethod)
	assert.Equal(t, "/images/scan/abc", gotPath)
	assert.Equal(t, int64(1234), size)
}

func TestR2Store_Size_ReportsAMissingObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	_, err := NewR2Store(r2TestConfig(server.URL)).Size(context.Background(), "scan/abc")

	assert.ErrorIs(t, err, corestore.ErrNotFound)
}
