package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/apperr"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ezutil/v2"
	"github.com/itsLeonB/ungerr"
)

// defaultCardSearchLimit/maxCardSearchLimit bound CardFilter.Limit before it
// ever reaches a query: a caller that omits Limit gets a reasonable page
// size, and one that asks for an unreasonably large page is clamped rather
// than allowed to pull the whole catalog in one response.
const (
	defaultCardSearchLimit = 24
	maxCardSearchLimit     = 100
)

// loginRequiredMsg is the client-safe text of the 401 a Guest gets for what the
// catalog preview locks; the stable signal is the apperr.CodeLoginRequired code.
const loginRequiredMsg = "sign in to use this part of the catalog"

// CatalogService answers the read-only catalog browse/search surface (ticket
// 05): listing Series/Expansion Sets to browse, and searching/filtering Cards by name, Expansion Set + card
// number, rarity, category, and tag. It never filters by ownership - that's
// Collection/Inventory territory (tickets 06/07), not implemented yet, so
// results always include Cards the caller doesn't own.
type CatalogService interface {
	// ListSeries returns every Series with its Expansion Sets nested
	// (Series and Expansion Sets both ordered most recently released first),
	// plus every Expansion Set that belongs to no Series at all - a
	// series-less Expansion Set is a legitimate domain state, not a special
	// case, and this is its only way to be reachable through the catalog
	// browse surface.
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
	// page's pagination metadata. A Guest (filter.Guest) gets a preview: one
	// page, reduced to the standard size of 24 if more was asked, with the true
	// total in the metadata. A later page, or a rarity, category or tag filter,
	// is a 401 login_required.
	SearchCards(ctx context.Context, filter dto.CardFilter) ([]dto.CardSummary, dto.PaginationMeta, error)
	// ListFacets returns each filter's available options given filter: the
	// values on Cards matching every other filter, plus the filter's own
	// selected values (flagged unavailable when unreachable). A Guest
	// (filter.Guest) gets a 401 login_required: facets are the costliest query.
	ListFacets(ctx context.Context, filter dto.CardFilter) (dto.CatalogFacets, error)
}

type catalogService struct {
	repo   repository.CatalogRepository
	images mapper.ImageHost
}

// NewCatalogService builds a CatalogService backed by repo, serving hosted
// images through images.
func NewCatalogService(repo repository.CatalogRepository, images mapper.ImageHost) CatalogService {
	return &catalogService{repo: repo, images: images}
}

func (s *catalogService) ListSeries(ctx context.Context) (dto.SeriesBrowseResult, error) {
	series, err := s.repo.ListSeries(ctx)
	if err != nil {
		return dto.SeriesBrowseResult{}, err
	}

	sets, err := s.repo.ListExpansionSets(ctx, ezutil.MapSlice(series, func(sr entity.Series) uuid.UUID { return sr.ID }))
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
		setsBySeries[*set.SeriesID] = append(setsBySeries[*set.SeriesID], mapper.ToExpansionSetSummary(s.images, set))
	}

	return dto.SeriesBrowseResult{
		Series: ezutil.MapSlice(series, func(sr entity.Series) dto.SeriesSummary {
			return mapper.ToSeriesSummary(s.images, sr, setsBySeries[sr.ID])
		}),
		UngroupedExpansionSets: ezutil.MapSlice(ungrouped, func(set entity.ExpansionSet) dto.ExpansionSetSummary {
			return mapper.ToExpansionSetSummary(s.images, set)
		}),
	}, nil
}

func (s *catalogService) ListRarities(ctx context.Context) ([]dto.RaritySummary, error) {
	rarities, err := s.repo.ListRarities(ctx)
	if err != nil {
		return nil, err
	}

	return ezutil.MapSlice(rarities, mapper.ToRaritySummary), nil
}

func (s *catalogService) ListCategories(ctx context.Context) ([]string, error) {
	return s.repo.ListDistinctCategories(ctx)
}

func (s *catalogService) ListTags(ctx context.Context) ([]string, error) {
	return s.repo.ListDistinctTags(ctx)
}

func (s *catalogService) SearchCards(ctx context.Context, filter dto.CardFilter) ([]dto.CardSummary, dto.PaginationMeta, error) {
	if filter.Guest {
		if filter.Page > 1 || len(filter.RarityIDs) > 0 || len(filter.Categories) > 0 || len(filter.Tags) > 0 {
			return nil, dto.PaginationMeta{}, apperr.WithCode(ungerr.UnauthorizedError(loginRequiredMsg), apperr.CodeLoginRequired)
		}
		// One standard page is all a Guest sees; a smaller size is kept.
		filter.Limit = min(filter.Limit, defaultCardSearchLimit)
	}

	repoFilter, page, limit := pagedRepoFilter(filter)

	results, total, err := s.repo.SearchCards(ctx, repoFilter)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	return ezutil.MapSlice(results, func(r repository.CardResult) dto.CardSummary { return mapper.ToCardSummary(s.images, r) }), dto.PaginationMeta{Total: int(total), Page: page, Limit: limit}, nil
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
	if filter.Guest {
		return dto.CatalogFacets{}, apperr.WithCode(ungerr.UnauthorizedError(loginRequiredMsg), apperr.CodeLoginRequired)
	}

	facets, err := s.repo.ListCardFacets(ctx, mapper.ToRepoCardFilter(filter))
	if err != nil {
		return dto.CatalogFacets{}, err
	}

	return mapper.ToCatalogFacets(s.images, facets), nil
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
