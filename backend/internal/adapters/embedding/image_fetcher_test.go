package embedding

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ embedding.ImageFetcher = (*HTTPImageFetcher)(nil)

func TestHTTPImageFetcher_ReturnsPNGAndJPEGAsTheyAre(t *testing.T) {
	png := pngBytes(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/cards/abc.png", r.URL.Path)
		_, _ = w.Write(png)
	}))
	t.Cleanup(srv.Close)

	mimeType, got, err := NewHTTPImageFetcher().Fetch(context.Background(), srv.URL+"/cards/abc.png")
	require.NoError(t, err)
	assert.Equal(t, "image/png", mimeType)
	assert.Equal(t, png, got)
}

func TestHTTPImageFetcher_ConvertsGIFAndWebPToPNG(t *testing.T) {
	for name, body := range map[string][]byte{"gif": gifBytes(t), "webp": webpBytes(t)} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(body)
			}))
			t.Cleanup(srv.Close)

			mimeType, got, err := NewHTTPImageFetcher().Fetch(context.Background(), srv.URL+"/card")
			require.NoError(t, err)
			assert.Equal(t, "image/png", mimeType)
			assert.Equal(t, "image/png", http.DetectContentType(got))
		})
	}
}

func TestHTTPImageFetcher_RejectsImageTypesTheProviderDoesNotAccept(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>"))
	}))
	t.Cleanup(srv.Close)

	_, _, err := NewHTTPImageFetcher().Fetch(context.Background(), srv.URL+"/card.svg")
	require.Error(t, err)
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	return buf.Bytes()
}

func gifBytes(t *testing.T) []byte {
	t.Helper()
	paletted := image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{color.Black, color.White})
	var buf bytes.Buffer
	require.NoError(t, gif.Encode(&buf, paletted, nil))
	return buf.Bytes()
}

func webpBytes(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/card.webp")
	require.NoError(t, err)
	return data
}

func TestHTTPImageFetcher_FailsOnNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	_, _, err := NewHTTPImageFetcher().Fetch(context.Background(), srv.URL+"/missing.png")
	require.Error(t, err)
}

func TestHTTPImageFetcher_FailsOverTheSizeLimit(t *testing.T) {
	// A valid PNG header keeps the body a real image, so only the size check can reject it.
	huge := append(pngBytes(t), []byte(strings.Repeat("x", maxImageBytes))...)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(huge)
	}))
	t.Cleanup(srv.Close)

	_, _, err := NewHTTPImageFetcher().Fetch(context.Background(), srv.URL+"/huge.png")
	require.Error(t, err)
}
