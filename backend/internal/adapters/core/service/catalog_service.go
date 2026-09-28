package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/repository"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	domainservice "github.com/itsLeonB/cardstack/backend/internal/domain/service"
)

// defaultCardSearchLimit/maxCardSearchLimit bound CardFilter.Limit before it
// ever reaches a query: a caller that omits Limit gets a reasonable page
// size, and one that asks for an unreasonably large page is clamped rather
// than allowed to pull the whole catalog in one response.
const (
	defaultCardSearchLimit = 24
	maxCardSearchLimit     = 100
)

type catalogService struct {
	repo repository.CatalogRepository
}

// NewCatalogService builds a domainservice.CatalogService backed by repo.
func NewCatalogService(repo repository.CatalogRepository) domainservice.CatalogService {
	return &catalogService{repo: repo}
}

func (s *catalogService) ListSeries(ctx context.Context) (domainservice.SeriesBrowseResult, error) {
	series, err := s.repo.ListSeries(ctx)
	if err != nil {
		return domainservice.SeriesBrowseResult{}, err
	}

	seriesIDs := make([]uuid.UUID, len(series))
	for i, sr := range series {
		seriesIDs[i] = sr.ID
	}

	sets, err := s.repo.ListExpansionSets(ctx, seriesIDs)
	if err != nil {
		return domainservice.SeriesBrowseResult{}, err
	}

	ungrouped, err := s.repo.ListUngroupedExpansionSets(ctx)
	if err != nil {
		return domainservice.SeriesBrowseResult{}, err
	}

	setsBySeries := make(map[uuid.UUID][]domainservice.ExpansionSetSummary, len(series))
	for _, set := range sets {
		if set.SeriesID == nil {
			continue
		}
		setsBySeries[*set.SeriesID] = append(setsBySeries[*set.SeriesID], mapper.ToExpansionSetSummary(set))
	}

	summaries := make([]domainservice.SeriesSummary, len(series))
	for i, sr := range series {
		summaries[i] = domainservice.SeriesSummary{
			ID:            sr.ID,
			Code:          sr.Code,
			Name:          sr.Name,
			ExpansionSets: setsBySeries[sr.ID],
		}
	}

	ungroupedSummaries := make([]domainservice.ExpansionSetSummary, len(ungrouped))
	for i, set := range ungrouped {
		ungroupedSummaries[i] = mapper.ToExpansionSetSummary(set)
	}

	return domainservice.SeriesBrowseResult{
		Series:                 summaries,
		UngroupedExpansionSets: ungroupedSummaries,
	}, nil
}

func (s *catalogService) ListRarities(ctx context.Context) ([]domainservice.RaritySummary, error) {
	rarities, err := s.repo.ListRarities(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]domainservice.RaritySummary, len(rarities))
	for i, r := range rarities {
		summaries[i] = domainservice.RaritySummary{ID: r.ID, Code: r.Code, Name: r.Name}
	}

	return summaries, nil
}

func (s *catalogService) ListCategories(ctx context.Context) ([]string, error) {
	return s.repo.ListDistinctCategories(ctx)
}

func (s *catalogService) ListTags(ctx context.Context) ([]string, error) {
	return s.repo.ListDistinctTags(ctx)
}

func (s *catalogService) SearchCards(ctx context.Context, filter domainservice.CardFilter) ([]domainservice.CardSummary, domainservice.PaginationMeta, error) {
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
		return nil, domainservice.PaginationMeta{}, err
	}

	cards := make([]domainservice.CardSummary, len(results))
	for i, r := range results {
		cards[i] = mapper.ToCardSummary(r)
	}

	return cards, domainservice.PaginationMeta{
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
