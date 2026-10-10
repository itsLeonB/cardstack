package embedding

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ embedding.ImageFetcher = (*HTTPImageFetcher)(nil)

func TestHTTPImageFetcher_ReturnsTheBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/cards/abc.png", r.URL.Path)
		_, _ = w.Write([]byte("image-bytes"))
	}))
	t.Cleanup(srv.Close)

	got, err := NewHTTPImageFetcher().Fetch(context.Background(), srv.URL+"/cards/abc.png")
	require.NoError(t, err)
	assert.Equal(t, []byte("image-bytes"), got)
}

func TestHTTPImageFetcher_FailsOnNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	_, err := NewHTTPImageFetcher().Fetch(context.Background(), srv.URL+"/missing.png")
	require.Error(t, err)
}

func TestHTTPImageFetcher_FailsOverTheSizeLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", maxImageBytes+1)))
	}))
	t.Cleanup(srv.Close)

	_, err := NewHTTPImageFetcher().Fetch(context.Background(), srv.URL+"/huge.png")
	require.Error(t, err)
}
