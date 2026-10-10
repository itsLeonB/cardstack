package embedding

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

// fakeGemini serves embedContent, recording each request body and answering
// with the handler's response. Handlers run on the server's goroutines, so
// request bodies are guarded by a mutex and assertions use t.Errorf.
type fakeGemini struct {
	calls   atomic.Int32
	mu      sync.Mutex
	bodies  [][]byte
	handler func(w http.ResponseWriter, r *http.Request, call int)
}

func (f *fakeGemini) body(i int) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.bodies[i]
}

func newFakeGemini(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, call int)) (*fakeGemini, *GeminiEmbedder) {
	t.Helper()
	fake := &fakeGemini{handler: handler}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := int(fake.calls.Add(1))
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
		}
		fake.mu.Lock()
		fake.bodies = append(fake.bodies, body)
		fake.mu.Unlock()
		fake.handler(w, r, call)
	}))
	t.Cleanup(srv.Close)

	g := NewGeminiEmbedder("test-key", "gemini-embedding-2")
	g.baseURL = srv.URL + "/v1beta"
	g.limiter = rate.NewLimiter(rate.Inf, 1)
	g.retryBase = time.Millisecond
	return fake, g
}

func respondVector(t *testing.T, w http.ResponseWriter, values []float32) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"embedding": map[string]any{"values": values}}); err != nil {
		t.Errorf("encoding response: %v", err)
	}
}

// firstAxes returns a 1536-value vector that is 3 on axis 0 and 4 on axis 1,
// so its normalised form is exactly 0.6 and 0.8.
func firstAxes() []float32 {
	v := make([]float32, geminiDimensions)
	v[0], v[1] = 3, 4
	return v
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	return buf.Bytes()
}

func jpegBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4)), nil))
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

func TestGeminiEmbedder_SendsOneImageAndNormalisesTheVector(t *testing.T) {
	png1 := pngBytes(t)
	fake, g := newFakeGemini(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1beta/models/gemini-embedding-2:embedContent", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-goog-api-key"))
		respondVector(t, w, firstAxes())
	})

	vector, err := g.Embed(context.Background(), png1)
	require.NoError(t, err)
	require.Len(t, vector, geminiDimensions)
	assert.InDelta(t, 0.6, vector[0], 1e-6)
	assert.InDelta(t, 0.8, vector[1], 1e-6)

	var sent map[string]any
	require.NoError(t, json.Unmarshal(fake.body(0), &sent))
	assert.Equal(t, float64(geminiDimensions), sent["output_dimensionality"])
	assert.NotContains(t, sent, "task_type", "task_type is unsupported by gemini-embedding-2")

	parts := sent["content"].(map[string]any)["parts"].([]any)
	require.Len(t, parts, 1, "one image per request, no text part")
	inline := parts[0].(map[string]any)["inline_data"].(map[string]any)
	assert.Equal(t, "image/png", inline["mime_type"])
	assert.Equal(t, base64.StdEncoding.EncodeToString(png1), inline["data"])
}

func TestGeminiEmbedder_SendsSupportedFormatsAsTheyAre(t *testing.T) {
	tests := []struct {
		name     string
		image    []byte
		wantMime string
	}{
		{"png", pngBytes(t), "image/png"},
		{"jpeg", jpegBytes(t), "image/jpeg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake, g := newFakeGemini(t, func(w http.ResponseWriter, r *http.Request, _ int) {
				respondVector(t, w, firstAxes())
			})

			_, err := g.Embed(context.Background(), tt.image)
			require.NoError(t, err)

			var sent map[string]any
			require.NoError(t, json.Unmarshal(fake.body(0), &sent))
			inline := sent["content"].(map[string]any)["parts"].([]any)[0].(map[string]any)["inline_data"].(map[string]any)
			assert.Equal(t, tt.wantMime, inline["mime_type"])
			assert.Equal(t, base64.StdEncoding.EncodeToString(tt.image), inline["data"])
		})
	}
}

