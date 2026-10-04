package pokemonasia

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
)

// maxImageBytes caps a downloaded image. The address comes from scraped
// markup, so the cap bounds what a hostile or broken response can make us
// buffer and upload; real card scans are far smaller.
const maxImageBytes = 10 << 20

// maxImageRedirects bounds how many same-host redirects an image download follows.
const maxImageRedirects = 3

// allowedImageTypes are the sniffed content types we host. SVG is excluded on
// purpose: it can carry script, and http.DetectContentType never reports it.
var allowedImageTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
}

// cardImageKey and expansionSetImageKey are the object-store keys for a row's
// hosted original. They derive from the row id alone, so a re-run finds the
// same key, and cards and Expansion Sets never share a namespace (ADR-0016).
func cardImageKey(id uuid.UUID) string { return "cards/" + id.String() }

func expansionSetImageKey(id uuid.UUID) string { return "expansion-sets/" + id.String() }

// newImageClient builds the HTTP client for image downloads. It is separate
// from the page client so it can refuse any redirect that leaves the source
// host, which would otherwise sidestep the allow-list.
func newImageClient(scheme, host string) *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > maxImageRedirects {
				return fmt.Errorf("stopped after %d redirects", maxImageRedirects)
			}
			return checkImageURL(req.URL, scheme, host)
		},
	}
}

// checkImageURL accepts only absolute addresses on the one known source host,
// over scheme (https in production).
func checkImageURL(u *url.URL, scheme, host string) error {
	if u.Scheme != scheme {
		return fmt.Errorf("image address must use %s, got scheme %q", scheme, u.Scheme)
	}
	if u.User != nil || !strings.EqualFold(u.Host, host) {
		return fmt.Errorf("image host %q is not the source host", u.Host)
	}
	return nil
}

// fetchImage downloads one image from the source host. It returns the bytes
// and the content type sniffed from them: the response's own Content-Type is
// not trusted. Anything off the source host, over maxImageBytes, or not a
// recognised raster image is an error, so nothing bad reaches the bucket.
func (c *client) fetchImage(ctx context.Context, rawURL string) ([]byte, string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, "", fmt.Errorf("parsing base address: %w", err)
	}
	ref, err := url.Parse(rawURL)
	if err != nil || rawURL == "" {
		return nil, "", fmt.Errorf("invalid image address %q", rawURL)
	}
	target := base.ResolveReference(ref)
	if err := checkImageURL(target, c.imageScheme, c.imageHost); err != nil {
		return nil, "", err
	}

	if err := c.limiter.Wait(ctx); err != nil {
		return nil, "", fmt.Errorf("waiting for rate limiter: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("building image request: %w", err)
	}
	resp, err := c.imageClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("requesting image: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Errorf("closing image response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status %d for image", resp.StatusCode)
	}
	if resp.ContentLength > maxImageBytes {
		return nil, "", fmt.Errorf("image is %d bytes, over the %d byte limit", resp.ContentLength, maxImageBytes)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("reading image body: %w", err)
	}
	if len(body) > maxImageBytes {
		return nil, "", fmt.Errorf("image is over the %d byte limit", maxImageBytes)
	}

	contentType := http.DetectContentType(body)
	if !allowedImageTypes[contentType] {
		return nil, "", fmt.Errorf("content is %q, not a supported image", contentType)
	}
	return body, contentType, nil
}

// hostImage copies the image at sourceURL into the object store under key.
func (in *Ingester) hostImage(ctx context.Context, key, sourceURL string) error {
	body, contentType, err := in.client.fetchImage(ctx, sourceURL)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", sourceURL, err)
	}
	if err := in.store.Put(ctx, key, contentType, body); err != nil {
		return fmt.Errorf("uploading %s: %w", key, err)
	}
	in.imagesHosted.Add(1)
	return nil
}

// hostCardImage hosts card's image unless it is already hosted, has no source
// address, or hosting is off. A failure is recorded and swallowed: the key
// stays empty so the API serves no image and a later run retries.
func (in *Ingester) hostCardImage(ctx context.Context, card entity.Card, expansionCode, siteCardID string) {
	if in.store == nil || card.ImageKey != "" || card.SourceImageURL == "" {
		return
	}
	key := cardImageKey(card.ID)
	in.hostAndSave(ctx, IngestFailure{ExpansionCode: expansionCode, CardID: siteCardID}, "card image", key, card.SourceImageURL, func() error {
		card.ImageKey = key
		_, err := in.cards.Update(ctx, card)
		return err
	})
}

// hostSetCover is hostCardImage for an Expansion Set's cover.
func (in *Ingester) hostSetCover(ctx context.Context, set entity.ExpansionSet, expansionCode string) {
	if in.store == nil || set.ImageKey != "" || set.SourceImageURL == "" {
		return
	}
	key := expansionSetImageKey(set.ID)
	in.hostAndSave(ctx, IngestFailure{ExpansionCode: expansionCode}, "expansion set cover", key, set.SourceImageURL, func() error {
		set.ImageKey = key
		_, err := in.sets.Update(ctx, set)
		return err
	})
}

// hostAndSave copies sourceURL to key, then runs save to persist the key.
// base carries the failure's identifiers; a failure at either step is
// recorded with Stage "hosting <what>" or "saving <what> key".
func (in *Ingester) hostAndSave(ctx context.Context, base IngestFailure, what, key, sourceURL string, save func() error) {
	if err := in.hostImage(ctx, key, sourceURL); err != nil {
		base.Stage, base.Err = "hosting "+what, err
		in.recordImageFailure(ctx, base)
		return
	}
	if err := save(); err != nil {
		base.Stage, base.Err = "saving "+what+" key", err
		in.recordImageFailure(ctx, base)
	}
}

// recordImageFailure records f unless ctx was canceled: a Ctrl+C is the
// caller stopping the run, not an image that needs a retry.
func (in *Ingester) recordImageFailure(ctx context.Context, f IngestFailure) {
	if ctx.Err() != nil {
		return
	}
	in.recordFailure(f)
}
