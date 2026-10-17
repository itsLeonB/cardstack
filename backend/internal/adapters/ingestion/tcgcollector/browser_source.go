package tcgcollector

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/ungerr"
)

const (
	// minGap spaces requests a few seconds apart, as a person browsing would.
	minGap = 3 * time.Second
	// challengePoll is how often the open tab is read while the operator solves the challenge.
	challengePoll = 5 * time.Second
)

// challengeMarkers mark Cloudflare's interstitial only, never the scripts on
// every normal page (ADR-0021).
var challengeMarkers = []string{"_cf_chl_opt", "<title>just a moment"}

// browserSource fetches pages through one visible Chrome. Its profile
// directory keeps the operator's unlock for later runs. It never runs
// headless and never tries to pass a challenge (ADR-0021): it waits while the
// operator solves it in the window.
type browserSource struct {
	tab context.Context
	gap time.Duration

	mu   sync.Mutex // one request at a time
	last time.Time
}

// NewBrowserSource returns a PageSource that drives a visible Chrome with
// profileDir as its persistent profile. Chrome starts on the first Fetch. The
// returned func closes Chrome.
func NewBrowserSource(profileDir string) (PageSource, func()) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", false), chromedp.UserDataDir(profileDir))
	alloc, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	tab, cancelTab := chromedp.NewContext(alloc)

	return &browserSource{tab: tab, gap: minGap}, func() {
		cancelTab()
		cancelAlloc()
	}
}

// Fetch returns the HTML of pageURL once no challenge is showing. It waits for
// the previous request to finish and for the gap to pass, and it never
// requests a ?page= URL.
func (s *browserSource) Fetch(ctx context.Context, pageURL string) ([]byte, error) {
	if err := checkPageURL(pageURL); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.pace(ctx); err != nil {
		return nil, err
	}

	if _, err := chromedp.Run(s.tab, chromedp.Navigate(pageURL)); err != nil {
		return nil, ungerr.Wrapf(err, "loading %s", pageURL)
	}

	page, err := s.outerHTML()
	if err != nil {
		return nil, ungerr.Wrapf(err, "reading %s", pageURL)
	}

	if isChallenge(page) {
		logger.Warn("TCG Collector shows a challenge: solve it in the Chrome window, the command continues once it clears")
		for isChallenge(page) {
			if err := sleepCtx(ctx, challengePoll); err != nil {
				return nil, err
			}
			if page, err = s.outerHTML(); err != nil {
				return nil, ungerr.Wrapf(err, "reading %s while the challenge shows", pageURL)
			}
		}
		logger.Info("challenge cleared, continuing")
	}

	s.last = time.Now()
	return []byte(page), nil
}

// pace waits until the gap has passed since the last successful request.
func (s *browserSource) pace(ctx context.Context) error {
	return sleepCtx(ctx, time.Until(s.last.Add(s.gap)))
}

func (s *browserSource) outerHTML() (string, error) {
	return chromedp.Run(s.tab, chromedp.OuterHTML("html"))
}

// checkPageURL refuses ?page= URLs: robots.txt bars them and no page we need
// uses them.
func checkPageURL(pageURL string) error {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return ungerr.Wrapf(err, "parsing page URL %s", pageURL)
	}

	if parsed.Query().Has("page") {
		return ungerr.Unknownf("refusing %s: ?page= URLs are barred by robots.txt", pageURL)
	}

	return nil
}

func isChallenge(page string) bool {
	lower := strings.ToLower(page)
	for _, marker := range challengeMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ungerr.Wrap(ctx.Err(), "waiting between requests")
	case <-timer.C:
		return nil
	}
}
