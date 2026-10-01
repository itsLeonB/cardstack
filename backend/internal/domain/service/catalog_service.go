package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ezutil/v2"
)

// defaultCardSearchLimit/maxCardSearchLimit bound CardFilter.Limit before it
// ever reaches a query: a caller that omits Limit gets a reasonable page
// size, and one that asks for an unreasonably large page is clamped rather
// than allowed to pull the whole catalog in one response.
const (
	defaultCardSearchLimit = 24
	maxCardSearchLimit     = 100
)

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
	ListSeries(ctx context.Context) (dto.SeriesBrowseResult, error)
	// ListRarities returns every Rarity across all Games, ordered by name.
	ListRarities(ctx context.Context) ([]dto.RaritySummary, error)
	// ListCategories returns the distinct Card categories actually in use,
	// ordered alphabetically.
	ListCategories(ctx context.Context) ([]string, error)
	// ListTags returns the distinct Card tags actually in use, ordered
	// alphabetically.
	ListTags(ctx context.Context) ([]string, error)
	// SearchCards returns the page of Cards matching filter, plus that
	// page's pagination metadata.
	SearchCards(ctx context.Context, filter dto.CardFilter) ([]dto.CardSummary, dto.PaginationMeta, error)
	// ListFacets returns each filter's available options given filter: the
	// values on Cards matching every other filter, plus the filter's own
	// selected values (flagged unavailable when unreachable).
	ListFacets(ctx context.Context, filter dto.CardFilter) (dto.CatalogFacets, error)
}

type catalogService struct {
	repo repository.CatalogRepository
}

// NewCatalogService builds a CatalogService backed by repo.
func NewCatalogService(repo repository.CatalogRepository) CatalogService {
	return &catalogService{repo: repo}
}

func (s *catalogService) ListSeries(ctx context.Context) (dto.SeriesBrowseResult, error) {
	series, err := s.repo.ListSeries(ctx)
	if err != nil {
		return dto.SeriesBrowseResult{}, err
	}

	seriesIDs := make([]uuid.UUID, len(series))
	for i, sr := range series {
		seriesIDs[i] = sr.ID
	}

	sets, err := s.repo.ListExpansionSets(ctx, seriesIDs)
	if err != nil {
		return dto.SeriesBrowseResult{}, err
	}

	ungrouped, err := s.repo.ListUngroupedExpansionSets(ctx)
	if err != nil {
		return dto.SeriesBrowseResult{}, err
	}

	setsBySeries := make(map[uuid.UUID][]dto.ExpansionSetSummary, len(series))
	for _, set := range sets {
		if set.SeriesID == nil {
			continue
		}
		setsBySeries[*set.SeriesID] = append(setsBySeries[*set.SeriesID], mapper.ToExpansionSetSummary(set))
	}

	summaries := make([]dto.SeriesSummary, len(series))
	for i, sr := range series {
		summaries[i] = dto.SeriesSummary{
			ID:            sr.ID,
			Code:          sr.Code,
			Name:          sr.Name,
			ExpansionSets: setsBySeries[sr.ID],
		}
	}

	ungroupedSummaries := make([]dto.ExpansionSetSummary, len(ungrouped))
	for i, set := range ungrouped {
		ungroupedSummaries[i] = mapper.ToExpansionSetSummary(set)
	}

	return dto.SeriesBrowseResult{
		Series:                 summaries,
		UngroupedExpansionSets: ungroupedSummaries,
	}, nil
}

func (s *catalogService) ListRarities(ctx context.Context) ([]dto.RaritySummary, error) {
	rarities, err := s.repo.ListRarities(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]dto.RaritySummary, len(rarities))
	for i, r := range rarities {
		summaries[i] = dto.RaritySummary{ID: r.ID, Code: r.Code, Name: r.Name}
	}

	return summaries, nil
}

func (s *catalogService) ListCategories(ctx context.Context) ([]string, error) {
	return s.repo.ListDistinctCategories(ctx)
}

func (s *catalogService) ListTags(ctx context.Context) ([]string, error) {
	return s.repo.ListDistinctTags(ctx)
}

func (s *catalogService) SearchCards(ctx context.Context, filter dto.CardFilter) ([]dto.CardSummary, dto.PaginationMeta, error) {
	repoFilter, page, limit := pagedRepoFilter(filter)

	results, total, err := s.repo.SearchCards(ctx, repoFilter)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	return ezutil.MapSlice(results, mapper.ToCardSummary), dto.PaginationMeta{Total: int(total), Page: page, Limit: limit}, nil
}

// pagedRepoFilter maps filter to the repository filter with its page's
// limit/offset applied, and returns the normalized page and limit.
func pagedRepoFilter(filter dto.CardFilter) (repository.CardFilter, int, int) {
	page, limit := normalizePagination(filter.Page, filter.Limit)

	repoFilter := mapper.ToRepoCardFilter(filter)
	repoFilter.Limit = limit
	repoFilter.Offset = (page - 1) * limit

	return repoFilter, page, limit
}

func (s *catalogService) ListFacets(ctx context.Context, filter dto.CardFilter) (dto.CatalogFacets, error) {
	facets, err := s.repo.ListCardFacets(ctx, mapper.ToRepoCardFilter(filter))
	if err != nil {
		return dto.CatalogFacets{}, err
	}

	return mapper.ToCatalogFacets(facets), nil
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
