package embedding

import (
	"bytes"
	"context"
	"image"
	_ "image/gif" // registers the GIF decoder for image.Decode
	"image/png"
	"io"
	"net/http"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/ungerr"
	_ "golang.org/x/image/webp" // registers the WebP decoder for image.Decode
)

const (
	imageRequestTimeout = 30 * time.Second
	// maxImageBytes bounds one hosted image. Catalog originals are far smaller,
	// so a larger body is refused rather than read into memory.
	maxImageBytes = 10 << 20
)

// HTTPImageFetcher downloads hosted images from the public image host. It
// never retries: a failed card is logged and counted by the batch job.
type HTTPImageFetcher struct {
	client *http.Client
}

// NewHTTPImageFetcher builds a fetcher with a per-request timeout.
func NewHTTPImageFetcher() *HTTPImageFetcher {
	return &HTTPImageFetcher{client: &http.Client{Timeout: imageRequestTimeout}}
}

// Fetch returns the body at url, which must answer 200 and fit maxImageBytes.
func (f *HTTPImageFetcher) Fetch(ctx context.Context, url string) (string, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", nil, ungerr.Wrapf(err, "building image request for %q", url)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", nil, ungerr.Wrapf(err, "downloading image %s", url)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Errorf("closing image response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", nil, ungerr.Unknownf("downloading image %s: HTTP %d", url, resp.StatusCode)
	}
	if resp.ContentLength > maxImageBytes {
		return "", nil, ungerr.Unknownf("image %s is %d bytes, over the %d byte limit", url, resp.ContentLength, maxImageBytes)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil {
		return "", nil, ungerr.Wrapf(err, "reading image %s", url)
	}
	if len(body) > maxImageBytes {
		return "", nil, ungerr.Unknownf("image %s is over the %d byte limit", url, maxImageBytes)
	}
	return providerImage(body)
}

// providerImage returns data as a PNG or JPEG, the image types the embedding
// provider accepts. GIF and WebP are decoded and re-encoded as PNG.
func providerImage(data []byte) (string, []byte, error) {
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
