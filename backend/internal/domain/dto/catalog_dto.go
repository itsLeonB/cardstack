package dto

import (
	"time"

	"github.com/google/uuid"
)

// ExpansionSetSummary is a browsable Expansion Set: enough to list and pick
// from without pulling every Card it contains.
type ExpansionSetSummary struct {
	ID          uuid.UUID  `json:"id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	ReleaseDate *time.Time `json:"releaseDate,omitempty"`
	ImageURL    string     `json:"imageUrl"`
}

// SeriesSummary is a browsable Series with its Expansion Sets nested, so a
// catalog browse page can render the whole Series -> Expansion Set tree
// from one call (see ticket 05).
type SeriesSummary struct {
	ID            uuid.UUID             `json:"id"`
	Code          string                `json:"code"`
	Name          string                `json:"name"`
	ExpansionSets []ExpansionSetSummary `json:"expansionSets"`
}

// SeriesBrowseResult is GET /catalog/series's response. Series is optional
// on Expansion Set - "a Game with no Series data simply has Expansion Sets
// belonging to none, not a special case to work around" (see GLOSSARY.md's
// Series entry) - so UngroupedExpansionSets surfaces every Expansion Set
// with no Series directly, rather than nesting it under a fake Series
// record just to fit the same shape as the rest.
type SeriesBrowseResult struct {
	Series                 []SeriesSummary       `json:"series"`
	UngroupedExpansionSets []ExpansionSetSummary `json:"ungroupedExpansionSets"`
}

// RaritySummary is a Rarity lookup row, exposed so a search/filter UI can
// list valid rarity values instead of decoding raw codes itself (see
// docs/adr/0009).
type RaritySummary struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

// CardSummary is one catalog search/browse result. Rarity and Expansion Set
// are resolved to readable values rather than left as bare foreign keys,
// since the frontend renders these directly (see ticket 05).
type CardSummary struct {
	ID           uuid.UUID           `json:"id"`
	ExpansionSet ExpansionSetSummary `json:"expansionSet"`
	LocalID      string              `json:"localId"`
	Name         string              `json:"name"`
	Category     string              `json:"category"`
	Tags         []string            `json:"tags"`
	Rarity       RaritySummary       `json:"rarity"`
	Illustrator  string              `json:"illustrator"`
	ImageURL     string              `json:"imageUrl"`
}

// CardFilter narrows CatalogService.SearchCards and ListFacets. Every field
// is optional; its empty value means "don't filter on this facet". Values
// within one multi-value field combine with OR, fields combine with AND.
// Page is 1-indexed (ignored by ListFacets).
type CardFilter struct {
	Name            string
	ExpansionSetIDs []uuid.UUID
	LocalID         string
	RarityIDs       []uuid.UUID
	Categories      []string
	Tags            []string
	// CardIDs restricts to these Cards; only the Collection entries list sets it.
	CardIDs []uuid.UUID
	Page    int
	Limit   int
	// Guest marks a caller with no account. The catalog service gives a Guest
	// one page of the standard size and refuses the rest (see
	// CatalogService.SearchCards). The zero value is a Guest, so a caller that
	// forgets to set it gets the locked preview, never the full catalog.
	Guest bool
}

// ExpansionSetFacetOption is one Expansion Set choice in GET
// /catalog/facets. Available is false when the set is selected but no Card
// matching the other filters belongs to it.
type ExpansionSetFacetOption struct {
	ExpansionSetSummary
	SeriesID  *uuid.UUID `json:"seriesId,omitempty"`
	Available bool       `json:"available"`
}

// RarityFacetOption is one Rarity choice in GET /catalog/facets.
type RarityFacetOption struct {
	RaritySummary
	Available bool `json:"available"`
}

// StringFacetOption is one Category or Tag choice in GET /catalog/facets.
type StringFacetOption struct {
	Value     string `json:"value"`
	Available bool   `json:"available"`
}

// CatalogFacets is GET /catalog/facets's response: per filter, the options
// present on Cards matching every other active filter, plus the filter's
// own selected values (Available=false when no longer reachable).
type CatalogFacets struct {
	ExpansionSets []ExpansionSetFacetOption `json:"expansionSets"`
	Rarities      []RarityFacetOption       `json:"rarities"`
	Categories    []StringFacetOption       `json:"categories"`
	Tags          []StringFacetOption       `json:"tags"`
}
