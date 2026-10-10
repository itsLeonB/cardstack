package embedding

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/ungerr"
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
func (f *HTTPImageFetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, ungerr.Wrapf(err, "building image request for %q", url)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, ungerr.Wrapf(err, "downloading image %s", url)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Errorf("closing image response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, ungerr.Unknownf("downloading image %s: HTTP %d", url, resp.StatusCode)
	}
	if resp.ContentLength > maxImageBytes {
		return nil, ungerr.Unknownf("image %s is %d bytes, over the %d byte limit", url, resp.ContentLength, maxImageBytes)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil {
		return nil, ungerr.Wrapf(err, "reading image %s", url)
	}
	if len(body) > maxImageBytes {
		return nil, ungerr.Unknownf("image %s is over the %d byte limit", url, maxImageBytes)
	}
	return body, nil
}
