package handler

import (
	"context"
	"net/http"

	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
)

type SharedWishlistHandler struct {
	sharedSvc service.SharedWishlistService
}

func NewSharedWishlistHandler(sharedSvc service.SharedWishlistService) *SharedWishlistHandler {
	return &SharedWishlistHandler{sharedSvc: sharedSvc}
}

// sharedWishlistLookupInput addresses a Shared Wishlist by the profile Slug in
// its link (docs/adr/0025). There is one Wishlist per profile at a fixed path.
type sharedWishlistLookupInput struct {
	ProfileSlug string `path:"profileSlug" doc:"The owner's profile Slug."`
}

// sharedWishlistEntriesInput is GET /shared/{profileSlug}/wishlist/entries: the
// owner's entries listing query, filters and pagination alike.
type sharedWishlistEntriesInput struct {
	ProfileSlug string `path:"profileSlug" doc:"The owner's profile Slug."`
	CardFilterParams
	CardIDs []string `query:"cardId,explode" maxItems:"100" doc:"Only these Cards (repeatable, at most 100), combined with the other filters."`
	Status  string   `query:"status" default:"all" enum:"all,needed,fulfilled" doc:"All, Still needed (owned below wanted) or Fulfilled (owned at least wanted). Derived on read."`
	Page    int      `query:"page" default:"1" minimum:"1" doc:"1-indexed page number."`
	Limit   int      `query:"limit" default:"24" minimum:"1" maximum:"100" doc:"Page size."`
	// ViewerFilter is the signed-in viewer's own "have"/"dont-have" filter on
	// their owned quantity of each item. A Guest or the owner gets no viewer data.
	ViewerFilter string `query:"viewerFilter" enum:"have,dont-have" doc:"For a signed-in viewer who is not the owner: only items the viewer owns (have) or does not own (dont-have) in their Master Inventory."`
}

// sharedWishlistFacetsInput is GET /shared/{profileSlug}/wishlist/facets: the
// same filters, without pagination.
type sharedWishlistFacetsInput struct {
	ProfileSlug string `path:"profileSlug" doc:"The owner's profile Slug."`
	CardFilterParams
	CardIDs []string `query:"cardId,explode" maxItems:"100" doc:"Only these Cards (repeatable, at most 100), combined with the other filters."`
	Status  string   `query:"status" default:"all" enum:"all,needed,fulfilled" doc:"All, Still needed (owned below wanted) or Fulfilled (owned at least wanted). Derived on read."`
}

func (h *SharedWishlistHandler) get(ctx context.Context, in sharedWishlistLookupInput) (dto.SharedWishlistSummary, error) {
	return h.sharedSvc.Get(ctx, dto.SharedWishlistLookup{ProfileSlug: in.ProfileSlug})
}

func (h *SharedWishlistHandler) listEntries(ctx context.Context, in sharedWishlistEntriesInput) ([]dto.WishlistItem, dto.PaginationMeta, error) {
	filter, err := buildCardFilter(in.CardFilterParams)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	if filter.CardIDs, err = parseUUIDs("cardId", in.CardIDs); err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	filter.Page, filter.Limit = in.Page, in.Limit

	viewer := authpkg.CallerFrom(ctx).ProfileID // uuid.Nil for a Guest

	return h.sharedSvc.ListEntries(ctx, dto.SharedWishlistListRequest{
		Lookup:          dto.SharedWishlistLookup{ProfileSlug: in.ProfileSlug},
		Filter:          filter,
		Status:          in.Status,
		ViewerProfileID: viewer,
		ViewerFilter:    in.ViewerFilter,
	})
}

func (h *SharedWishlistHandler) listFacets(ctx context.Context, in sharedWishlistFacetsInput) (dto.CatalogFacets, error) {
	filter, err := buildCardFilter(in.CardFilterParams)
	if err != nil {
		return dto.CatalogFacets{}, err
	}
	if filter.CardIDs, err = parseUUIDs("cardId", in.CardIDs); err != nil {
		return dto.CatalogFacets{}, err
	}

	return h.sharedSvc.ListFacets(ctx, dto.SharedWishlistListRequest{
		Lookup: dto.SharedWishlistLookup{ProfileSlug: in.ProfileSlug},
		Filter: filter,
		Status: in.Status,
	})
}

// Routes are guest-allowed: the router passes the guest guard and the per-client
// limit (ratelimit.PerIP), never the per-user one.
func (h *SharedWishlistHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[sharedWishlistLookupInput, dto.SharedWishlistSummary]{
			OperationID: "get-shared-wishlist",
			Method:      http.MethodGet,
			Path:        "/shared/{profileSlug}/wishlist",
			Summary:     "Get a Shared Wishlist's header (owner's display name) by its link; any viewer, a Guest included. Any private, unshared or missing link is the same 404",
			Tags:        []string{"shared"},
			SuccessCode: http.StatusOK,
			HandlerFunc: h.get,
		}),
		endpoint.NewWithMeta(endpoint.EndpointWithMeta[sharedWishlistEntriesInput, []dto.WishlistItem, dto.PaginationMeta]{
			OperationID: "list-shared-wishlist-entries",
			Method:      http.MethodGet,
			Path:        "/shared/{profileSlug}/wishlist/entries",
			Summary:     "Search/page the Cards and quantities of a Shared Wishlist, with the owner's listing's filters and the All/Still needed/Fulfilled status filter; any viewer, a Guest included",
			Tags:        []string{"shared"},
			SuccessCode: http.StatusOK,
			HandlerFunc: h.listEntries,
		}),
		endpoint.New(endpoint.Endpoint[sharedWishlistFacetsInput, dto.CatalogFacets]{
			OperationID: "list-shared-wishlist-facets",
			Method:      http.MethodGet,
			Path:        "/shared/{profileSlug}/wishlist/facets",
			Summary:     "List each filter's available options computed only from the Cards of a Shared Wishlist; any viewer, a Guest included",
			Tags:        []string{"shared"},
			SuccessCode: http.StatusOK,
			HandlerFunc: h.listFacets,
		}),
	}
}