func TestGeminiEmbedder_ConvertsGIFAndWebPToPNG(t *testing.T) {
	tests := []struct {
		name  string
		image []byte
	}{
		{"gif", gifBytes(t)},
		{"webp", webpBytes(t)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake, g := newFakeGemini(t, func(w http.ResponseWriter, r *http.Request, _ int) {
				respondVector(t, w, firstAxes())
			})

			_, err := g.Embed(context.Background(), tt.image)
			require.NoError(t, err)

			var sent map[string]any
			require.NoError(t, json.Unmarshal(fake.body(0), &sent))
			inline := sent["content"].(map[string]any)["parts"].([]any)[0].(map[string]any)["inline_data"].(map[string]any)
			assert.Equal(t, "image/png", inline["mime_type"])

			decoded, err := base64.StdEncoding.DecodeString(inline["data"].(string))
			require.NoError(t, err)
			_, err = png.Decode(bytes.NewReader(decoded))
			require.NoError(t, err, "the sent bytes are a PNG")
		})
	}
}

func TestGeminiEmbedder_RejectsUnsupportedImageWithoutCallingGemini(t *testing.T) {
	fake, g := newFakeGemini(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		respondVector(t, w, firstAxes())
	})

	_, err := g.Embed(context.Background(), []byte("definitely not an image"))
	require.Error(t, err)
	assert.Zero(t, fake.calls.Load())
}

func TestGeminiEmbedder_RetriesRateLimitHonouringRetryDelay(t *testing.T) {
	fake, g := newFakeGemini(t, func(w http.ResponseWriter, r *http.Request, call int) {
		if call == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":429,"message":"quota per minute","details":[{"retryDelay":"1s"}]}}`))
			return
		}
		respondVector(t, w, firstAxes())
	})

	start := time.Now()
	_, err := g.Embed(context.Background(), pngBytes(t))
	require.NoError(t, err)
	assert.Equal(t, int32(2), fake.calls.Load())
	assert.GreaterOrEqual(t, time.Since(start), 2*time.Second, "the retryDelay from the body is honoured, plus a second of margin")
}

func TestGeminiEmbedder_DailyQuotaStopsWithoutRetry(t *testing.T) {
	fake, g := newFakeGemini(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"Quota exceeded for metric: EmbedContentRequestsPerDayPerUserPerProjectPerModel"}}`))
	})

	_, err := g.Embed(context.Background(), pngBytes(t))
	require.ErrorIs(t, err, embedding.ErrDailyQuotaExhausted)
	assert.ErrorContains(t, err, "PerDay", "the body is carried for the stop log")
	assert.Equal(t, int32(1), fake.calls.Load(), "a spent daily quota is not retried")
}

func TestGeminiEmbedder_RetriesServerErrorsThenGivesUp(t *testing.T) {
	fake, g := newFakeGemini(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := g.Embed(context.Background(), pngBytes(t))
	require.Error(t, err)
	assert.False(t, errors.Is(err, embedding.ErrDailyQuotaExhausted))
	assert.Equal(t, int32(geminiMaxRetries+1), fake.calls.Load())
}

func TestGeminiEmbedder_ClientErrorFailsWithoutRetry(t *testing.T) {
	fake, g := newFakeGemini(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad request"}}`))
	})

	_, err := g.Embed(context.Background(), pngBytes(t))
	require.Error(t, err)
	assert.Equal(t, int32(1), fake.calls.Load())
}

func TestGeminiEmbedder_RejectsWrongDimensionCount(t *testing.T) {
	fake, g := newFakeGemini(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		respondVector(t, w, make([]float32, 768))
	})

	_, err := g.Embed(context.Background(), pngBytes(t))
	require.Error(t, err)
	assert.Equal(t, int32(1), fake.calls.Load())
}

func TestGeminiEmbedder_SpacesRequestsByTheLimiter(t *testing.T) {
	_, g := newFakeGemini(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		respondVector(t, w, firstAxes())
	})
	g.limiter = rate.NewLimiter(rate.Every(50*time.Millisecond), 1)

	start := time.Now()
	for range 3 {
		_, err := g.Embed(context.Background(), pngBytes(t))
		require.NoError(t, err)
	}
	assert.GreaterOrEqual(t, time.Since(start), 100*time.Millisecond, "three requests at one per 50ms take at least two intervals")
}
