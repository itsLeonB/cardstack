package handler

import (
	"context"
	"net/http"

	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
)

type SharedCollectionHandler struct {
	sharedSvc service.SharedCollectionService
}

func NewSharedCollectionHandler(sharedSvc service.SharedCollectionService) *SharedCollectionHandler {
	return &SharedCollectionHandler{sharedSvc: sharedSvc}
}

// sharedLookupInput addresses a Shared Collection by the two Slugs in its link.
type sharedLookupInput struct {
	ProfileSlug    string `path:"profileSlug" doc:"The owner's profile Slug."`
	CollectionSlug string `path:"collectionSlug" doc:"The Collection's Slug."`
}

// sharedEntriesInput is GET /shared/{profileSlug}/{collectionSlug}/entries: the
// owner's entries listing query, filters and pagination alike.
type sharedEntriesInput struct {
	ProfileSlug    string `path:"profileSlug" doc:"The owner's profile Slug."`
	CollectionSlug string `path:"collectionSlug" doc:"The Collection's Slug."`
	CardFilterParams
	CardIDs []string `query:"cardId,explode" maxItems:"100" doc:"Only these Cards (repeatable, at most 100), combined with the other filters."`
	Page    int      `query:"page" default:"1" minimum:"1" doc:"1-indexed page number."`
	Limit   int      `query:"limit" default:"24" minimum:"1" maximum:"100" doc:"Page size."`
}

// sharedFacetsInput is GET /shared/{profileSlug}/{collectionSlug}/facets: the
// same filters, without pagination.
type sharedFacetsInput struct {
	ProfileSlug    string `path:"profileSlug" doc:"The owner's profile Slug."`
	CollectionSlug string `path:"collectionSlug" doc:"The Collection's Slug."`
	CardFilterParams
}

// sharedPageInput is GET /shared/{profileSlug}/{collectionSlug}/pages/{page}: one
// Page of a Shared binder.
type sharedPageInput struct {
	ProfileSlug    string `path:"profileSlug" doc:"The owner's profile Slug."`
	CollectionSlug string `path:"collectionSlug" doc:"The Collection's Slug."`
	Page           int    `path:"page" minimum:"1" doc:"Page number in turning order, counted from 1."`
}

func (h *SharedCollectionHandler) get(ctx context.Context, in sharedLookupInput) (dto.SharedCollectionSummary, error) {
	return h.sharedSvc.Get(ctx, dto.SharedCollectionLookup{ProfileSlug: in.ProfileSlug, CollectionSlug: in.CollectionSlug})
}

func (h *SharedCollectionHandler) listEntries(ctx context.Context, in sharedEntriesInput) ([]dto.InventoryItem, dto.PaginationMeta, error) {
	filter, err := buildCardFilter(in.CardFilterParams)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	if filter.CardIDs, err = parseUUIDs("cardId", in.CardIDs); err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	filter.Page, filter.Limit = in.Page, in.Limit

	return h.sharedSvc.ListEntries(ctx, dto.SharedCollectionListRequest{
		Lookup: dto.SharedCollectionLookup{ProfileSlug: in.ProfileSlug, CollectionSlug: in.CollectionSlug},
		Filter: filter,
	})
}

func (h *SharedCollectionHandler) listFacets(ctx context.Context, in sharedFacetsInput) (dto.CatalogFacets, error) {
	filter, err := buildCardFilter(in.CardFilterParams)
	if err != nil {
		return dto.CatalogFacets{}, err
	}

	return h.sharedSvc.ListFacets(ctx, dto.SharedCollectionListRequest{
		Lookup: dto.SharedCollectionLookup{ProfileSlug: in.ProfileSlug, CollectionSlug: in.CollectionSlug},
		Filter: filter,
	})
}

func (h *SharedCollectionHandler) getPage(ctx context.Context, in sharedPageInput) (dto.BinderPage, error) {
	return h.sharedSvc.GetPage(ctx, dto.SharedCollectionPageRequest{
		Lookup: dto.SharedCollectionLookup{ProfileSlug: in.ProfileSlug, CollectionSlug: in.CollectionSlug},
		Page:   in.Page,
	})
}

// Routes are guest-allowed: the router passes the guest guard and the per-client
// limit (ratelimit.PerIP), never the per-user one.
func (h *SharedCollectionHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[sharedLookupInput, dto.SharedCollectionSummary]{
			OperationID: "get-shared-collection",
			Method:      http.MethodGet,
			Path:        "/shared/{profileSlug}/{collectionSlug}",
			Summary:     "Get a Shared Collection's header (owner's display name, title, description, and a binder's Layout and Unidentified count) by its link; any viewer, a Guest included. Any private, unshared or missing link is the same 404",
			Tags:        []string{"shared"},
			SuccessCode: http.StatusOK,
			HandlerFunc: h.get,
		}),
		endpoint.NewWithMeta(endpoint.EndpointWithMeta[sharedEntriesInput, []dto.InventoryItem, dto.PaginationMeta]{
			OperationID: "list-shared-collection-entries",
			Method:      http.MethodGet,
			Path:        "/shared/{profileSlug}/{collectionSlug}/entries",
			Summary:     "Search/page the Cards and quantities of a Shared Collection, with the owner's listing's filters; any viewer, a Guest included",
			Tags:        []string{"shared"},
			SuccessCode: http.StatusOK,
			HandlerFunc: h.listEntries,
		}),
		endpoint.New(endpoint.Endpoint[sharedFacetsInput, dto.CatalogFacets]{
			OperationID: "list-shared-collection-facets",
			Method:      http.MethodGet,
			Path:        "/shared/{profileSlug}/{collectionSlug}/facets",
			Summary:     "List each filter's available options computed only from the Cards of a Shared Collection; any viewer, a Guest included",
			Tags:        []string{"shared"},
			SuccessCode: http.StatusOK,
			HandlerFunc: h.listFacets,
		}),
		endpoint.New(endpoint.Endpoint[sharedPageInput, dto.BinderPage]{
			OperationID: "get-shared-collection-page",
			Method:      http.MethodGet,
			Path:        "/shared/{profileSlug}/{collectionSlug}/pages/{page}",
			Summary:     "Get one Page of a Shared binder: its Slots, with Cards of unpublished sets read as empty; any viewer, a Guest included. A plain Collection or a Page past the last one is the same 404 as a missing link",
			Tags:        []string{"shared"},
			SuccessCode: http.StatusOK,
			HandlerFunc: h.getPage,
		}),
	}
}
