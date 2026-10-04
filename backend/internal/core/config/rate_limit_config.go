package config

import "errors"

// Tier is one token bucket per user: PerMinute tokens come back a minute, up
// to Burst.
type Tier struct {
	PerMinute int `split_words:"true"`
	Burst     int
}

// RateLimit sets the per-user limits for signed-in callers. Name search and
// facets have their own tighter tiers and do not spend the general User one.
// Guests are limited only per IP.
type RateLimit struct {
	User   Tier
	Search Tier
	Facets Tier
}

func (RateLimit) Prefix() string { return "RATE_LIMIT" }

// DefaultRateLimit is what Load starts from, because envconfig's default tag
// cannot give the three tiers of one nested type different values.
func DefaultRateLimit() RateLimit {
	return RateLimit{
		User:   Tier{PerMinute: 300, Burst: 100},
		Search: Tier{PerMinute: 60, Burst: 30},
		Facets: Tier{PerMinute: 30, Burst: 15},
	}
}

// ValidateRateLimit refuses a zero or negative limit, which would answer every
// signed-in request with 429.
func (r RateLimit) ValidateRateLimit() error {
	for _, tier := range []Tier{r.User, r.Search, r.Facets} {
		if tier.PerMinute <= 0 || tier.Burst <= 0 {
			return errors.New("every RATE_LIMIT_* setting must be positive")
		}
	}
	return nil
}
