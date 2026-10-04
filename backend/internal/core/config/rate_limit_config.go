package config

import "errors"

// RateLimit sets the per-user limits for signed-in callers: a token bucket per
// user per tier, refilling PerMinute tokens a minute up to Burst. Name search
// (GET /catalog/cards) and facets have their own tighter tiers and do not
// spend the general one. Guests are limited only per IP.
type RateLimit struct {
	UserPerMinute   int `split_words:"true" default:"300"`
	UserBurst       int `split_words:"true" default:"100"`
	SearchPerMinute int `split_words:"true" default:"60"`
	SearchBurst     int `split_words:"true" default:"30"`
	FacetsPerMinute int `split_words:"true" default:"30"`
	FacetsBurst     int `split_words:"true" default:"15"`
}

func (RateLimit) Prefix() string { return "RATE_LIMIT" }

// ValidateRateLimit refuses a zero or negative limit, which would answer every
// signed-in request with 429.
func (r RateLimit) ValidateRateLimit() error {
	for _, v := range []int{r.UserPerMinute, r.UserBurst, r.SearchPerMinute, r.SearchBurst, r.FacetsPerMinute, r.FacetsBurst} {
		if v <= 0 {
			return errors.New("every RATE_LIMIT_* setting must be positive")
		}
	}
	return nil
}
