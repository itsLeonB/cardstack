package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	"github.com/itsLeonB/ungerr"
)

// CatalogHandler serves the read-only catalog browse/search surface (ticket
// 05): listing Series/Expansion Sets to browse, and searching/filtering Cards.
// Its routes are registered with the guest-allowed guard, so a caller with no
// token reaches them; the card search and facets tell a Guest apart through
// authpkg.CallerFrom and the catalog service applies the preview lock.
type CatalogHandler struct {
	catalogSvc service.CatalogService
}

// NewCatalogHandler builds a CatalogHandler.
func NewCatalogHandler(catalogSvc service.CatalogService) *CatalogHandler {
	return &CatalogHandler{catalogSvc: catalogSvc}
}

type listSeriesInput struct{}

func (h *CatalogHandler) listSeries(ctx context.Context, _ listSeriesInput) (dto.SeriesBrowseResult, error) {
	return h.catalogSvc.ListSeries(ctx)
}

type listRaritiesInput struct{}

func (h *CatalogHandler) listRarities(ctx context.Context, _ listRaritiesInput) ([]dto.RaritySummary, error) {
	return h.catalogSvc.ListRarities(ctx)
}

type listCategoriesInput struct{}

func (h *CatalogHandler) listCategories(ctx context.Context, _ listCategoriesInput) ([]string, error) {
	return h.catalogSvc.ListCategories(ctx)
}

type listTagsInput struct{}

func (h *CatalogHandler) listTags(ctx context.Context, _ listTagsInput) ([]string, error) {
	return h.catalogSvc.ListTags(ctx)
}

// CardFilterParams are the filter query parameters shared by GET
// /catalog/cards and GET /catalog/facets. Every field is optional; an empty
// value means "don't filter on this facet". The multi-value ones are
// repeated params (?rarityId=a&rarityId=b): OR within a param, AND across.
type CardFilterParams struct {
	Name            string   `query:"name" maxLength:"64" doc:"Case-insensitive substring match on the Card's name (at most 64 characters)."`
	ExpansionSetIDs []string `query:"expansionSetId,explode" maxItems:"20" doc:"Only Cards in any of these Expansion Sets (repeatable, at most 20)."`
	LocalID         string   `query:"localId" doc:"Only the Card with this number within its Expansion Set (e.g. \"001\")."`
	RarityIDs       []string `query:"rarityId,explode" maxItems:"20" doc:"Only Cards with any of these Rarities (repeatable, at most 20)."`
	Categories      []string `query:"category,explode" maxItems:"20" doc:"Only Cards with any of these exact categories (repeatable, at most 20; e.g. Pokémon, Trainer, Energi)."`
	Tags            []string `query:"tag,explode" maxItems:"20" doc:"Only Cards carrying any of these tags (repeatable, at most 20)."`
}

// searchCardsInput is GET /catalog/cards's query string.
type searchCardsInput struct {
	CardFilterParams
	CardIDs []string `query:"cardId,explode" maxItems:"100" doc:"Only these Cards (repeatable, at most 100), combined with the other filters; use it to render Cards known by id."`
	Page    int      `query:"page" default:"1" minimum:"1" doc:"1-indexed page number. A Guest may only ask for page 1; a later page is 401 login_required."`
	Limit   int      `query:"limit" default:"24" minimum:"1" maximum:"100" doc:"Page size. A Guest gets at most 24."`
}

// listFacetsInput is GET /catalog/facets's query string: the same filters
// as the search, without pagination.
type listFacetsInput struct {
	CardFilterParams
}

