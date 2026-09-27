package pokemonasia

import (
	"bytes"
	"context"
	"fmt"
	"io"
	mrand "math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
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

// maxRetries bounds how many extra attempts get makes for a 429/5xx
// response before giving up. retryBaseWait backs off exponentially when the
// server gives no Retry-After header.
const (
	maxRetries    = 3
	retryBaseWait = 1 * time.Second
)

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
// Set's complete card list. regulation=all is the site's own "Semua" (All)
// value — confirmed live to reliably return a set's true, complete card
// count, unlike the regulation=1/2/3 partition it replaces (see ADR-0010).
func (c *client) resultsPage(ctx context.Context, expansionCode string, pageNo int) (*goquery.Document, []byte, error) {
	path := fmt.Sprintf(
		"/card-search/list/?expansionCodes=%s&regulation=all&cardType=all&pageNo=%d",
		url.QueryEscape(expansionCode), pageNo,
	)
	return c.get(ctx, path)
}

// rarityResultsPage fetches one page of card-thumbnail results for one
// Expansion Set filtered to a single rarity[] filter id (see mapper.go's
// parseRarityFilterOptions for how ids are discovered — they're undocumented
// and not assumed stable). Querying one id at a time and paginating returns
// exactly the card ids carrying that rarity code.
func (c *client) rarityResultsPage(ctx context.Context, expansionCode, rarityFilterID string, pageNo int) (*goquery.Document, []byte, error) {
	q := url.Values{}
	q.Set("expansionCodes", expansionCode)
	q.Set("rarity[]", rarityFilterID)
	q.Set("cardType", "all")
	q.Set("pageNo", strconv.Itoa(pageNo))
	return c.get(ctx, "/card-search/list/?"+q.Encode())
}

// cardDetail fetches one card's full detail page by its numeric detail-page
// id (not the card's LocalID/collector number).
func (c *client) cardDetail(ctx context.Context, id string) (*goquery.Document, []byte, error) {
	return c.get(ctx, fmt.Sprintf("/card-search/detail/%s/", id))
}

func (c *client) get(ctx context.Context, path string) (*goquery.Document, []byte, error) {
	var lastErr error
	for attempt := 0; ; attempt++ {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, nil, fmt.Errorf("waiting for rate limiter for %s: %w", path, err)
		}

		doc, raw, status, retryAfter, err := c.doGet(ctx, path)
		if err == nil {
			return doc, raw, nil
		}
		lastErr = err

		// A caller-driven cancellation (e.g. Ctrl+C on the CLI) should stop
		// immediately, never be treated as a retryable condition.
		if ctx.Err() != nil {
			return nil, nil, lastErr
		}

		// status == 0 means the request never got an HTTP response at all
		// (timeout, connection reset, DNS hiccup, ...) - these are exactly
		// as transient as a 429/5xx and must be retried the same way.
		retryable := status == 0 || status == http.StatusTooManyRequests || status >= 500
		if !retryable || attempt >= maxRetries {
			return nil, nil, lastErr
		}

		wait := retryAfter
		if wait <= 0 {
			wait = backoffWithJitter(attempt)
		}
		logger.Warnf(
			"retrying %s after status %d (attempt %d/%d), waiting %s: %v",
			path, status, attempt+1, maxRetries, wait, err,
		)

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// backoffWithJitter returns an exponential backoff duration for the given
// zero-based attempt number, using "equal jitter" (half fixed, half random)
// so concurrent workers hitting the same failure don't all retry in
// lockstep against the site.
func backoffWithJitter(attempt int) time.Duration {
	base := retryBaseWait * time.Duration(1<<attempt)
	half := base / 2
	return half + time.Duration(mrand.Int64N(int64(half)+1))
}

// doGet performs a single request attempt. status is the HTTP status code
// (0 if the request never got a response), and retryAfter is the server's
// requested backoff for a 429/5xx response, if any.
func (c *client) doGet(ctx context.Context, path string) (*goquery.Document, []byte, int, time.Duration, error) {
	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("building request for %s: %w", path, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("requesting %s: %w", path, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Errorf("closing response body for %s: %v", path, err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return nil, nil, resp.StatusCode, retryAfter, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, path)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, resp.StatusCode, 0, fmt.Errorf("reading body for %s: %w", path, err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
	if err != nil {
		return nil, nil, resp.StatusCode, 0, fmt.Errorf("parsing html for %s: %w", path, err)
	}

	return doc, raw, resp.StatusCode, 0, nil
}

// parseRetryAfter parses an HTTP Retry-After header, which is either a
// delta in seconds or an HTTP-date. Returns 0 (let the caller fall back to
// its own backoff) if the header is absent or unparseable.
func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}
