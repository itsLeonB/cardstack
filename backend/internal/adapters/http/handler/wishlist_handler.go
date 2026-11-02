package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
)

const (
	OpListWishlist           = "list-wishlist"
	OpListWishlistFacets     = "list-wishlist-facets"
	OpSetWishlistItem        = "set-wishlist-item"
	OpRemoveWishlistItem     = "remove-wishlist-item"
	OpClearFulfilledWishlist = "clear-fulfilled-wishlist"
	OpPreviewWishlistPreset  = "preview-wishlist-preset"
	OpApplyWishlistPreset    = "apply-wishlist-preset"
)

type WishlistHandler struct {
	wishlistSvc service.WishlistService
}

func NewWishlistHandler(wishlistSvc service.WishlistService) *WishlistHandler {
	return &WishlistHandler{wishlistSvc: wishlistSvc}
}

// The filter params are repeated on each input rather than embedded one level
// deeper: Huma reads one level of embedding only.
type listWishlistInput struct {
	CardFilterParams
	CardIDs []string `query:"cardId,explode" maxItems:"100" doc:"Only these Cards (repeatable, at most 100), combined with the other filters."`
	Status  string   `query:"status" default:"all" enum:"all,needed,fulfilled" doc:"All, Still needed (owned below wanted) or Fulfilled (owned at least wanted). Derived on read."`
	Page    int      `query:"page" default:"1" minimum:"1" doc:"1-indexed page number."`
	Limit   int      `query:"limit" default:"24" minimum:"1" maximum:"100" doc:"Page size."`
}

type listWishlistFacetsInput struct {
	CardFilterParams
	CardIDs []string `query:"cardId,explode" maxItems:"100" doc:"Only these Cards (repeatable, at most 100), combined with the other filters."`
	Status  string   `query:"status" default:"all" enum:"all,needed,fulfilled" doc:"All, Still needed (owned below wanted) or Fulfilled (owned at least wanted). Derived on read."`
}

type setWishlistItemInput struct {
	CardID uuid.UUID `path:"cardId" doc:"Card ID"`
	Body   struct {
		Quantity int `json:"quantity" required:"true" minimum:"1" maximum:"2147483647" doc:"Wanted quantity; replaces the item's when it is already on the Wishlist."`
	}
}

type wishlistItemInput struct {
	CardID uuid.UUID `path:"cardId" doc:"Card ID"`
}

type clearFulfilledWishlistInput struct{}

type wishlistPresetInput struct {
	Body struct {
		ExpansionSetIDs []string `json:"expansionSetIds" required:"true" minItems:"1" doc:"Expansion Sets to include."`
		Quantity        int      `json:"quantity" required:"true" minimum:"1" maximum:"2147483647" doc:"Quantity to add for each Card."`
		RarityIDs       []string `json:"rarityIds,omitempty" doc:"Narrow by rarity (repeatable)."`
		Categories      []string `json:"categories,omitempty" doc:"Narrow by category (repeatable)."`
		Tags            []string `json:"tags,omitempty" doc:"Narrow by tag (repeatable)."`
	}
}

func (h *WishlistHandler) list(ctx context.Context, in listWishlistInput) ([]dto.WishlistItem, dto.PaginationMeta, error) {
	req, err := wishlistRequest(ctx, in.CardFilterParams, in.CardIDs, in.Status)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	req.Filter.Page, req.Filter.Limit = in.Page, in.Limit

	return h.wishlistSvc.List(ctx, req)
}

func (h *WishlistHandler) listFacets(ctx context.Context, in listWishlistFacetsInput) (dto.CatalogFacets, error) {
	req, err := wishlistRequest(ctx, in.CardFilterParams, in.CardIDs, in.Status)
	if err != nil {
		return dto.CatalogFacets{}, err
	}

	return h.wishlistSvc.ListFacets(ctx, req)
}

func (h *WishlistHandler) set(ctx context.Context, in setWishlistItemInput) error {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return err
	}

	return h.wishlistSvc.SetQuantity(ctx, dto.WishlistItemRequest{
		ProfileID: profileID,
		CardID:    in.CardID,
		Quantity:  in.Body.Quantity,
	})
}

func (h *WishlistHandler) remove(ctx context.Context, in wishlistItemInput) error {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return err
	}

	return h.wishlistSvc.Remove(ctx, dto.WishlistItemLookup{ProfileID: profileID, CardID: in.CardID})
}

func (h *WishlistHandler) clearFulfilled(ctx context.Context, _ clearFulfilledWishlistInput) (dto.WishlistClearResult, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.WishlistClearResult{}, err
	}

	return h.wishlistSvc.ClearFulfilled(ctx, dto.WishlistLookup{ProfileID: profileID})
}

func (h *WishlistHandler) previewPreset(ctx context.Context, in wishlistPresetInput) (dto.WishlistPresetResult, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.WishlistPresetResult{}, err
	}

	expansionSetIDs, err := parseUUIDs("expansionSetId", in.Body.ExpansionSetIDs)
	if err != nil {
		return dto.WishlistPresetResult{}, err
	}
	rarityIDs, err := parseUUIDs("rarityId", in.Body.RarityIDs)
	if err != nil {
		return dto.WishlistPresetResult{}, err
	}

	return h.wishlistSvc.PreviewPreset(ctx, dto.WishlistPresetRequest{
		ProfileID:       profileID,
		ExpansionSetIDs: expansionSetIDs,
		Filter: dto.CardFilter{
			RarityIDs:  rarityIDs,
			Categories: in.Body.Categories,
			Tags:       in.Body.Tags,
		},
		Quantity: in.Body.Quantity,
	})
}

