package dto

import "time"

// EmbedCatalogRequest scopes one embedding run to the Expansion Set with Set
// as its code, or to every card when Set is empty.
type EmbedCatalogRequest struct {
	Set string
}

// EmbedCatalogSummary reports one run. Total, NoImage and AlreadyEmbedded
// describe the scope before the run; Embedded and Failed count this run's
// work. Stopped means the daily quota ran out, so a rerun resumes.
type EmbedCatalogSummary struct {
	Total           int64
	NoImage         int64
	AlreadyEmbedded int64
	Embedded        int
	Failed          int
	Stopped         bool
	Elapsed         time.Duration
}
