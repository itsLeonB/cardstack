package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	"github.com/itsLeonB/ungerr"
)

// CatalogHandler serves the unauthenticated, read-only catalog browse/search
// surface (ticket 05): listing Series/Expansion Sets to browse, and
// searching/filtering Cards. None of its routes are Secured - catalog
// browsing includes Cards the caller doesn't own, since Collections/
// Inventory (ownership) don't exist yet (tickets 06/07).
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

// searchCardsInput is GET /catalog/cards's query string. Every filter field
// is optional; an empty/zero value means "don't filter on this facet" (see
// dto.CardFilter).
type searchCardsInput struct {
	Name           string `query:"name" doc:"Case-insensitive substring match on the Card's name."`
	ExpansionSetID string `query:"expansionSetId" doc:"Only Cards in this Expansion Set."`
	LocalID        string `query:"localId" doc:"Only the Card with this number within its Expansion Set (e.g. \"001\"), typically combined with expansionSetId."`
	RarityID       string `query:"rarityId" doc:"Only Cards with this Rarity."`
	Category       string `query:"category" doc:"Only Cards with this exact category (e.g. Pokémon, Trainer, Energi)."`
	Tag            string `query:"tag" doc:"Only Cards carrying this tag."`
	Page           int    `query:"page" default:"1" minimum:"1" doc:"1-indexed page number."`
	Limit          int    `query:"limit" default:"24" minimum:"1" maximum:"100" doc:"Page size."`
}

// buildCardFilter validates and converts in into a dto.CardFilter,
// parsing its string ID fields to uuid.UUID up front so an invalid ID is
// rejected as a 400 before any query runs, rather than surfacing as an
// empty/mismatched result.
func buildCardFilter(in searchCardsInput) (dto.CardFilter, error) {
	filter := dto.CardFilter{
		Name:     in.Name,
		LocalID:  in.LocalID,
		Category: in.Category,
		Tag:      in.Tag,
		Page:     in.Page,
		Limit:    in.Limit,
	}

	if in.ExpansionSetID != "" {
		id, err := uuid.Parse(in.ExpansionSetID)
		if err != nil {
			return dto.CardFilter{}, ungerr.BadRequestError("invalid expansionSetId: " + in.ExpansionSetID)
		}
		filter.ExpansionSetID = id
	}

	if in.RarityID != "" {
		id, err := uuid.Parse(in.RarityID)
		if err != nil {
			return dto.CardFilter{}, ungerr.BadRequestError("invalid rarityId: " + in.RarityID)
		}
		filter.RarityID = id
	}

	return filter, nil
}

func (h *CatalogHandler) searchCards(ctx context.Context, in searchCardsInput) ([]dto.CardSummary, dto.PaginationMeta, error) {
	filter, err := buildCardFilter(in)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	return h.catalogSvc.SearchCards(ctx, filter)
}

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
			OperationID: "search-catalog-cards",
			Method:      http.MethodGet,
			Path:        "/catalog/cards",
			Summary:     "Search/browse Cards by name, Expansion Set + card number, rarity, category, and tag",
			Tags:        []string{"catalog"},
			SuccessCode: http.StatusOK,
			Secured:     false,
			HandlerFunc: h.searchCards,
		}),
	}
}
