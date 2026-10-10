// Package embedding holds the production adapters behind the embedding seams
// in core/embedding: the Gemini embedding client and the hosted-image fetcher.
package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif" // registers the GIF decoder for image.Decode
	"image/png"
	"io"
	"math"
	mrand "math/rand/v2"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/ungerr"
	_ "golang.org/x/image/webp" // registers the WebP decoder for image.Decode
	"golang.org/x/time/rate"
)

const (
	geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"
	// geminiDimensions is the size requested and stored. The model's default is
	// 3072; 1536 is the largest recommended size under pgvector's 2000-dimension
	// index limit. Gemini normalises truncated sizes itself, but a later
	// truncation of stored vectors must re-normalise them.
	geminiDimensions = 1536
	// geminiRequestsPerMinute stays under the free tier's 100 RPM with headroom.
	geminiRequestsPerMinute = 90
	geminiRequestTimeout    = 60 * time.Second
	geminiMaxRetries        = 7
	geminiRetryBase         = 2 * time.Second
	geminiMaxResponseBytes  = 1 << 20
)

// retryDelayPattern reads the wait Google asks for in a 429 body, e.g.
// "retryDelay": "23s".
var retryDelayPattern = regexp.MustCompile(`"retryDelay":\s*"([\d.]+)s"`)

// GeminiEmbedder embeds one image per request with Gemini's embedContent REST
// endpoint (the request shape proven by the accuracy spike). It paces requests
// to geminiRequestsPerMinute, retries rate limits and server errors with
// jittered backoff, and stops at once on an exhausted daily quota.
type GeminiEmbedder struct {
	baseURL   string
	apiKey    string
	model     string
	client    *http.Client
	limiter   *rate.Limiter
	retryBase time.Duration
}

// NewGeminiEmbedder builds an embedder for model, authenticated with apiKey.
func NewGeminiEmbedder(apiKey, model string) *GeminiEmbedder {
	return &GeminiEmbedder{
		baseURL:   geminiBaseURL,
		apiKey:    apiKey,
		model:     model,
		client:    &http.Client{Timeout: geminiRequestTimeout},
		limiter:   rate.NewLimiter(rate.Every(time.Minute/geminiRequestsPerMinute), 1),
		retryBase: geminiRetryBase,
	}
}

// Embed returns the unit-length embedding of image. Images Gemini does not
// accept (anything but PNG or JPEG) are converted to PNG first.
func (g *GeminiEmbedder) Embed(ctx context.Context, img []byte) ([]float32, error) {
	mimeType, data, err := geminiImage(img)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(embedContentRequest{
		Content:              content{Parts: []part{{InlineData: &inlineData{MimeType: mimeType, Data: data}}}},
		OutputDimensionality: geminiDimensions,
	})
	if err != nil {
		return nil, ungerr.Wrap(err, "encoding embedContent request")
	}

	for attempt := 0; ; attempt++ {
		values, retryAfter, retryable, err := g.post(ctx, body)
		if err == nil {
			return normalise(values)
		}
		if !retryable || attempt >= geminiMaxRetries {
			return nil, err
		}

		wait := max(g.backoff(attempt), retryAfter)
		logger.Warnf("retrying embedContent in %s (attempt %d/%d): %v", wait, attempt+1, geminiMaxRetries, err)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ungerr.Wrap(ctx.Err(), "waiting to retry embedContent")
		case <-timer.C:
		}
	}
}