func (h *WishlistHandler) applyPreset(ctx context.Context, in wishlistPresetInput) (dto.WishlistPresetResult, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.WishlistPresetResult{}, err
	}

	expansionSetIDs, err := parseUUIDs("expansionSetId", in.Body.ExpansionSetIDs)
	if err != nil {
		return dto.WishlistPresetResult{}, err
	}
	rarityIDs, err := parseUUIDs("rarityId", in.Body.RarityIDs)
	if err != nil {
		return dto.WishlistPresetResult{}, err
	}

	return h.wishlistSvc.ApplyPreset(ctx, dto.WishlistPresetRequest{
		ProfileID:       profileID,
		ExpansionSetIDs: expansionSetIDs,
		Filter: dto.CardFilter{
			RarityIDs:  rarityIDs,
			Categories: in.Body.Categories,
			Tags:       in.Body.Tags,
		},
		Quantity: in.Body.Quantity,
	})
}

func wishlistRequest(ctx context.Context, params CardFilterParams, cardIDs []string, status string) (dto.WishlistListRequest, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.WishlistListRequest{}, err
	}

	filter, err := buildCardFilter(params)
	if err != nil {
		return dto.WishlistListRequest{}, err
	}
	if filter.CardIDs, err = parseUUIDs("cardId", cardIDs); err != nil {
		return dto.WishlistListRequest{}, err
	}

	return dto.WishlistListRequest{ProfileID: profileID, Status: status, Filter: filter}, nil
}

func (h *WishlistHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.NewWithMeta(endpoint.EndpointWithMeta[listWishlistInput, []dto.WishlistItem, dto.PaginationMeta]{
			OperationID: OpListWishlist,
			Method:      http.MethodGet,
			Path:        "/wishlist",
			Summary:     "Search/page the current user's Wishlist with the catalog search filters: each wanted Card with the wanted quantity, the quantity owned in Master Inventory, and whether it is fulfilled (computed on read)",
			Tags:        []string{"wishlist"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.list,
		}),
		endpoint.New(endpoint.Endpoint[listWishlistFacetsInput, dto.CatalogFacets]{
			OperationID: OpListWishlistFacets,
			Method:      http.MethodGet,
			Path:        "/wishlist/facets",
			Summary:     "List each filter's available options computed only from the Cards on the current user's Wishlist (same faceted rule as the catalog)",
			Tags:        []string{"wishlist"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.listFacets,
		}),
		endpoint.NewNoBody(endpoint.NoBodyEndpoint[setWishlistItemInput]{
			OperationID: OpSetWishlistItem,
			Method:      http.MethodPut,
			Path:        "/wishlist/items/{cardId}",
			Summary:     "Put a Card on the current user's Wishlist with a wanted quantity, replacing the wanted quantity when it is already there",
			Tags:        []string{"wishlist"},
			Secured:     true,
			HandlerFunc: h.set,
		}),
		endpoint.NewNoBody(endpoint.NoBodyEndpoint[wishlistItemInput]{
			OperationID: OpRemoveWishlistItem,
			Method:      http.MethodDelete,
			Path:        "/wishlist/items/{cardId}",
			Summary:     "Remove a Card from the current user's Wishlist",
			Tags:        []string{"wishlist"},
			Secured:     true,
			HandlerFunc: h.remove,
		}),
		endpoint.New(endpoint.Endpoint[clearFulfilledWishlistInput, dto.WishlistClearResult]{
			OperationID: OpClearFulfilledWishlist,
			Method:      http.MethodDelete,
			Path:        "/wishlist/fulfilled",
			Summary:     "Remove every fulfilled item from the current user's Wishlist in one call, and return how many were removed",
			Tags:        []string{"wishlist"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.clearFulfilled,
		}),
		endpoint.New(endpoint.Endpoint[wishlistPresetInput, dto.WishlistPresetResult]{
			OperationID: OpPreviewWishlistPreset,
			Method:      http.MethodPost,
			Path:        "/wishlist/preset/preview",
			Summary:     "Preview a Wishlist Preset: return the Cards that would be added and how many, for the chosen Expansion Sets (optionally narrowed by rarity, category and tag) and quantity, of which the user owns none in Master Inventory and none are already on the Wishlist",
			Tags:        []string{"wishlist"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.previewPreset,
		}),
		endpoint.New(endpoint.Endpoint[wishlistPresetInput, dto.WishlistPresetResult]{
			OperationID: OpApplyWishlistPreset,
			Method:      http.MethodPost,
			Path:        "/wishlist/preset/apply",
			Summary:     "Apply a Wishlist Preset: add the matching Cards to the Wishlist with the chosen quantity, using the same query as preview",
			Tags:        []string{"wishlist"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.applyPreset,
		}),
	}
}
