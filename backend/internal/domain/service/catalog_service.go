package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
)

// defaultCardSearchLimit/maxCardSearchLimit bound CardFilter.Limit before it
// ever reaches a query: a caller that omits Limit gets a reasonable page
// size, and one that asks for an unreasonably large page is clamped rather
// than allowed to pull the whole catalog in one response.
const (
	defaultCardSearchLimit = 24
	maxCardSearchLimit     = 100
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

type catalogService struct {
	repo repository.CatalogRepository
}

// NewCatalogService builds a CatalogService backed by repo.
func NewCatalogService(repo repository.CatalogRepository) CatalogService {
	return &catalogService{repo: repo}
}

func (s *catalogService) ListSeries(ctx context.Context) (SeriesBrowseResult, error) {
	series, err := s.repo.ListSeries(ctx)
	if err != nil {
		return SeriesBrowseResult{}, err
	}

	seriesIDs := make([]uuid.UUID, len(series))
	for i, sr := range series {
		seriesIDs[i] = sr.ID
	}

	sets, err := s.repo.ListExpansionSets(ctx, seriesIDs)
	if err != nil {
		return SeriesBrowseResult{}, err
	}

	ungrouped, err := s.repo.ListUngroupedExpansionSets(ctx)
	if err != nil {
		return SeriesBrowseResult{}, err
	}

	setsBySeries := make(map[uuid.UUID][]ExpansionSetSummary, len(series))
	for _, set := range sets {
		if set.SeriesID == nil {
			continue
		}
		setsBySeries[*set.SeriesID] = append(setsBySeries[*set.SeriesID], toExpansionSetSummary(set))
	}

	summaries := make([]SeriesSummary, len(series))
	for i, sr := range series {
		summaries[i] = SeriesSummary{
			ID:            sr.ID,
			Code:          sr.Code,
			Name:          sr.Name,
			ExpansionSets: setsBySeries[sr.ID],
		}
	}

	ungroupedSummaries := make([]ExpansionSetSummary, len(ungrouped))
	for i, set := range ungrouped {
		ungroupedSummaries[i] = toExpansionSetSummary(set)
	}

	return SeriesBrowseResult{
		Series:                 summaries,
		UngroupedExpansionSets: ungroupedSummaries,
	}, nil
}

func (s *catalogService) ListRarities(ctx context.Context) ([]RaritySummary, error) {
	rarities, err := s.repo.ListRarities(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]RaritySummary, len(rarities))
	for i, r := range rarities {
		summaries[i] = RaritySummary{ID: r.ID, Code: r.Code, Name: r.Name}
	}

	return summaries, nil
}

func (s *catalogService) ListCategories(ctx context.Context) ([]string, error) {
	return s.repo.ListDistinctCategories(ctx)
}

func (s *catalogService) ListTags(ctx context.Context) ([]string, error) {
	return s.repo.ListDistinctTags(ctx)
}

func (s *catalogService) SearchCards(ctx context.Context, filter CardFilter) ([]CardSummary, PaginationMeta, error) {
	page, limit := normalizePagination(filter.Page, filter.Limit)

	results, total, err := s.repo.SearchCards(ctx, repository.CardFilter{
		Name:           filter.Name,
		ExpansionSetID: filter.ExpansionSetID,
		LocalID:        filter.LocalID,
		RarityID:       filter.RarityID,
		Category:       filter.Category,
		Tag:            filter.Tag,
		Limit:          limit,
		Offset:         (page - 1) * limit,
	})
	if err != nil {
		return nil, PaginationMeta{}, err
	}

	cards := make([]CardSummary, len(results))
	for i, r := range results {
		cards[i] = toCardSummary(r)
	}

	return cards, PaginationMeta{
		Total: int(total),
		Page:  page,
		Limit: limit,
	}, nil
}

// normalizePagination fills in CardFilter's page/limit defaults and clamps
// limit to maxCardSearchLimit, so a caller that omits them (or passes an
// out-of-range value) still gets a bounded, well-formed query rather than
// an error or an unbounded result set.
func normalizePagination(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultCardSearchLimit
	}
	if limit > maxCardSearchLimit {
		limit = maxCardSearchLimit
	}

	return page, limit
}

// toExpansionSetSummary converts an entity.ExpansionSet into the catalog
// service's browsable summary DTO. Moved in from the now-deleted
// domain/mapper package: that package existed solely for catalogService's
// use, and importing it from here (once catalogService itself lived in this
// package) would have created an import cycle, since it in turn imported
// this package for the DTO types.
func toExpansionSetSummary(set entity.ExpansionSet) ExpansionSetSummary {
	return ExpansionSetSummary{
		ID:          set.ID,
		Code:        set.Code,
		Name:        set.Name,
		ReleaseDate: set.ReleaseDate,
	}
}

// toCardSummary converts a repository.CardResult row (a Card already joined
// with its Rarity and Expansion Set) into the catalog service's search
// result DTO. See toExpansionSetSummary's doc comment for why this moved
// here rather than staying in domain/mapper.
func toCardSummary(r repository.CardResult) CardSummary {
	return CardSummary{
		ID: r.ID,
		ExpansionSet: ExpansionSetSummary{
			ID:          r.ExpansionSetID,
			Code:        r.ExpansionSetCode,
			Name:        r.ExpansionSetName,
			ReleaseDate: r.ExpansionSetReleaseDate,
		},
		LocalID:  r.LocalID,
		Name:     r.Name,
		Category: r.Category,
		Tags:     []string(r.Tags),
		Rarity: RaritySummary{
			ID:   r.RarityID,
			Code: r.RarityCode,
			Name: r.RarityName,
		},
		Illustrator: r.Illustrator,
		ImageURL:    r.ImageURL,
	}
}