// post makes one attempt. retryable reports whether another attempt may
// succeed; retryAfter is the wait the provider asked for, if any.
func (g *GeminiEmbedder) post(ctx context.Context, body []byte) (values []float32, retryAfter time.Duration, retryable bool, err error) {
	if err := g.limiter.Wait(ctx); err != nil {
		return nil, 0, false, ungerr.Wrap(err, "waiting for the embedding rate limiter")
	}

	url := g.baseURL + "/models/" + g.model + ":embedContent"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, false, ungerr.Wrap(err, "building embedContent request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, 0, ctx.Err() == nil, ungerr.Wrap(err, "requesting embedContent")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Errorf("closing embedContent response body: %v", err)
		}
	}()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, geminiMaxResponseBytes))
	if err != nil {
		return nil, 0, ctx.Err() == nil, ungerr.Wrap(err, "reading embedContent response")
	}

	switch {
	case resp.StatusCode == http.StatusOK:
		values, err := decodeEmbedding(raw)
		return values, 0, false, err
	case resp.StatusCode == http.StatusTooManyRequests:
		if isDailyQuota(raw) {
			// Not ungerr.Wrap: it does not unwrap, so errors.Is would miss the sentinel.
			return nil, 0, false, fmt.Errorf("%w: %s", embedding.ErrDailyQuotaExhausted, truncate(raw))
		}
		return nil, retryDelay(raw), true, ungerr.Unknownf("embedContent rate limited: HTTP %d: %s", resp.StatusCode, truncate(raw))
	case resp.StatusCode >= http.StatusInternalServerError:
		return nil, 0, true, ungerr.Unknownf("embedContent server error: HTTP %d: %s", resp.StatusCode, truncate(raw))
	default:
		return nil, 0, false, ungerr.Unknownf("embedContent rejected: HTTP %d: %s", resp.StatusCode, truncate(raw))
	}
}

// backoff is exponential from retryBase with jitter over the upper half, so
// retries from parallel runs do not line up.
func (g *GeminiEmbedder) backoff(attempt int) time.Duration {
	base := g.retryBase * time.Duration(1<<attempt)
	half := base / 2
	return half + time.Duration(mrand.Int64N(int64(half)+1))
}

type embedContentRequest struct {
	Content              content `json:"content"`
	OutputDimensionality int     `json:"output_dimensionality"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	InlineData *inlineData `json:"inline_data,omitempty"`
}

type inlineData struct {
	MimeType string `json:"mime_type"`
	Data     []byte `json:"data"`
}

// geminiImage returns the MIME type and bytes Gemini accepts. PNG and JPEG
// pass through untouched; GIF and WebP are decoded and re-encoded as PNG.
func geminiImage(data []byte) (string, []byte, error) {
	switch mimeType := http.DetectContentType(data); mimeType {
	case "image/png", "image/jpeg":
		return mimeType, data, nil
	case "image/gif", "image/webp":
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return "", nil, ungerr.Wrapf(err, "decoding %s image", mimeType)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return "", nil, ungerr.Wrap(err, "encoding image as PNG")
		}
		return "image/png", buf.Bytes(), nil
	default:
		return "", nil, ungerr.Unknownf("image is %q, not PNG, JPEG, GIF or WebP", mimeType)
	}
}

func decodeEmbedding(raw []byte) ([]float32, error) {
	var out struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, ungerr.Wrap(err, "decoding embedContent response")
	}
	if len(out.Embedding.Values) != geminiDimensions {
		return nil, ungerr.Unknownf("embedContent returned %d dimensions, want %d", len(out.Embedding.Values), geminiDimensions)
	}
	return out.Embedding.Values, nil
}

// normalise scales v to unit length. Stored vectors are unit length so that
// cosine distance and dot product agree.
func normalise(v []float32) ([]float32, error) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	norm := math.Sqrt(sum)
	if norm == 0 {
		return nil, ungerr.Unknown("embedContent returned a zero vector")
	}

	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(float64(x) / norm)
	}
	return out, nil
}

func isDailyQuota(raw []byte) bool {
	return bytes.Contains(raw, []byte("PerDay")) || strings.Contains(strings.ToLower(string(raw)), "per day")
}

// retryDelay returns the wait a 429 body asks for plus a second of margin, as
// the spike did, or zero when the body names none.
func retryDelay(raw []byte) time.Duration {
	m := retryDelayPattern.FindSubmatch(raw)
	if m == nil {
		return 0
	}
	secs, err := strconv.ParseFloat(string(m[1]), 64)
	if err != nil {
		return 0
	}
	return time.Duration(secs*float64(time.Second)) + time.Second
}

func truncate(raw []byte) string {
	const limit = 300
	if len(raw) > limit {
		return string(raw[:limit]) + "..."
	}
	return string(raw)
}
