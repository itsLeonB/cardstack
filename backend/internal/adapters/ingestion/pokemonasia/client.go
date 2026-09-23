package pokemonasia

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"golang.org/x/time/rate"
)

// baseURL is the Indonesian ("id") locale of the live site. No auth/API key
// required; the whole flow is plain server-rendered HTML over GET.
const baseURL = "https://asia.pokemon-card.com/id"

// requestTimeout bounds each outgoing request so a hung response can't
// block ingestion forever (the CLI passes context.Background()).
const requestTimeout = 15 * time.Second

// requestInterval paces every outgoing request through a single shared
// rate.Limiter, regardless of how many workers are fetching concurrently
// (see ingest.go's maxConcurrency) — this is a real consumer website, not a
// service built for scraping. See the plan's "bounded concurrency + a small
// per-request delay" decision.
const requestInterval = 500 * time.Millisecond

// client fetches Pokémon Asia catalog pages over plain HTTP GET + HTML
// parsing (goquery) — the site has no JSON API.
type client struct {
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
}

func newClient() *client {
	return &client{
		baseURL:    baseURL,
		httpClient: http.DefaultClient,
		limiter:    rate.NewLimiter(rate.Every(requestInterval), 1),
	}
}

// expansionListPage fetches one page of the Series/Expansion Set/release-
// date enumeration (ul.expansionList).
func (c *client) expansionListPage(ctx context.Context, pageNo int) (*goquery.Document, []byte, error) {
	return c.get(ctx, fmt.Sprintf("/card-search/?pageNo=%d", pageNo))
}

// resultsPage fetches one page of card-thumbnail results for one Expansion
// Set/regulation bucket (regulation is 1, 2, or 3).
func (c *client) resultsPage(ctx context.Context, expansionCode string, regulation, pageNo int) (*goquery.Document, []byte, error) {
	path := fmt.Sprintf(
		"/card-search/list/?expansionCodes=%s&regulation=%d&cardType=all&pageNo=%d",
		url.QueryEscape(expansionCode), regulation, pageNo,
	)
	return c.get(ctx, path)
}

// cardDetail fetches one card's full detail page by its numeric detail-page
// id (not the card's LocalID/collector number).
func (c *client) cardDetail(ctx context.Context, id string) (*goquery.Document, []byte, error) {
	return c.get(ctx, fmt.Sprintf("/card-search/detail/%s/", id))
}

func (c *client) get(ctx context.Context, path string) (*goquery.Document, []byte, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, nil, fmt.Errorf("waiting for rate limiter for %s: %w", path, err)
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request for %s: %w", path, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("requesting %s: %w", path, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Errorf("closing response body for %s: %v", path, err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, path)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading body for %s: %w", path, err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing html for %s: %w", path, err)
	}

	return doc, raw, nil
}
