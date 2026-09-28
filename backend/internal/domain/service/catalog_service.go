package service

import (
	"context"
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
// belonging to none, not a special case to work around" (see CONTEXT.md's
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

// CardFilter narrows CatalogService.SearchCards. Every field is optional;
// its zero value means "don't filter on this facet". Page is 1-indexed.
type CardFilter struct {
	Name           string
	ExpansionSetID uuid.UUID
	LocalID        string
	RarityID       uuid.UUID
	Category       string
	Tag            string
	Page           int
	Limit          int
}

// PaginationMeta is the pagination bookkeeping alongside a page of
// CatalogService.SearchCards results: the total number of Cards matching the
// filter (before pagination) plus the page/limit that produced this page, so
// a frontend can render page controls.
type PaginationMeta struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// CatalogService answers the unauthenticated, read-only catalog
// browse/search surface (ticket 05): listing Series/Expansion Sets to
// browse, and searching/filtering Cards by name, Expansion Set + card
// number, rarity, category, and tag. It never filters by ownership - that's
// Collection/Inventory territory (tickets 06/07), not implemented yet, so
// results always include Cards the caller doesn't own.
type CatalogService interface {
	// ListSeries returns every Series with its Expansion Sets nested
	// (ordered by Series name then Expansion Set release date), plus every
	// Expansion Set that belongs to no Series at all - a series-less
	// Expansion Set is a legitimate domain state, not a special case, and
	// this is its only way to be reachable through the catalog browse
	// surface.
	ListSeries(ctx context.Context) (SeriesBrowseResult, error)
	// ListRarities returns every Rarity across all Games, ordered by name.
	ListRarities(ctx context.Context) ([]RaritySummary, error)
	// ListCategories returns the distinct Card categories actually in use,
	// ordered alphabetically.
	ListCategories(ctx context.Context) ([]string, error)
	// ListTags returns the distinct Card tags actually in use, ordered
	// alphabetically.
	ListTags(ctx context.Context) ([]string, error)
	// SearchCards returns the page of Cards matching filter, plus that
	// page's pagination metadata.
	SearchCards(ctx context.Context, filter CardFilter) ([]CardSummary, PaginationMeta, error)
}
