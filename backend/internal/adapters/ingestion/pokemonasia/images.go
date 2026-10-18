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
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
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

// imageFetcher downloads one image address, so a row's source decides which host it may come from.
type imageFetcher func(ctx context.Context, rawURL string) ([]byte, string, error)

// Set logos and TCG Collector's card images come from its CDN, not the Asia
// host, so they share this allowed host and client (ticket 03, ADR-0016).
const (
	logoScheme = "https"
	logoHost   = "static.tcgcollector.com"
)

// expansionSetLogoKey is kept apart from expansionSetImageKey so the logo and the cover never share a key.
func expansionSetLogoKey(id uuid.UUID) string { return "expansion-set-logos/" + id.String() }

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
	return c.fetchFrom(ctx, rawURL, c.imageScheme, c.imageHost, c.imageClient)
}

// fetchLogo downloads a set logo, which may come only from the TCG Collector CDN.
// fetchTCGCollector downloads an image from TCG Collector's CDN, the one host
// its set logos and card images may come from.
func (c *client) fetchTCGCollector(ctx context.Context, rawURL string) ([]byte, string, error) {
	return c.fetchFrom(ctx, rawURL, logoScheme, logoHost, c.logoClient)
}

// cardImageFetcher returns how a card's image is downloaded, from the source
// that last wrote its row. It is nil for a row no source may re-host (an Admin
// edit keeps the image it was given), so the card is left alone.
func (c *client) cardImageFetcher(updatedBy string) imageFetcher {
	switch updatedBy {
	case entity.SourcePokemonAsia:
		return c.fetchImage
	case entity.SourceTCGCollector:
		return c.fetchTCGCollector
	}
	return nil
}

func (c *client) fetchFrom(ctx context.Context, rawURL, scheme, host string, hc *http.Client) ([]byte, string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, "", fmt.Errorf("parsing base address: %w", err)
	}
	ref, err := url.Parse(rawURL)
	if err != nil || rawURL == "" {
		return nil, "", fmt.Errorf("invalid image address %q", rawURL)
	}
	target := base.ResolveReference(ref)
	if err := checkImageURL(target, scheme, host); err != nil {
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
	resp, err := hc.Do(req)
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
func (in *Ingester) hostImage(ctx context.Context, fetch imageFetcher, key, sourceURL string) error {
	body, contentType, err := fetch(ctx, sourceURL)
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
	fetch := in.client.cardImageFetcher(card.UpdatedBy)
	if in.store == nil || fetch == nil || card.ImageKey != "" || card.SourceImageURL == "" {
		return
	}
	key := cardImageKey(card.ID)
	in.hostAndSave(ctx, IngestFailure{ExpansionCode: expansionCode, CardID: siteCardID}, "card image", key, card.SourceImageURL, fetch, func() error {
		card.ImageKey = key
		_, err := in.cards.Update(ctx, card)
		return err
	})
}

// hostSetCover is hostCardImage for an Expansion Set's cover.
func (in *Ingester) hostSetCover(ctx context.Context, set entity.ExpansionSet, expansionCode string) {
	if in.store == nil || !ownedByAsia(set.UpdatedBy) || set.ImageKey != "" || set.SourceImageURL == "" {
		return
	}
	key := expansionSetImageKey(set.ID)
	in.hostAndSave(ctx, IngestFailure{ExpansionCode: expansionCode}, "expansion set cover", key, set.SourceImageURL, in.client.fetchImage, func() error {
		set.ImageKey = key
		_, err := in.sets.Update(ctx, set)
		return err
	})
}

// hostSetLogo hosts a set's logo under its own key, once. The save reloads the row, because
// the caller's copy predates the cover pass in the same run and a whole-row Update would
// write its stale image key back.
func (in *Ingester) hostSetLogo(ctx context.Context, set entity.ExpansionSet, expansionCode string) {
	if in.store == nil || set.LogoKey != "" || set.SourceLogoURL == "" {
		return
	}
	key := expansionSetLogoKey(set.ID)
	in.hostAndSave(ctx, IngestFailure{ExpansionCode: expansionCode}, "expansion set logo", key, set.SourceLogoURL, in.client.fetchTCGCollector, func() error {
		row, err := in.sets.FindFirst(ctx, crud.Specification[entity.ExpansionSet]{Model: entity.ExpansionSet{BaseEntity: crud.BaseEntity{ID: set.ID}}})
		if err != nil {
			return err
		}
		if row.IsZero() {
			return ungerr.Unknownf("expansion set %s was removed before its logo key was saved", expansionCode)
		}
		row.LogoKey = key
		_, err = in.sets.Update(ctx, row)
		return err
	})
}

// hostAndSave copies sourceURL to key, then runs save to persist the key.
// base carries the failure's identifiers; a failure at either step is
// recorded with Stage "hosting <what>" or "saving <what> key".
func (in *Ingester) hostAndSave(ctx context.Context, base IngestFailure, what, key, sourceURL string, fetch imageFetcher, save func() error) {
	if err := in.hostImage(ctx, fetch, key, sourceURL); err != nil {
		base.Stage, base.Err = "hosting "+what, err
		in.recordImageFailure(ctx, base)
		return
	}
	if err := save(); err != nil {
		base.Stage, base.Err = "saving "+what+" key", err
		in.recordImageFailure(ctx, base)
	}
}

func (in *Ingester) failureCount() int {
	in.failuresMu.Lock()
	defer in.failuresMu.Unlock()
	return len(in.failures)
}

// recordImageFailure records f unless ctx was canceled: a Ctrl+C is the
// caller stopping the run, not an image that needs a retry.
func (in *Ingester) recordImageFailure(ctx context.Context, f IngestFailure) {
	if ctx.Err() != nil {
		return
	}
	in.recordFailure(f)
}