func parseUUIDs(param string, raw []string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for _, r := range raw {
		if r == "" {
			continue
		}
		id, err := uuid.Parse(r)
		if err != nil {
			return nil, ungerr.BadRequestError("invalid " + param)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// buildCardFilter validates and converts p into a dto.CardFilter, parsing
// its string ID fields to uuid.UUID up front so an invalid ID is rejected
// as a 400 before any query runs, rather than surfacing as an
// empty/mismatched result.
func buildCardFilter(p CardFilterParams) (dto.CardFilter, error) {
	setIDs, err := parseUUIDs("expansionSetId", p.ExpansionSetIDs)
	if err != nil {
		return dto.CardFilter{}, err
	}
	rarityIDs, err := parseUUIDs("rarityId", p.RarityIDs)
	if err != nil {
		return dto.CardFilter{}, err
	}

	return dto.CardFilter{
		Name:            p.Name,
		ExpansionSetIDs: setIDs,
		LocalID:         p.LocalID,
		RarityIDs:       rarityIDs,
		Categories:      p.Categories,
		Tags:            p.Tags,
	}, nil
}

func (h *CatalogHandler) searchCards(ctx context.Context, in searchCardsInput) ([]dto.CardSummary, dto.PaginationMeta, error) {
	filter, err := buildCardFilter(in.CardFilterParams)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	if filter.CardIDs, err = parseUUIDs("cardId", in.CardIDs); err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	filter.Page = in.Page
	filter.Limit = in.Limit
	filter.Guest = authpkg.CallerFrom(ctx).IsGuest()

	return h.catalogSvc.SearchCards(ctx, filter)
}

func (h *CatalogHandler) listFacets(ctx context.Context, in listFacetsInput) (dto.CatalogFacets, error) {
	filter, err := buildCardFilter(in.CardFilterParams)
	if err != nil {
		return dto.CatalogFacets{}, err
	}
	filter.Guest = authpkg.CallerFrom(ctx).IsGuest()

	return h.catalogSvc.ListFacets(ctx, filter)
}

// Operation IDs the router gives the tighter per-user rate-limit tiers.
const (
	OpSearchCatalogCards = "search-catalog-cards"
	OpListCatalogFacets  = "list-catalog-facets"
)

// Routes returns every route CatalogHandler exposes, for registration via
// endpoint.RegisterAll.
func (h *CatalogHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[listSeriesInput, dto.SeriesBrowseResult]{
			OperationID: "list-catalog-series",
			Method:      http.MethodGet,
			Path:        "/catalog/series",
			Summary:     "List every Series (with its Expansion Sets nested) plus every ungrouped Expansion Set",
			Tags:        []string{"catalog"},
			SuccessCode: http.StatusOK,
			Secured:     false,
			HandlerFunc: h.listSeries,
		}),
		endpoint.NewList(endpoint.ListEndpoint[listRaritiesInput, dto.RaritySummary]{
			OperationID: "list-catalog-rarities",
			Method:      http.MethodGet,
			Path:        "/catalog/rarities",
			Summary:     "List every Rarity, for use as a search filter value",
			Tags:        []string{"catalog"},
			Secured:     false,
			HandlerFunc: h.listRarities,
		}),
		endpoint.NewList(endpoint.ListEndpoint[listCategoriesInput, string]{
			OperationID: "list-catalog-categories",
			Method:      http.MethodGet,
			Path:        "/catalog/categories",
			Summary:     "List the distinct Card categories in use, for use as a search filter value",
			Tags:        []string{"catalog"},
			Secured:     false,
			HandlerFunc: h.listCategories,
		}),
		endpoint.NewList(endpoint.ListEndpoint[listTagsInput, string]{
			OperationID: "list-catalog-tags",
			Method:      http.MethodGet,
			Path:        "/catalog/tags",
			Summary:     "List the distinct Card tags in use, for use as a search filter value",
			Tags:        []string{"catalog"},
			Secured:     false,
			HandlerFunc: h.listTags,
		}),
		endpoint.NewWithMeta(endpoint.EndpointWithMeta[searchCardsInput, []dto.CardSummary, dto.PaginationMeta]{
			OperationID: OpSearchCatalogCards,
			Method:      http.MethodGet,
			Path:        "/catalog/cards",
			Summary:     "Search/browse Cards by name, Expansion Set + card number, rarity, category, and tag. A Guest gets one page of at most 24; a later page or a rarity, category or tag filter is 401 login_required",
			Tags:        []string{"catalog"},
			SuccessCode: http.StatusOK,
			Secured:     false,
			HandlerFunc: h.searchCards,
		}),
		endpoint.New(endpoint.Endpoint[listFacetsInput, dto.CatalogFacets]{
			OperationID: OpListCatalogFacets,
			Method:      http.MethodGet,
			Path:        "/catalog/facets",
			Summary:     "List each search filter's available options given the active filters (a filter's own selection is excluded from its options' calculation; selected values are always included). A Guest gets 401 login_required",
			Tags:        []string{"catalog"},
			SuccessCode: http.StatusOK,
			Secured:     false,
			HandlerFunc: h.listFacets,
		}),
	}
}
