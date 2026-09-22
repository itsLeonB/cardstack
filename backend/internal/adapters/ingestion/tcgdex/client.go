package tcgdex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// baseURL is TCGDex's public REST API. No auth/API key required.
const baseURL = "https://api.tcgdex.net/v2"

// client fetches TCGDex catalog data over plain HTTP GET + JSON — a
// handful of GET calls against a plain JSON REST API doesn't warrant a
// third-party client library.
type client struct {
	baseURL    string
	httpClient *http.Client
}

func newClient() *client {
	return &client{baseURL: baseURL, httpClient: http.DefaultClient}
}

// getSeries fetches a series (e.g. "SV") and the sets released under it for
// the given locale.
func (c *client) getSeries(ctx context.Context, locale, seriesID string) (seriesResponse, error) {
	var out seriesResponse
	err := c.get(ctx, fmt.Sprintf("/%s/series/%s", locale, seriesID), &out)
	return out, err
}

// getSet fetches one set's brief card list for the given locale.
func (c *client) getSet(ctx context.Context, locale, setID string) (setResponse, error) {
	var out setResponse
	err := c.get(ctx, fmt.Sprintf("/%s/sets/%s", locale, setID), &out)
	return out, err
}

// getCard fetches full card detail for one locale. A 404 means this locale
// legitimately has no data for the card (e.g. `en` has no SV1V-008) — not
// an error. The second return value reports whether the card was found.
func (c *client) getCard(ctx context.Context, locale, cardID string) (cardResponse, bool, error) {
	path := fmt.Sprintf("/%s/cards/%s", locale, cardID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return cardResponse{}, false, fmt.Errorf("building request for %s: %w", path, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return cardResponse{}, false, fmt.Errorf("requesting %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return cardResponse{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return cardResponse{}, false, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, path)
	}

	var out cardResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return cardResponse{}, false, fmt.Errorf("decoding %s: %w", path, err)
	}
	return out, true, nil
}

func (c *client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("building request for %s: %w", path, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("requesting %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d for %s", resp.StatusCode, path)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding %s: %w", path, err)
	}
	return nil
}
