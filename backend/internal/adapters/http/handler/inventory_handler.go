package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	"github.com/itsLeonB/ezutil/v2"
)

type InventoryHandler struct {
	inventorySvc service.InventoryService
}

func NewInventoryHandler(inventorySvc service.InventoryService) *InventoryHandler {
	return &InventoryHandler{inventorySvc: inventorySvc}
}

type addEntryInput struct {
	CollectionID uuid.UUID `path:"id" doc:"Collection ID"`
	Body         struct {
		CardID   uuid.UUID `json:"cardId" required:"true" doc:"The Card to add."`
		Quantity int       `json:"quantity" required:"true" minimum:"1" maximum:"2147483647" doc:"Number of copies."`
	}
}

type entryInput struct {
	CollectionID uuid.UUID `path:"id" doc:"Collection ID"`
	CardID       uuid.UUID `path:"cardId" doc:"Card ID"`
}

type updateEntryInput struct {
	CollectionID uuid.UUID `path:"id" doc:"Collection ID"`
	CardID       uuid.UUID `path:"cardId" doc:"Card ID"`
	Body         struct {
		Quantity int `json:"quantity" required:"true" minimum:"1" maximum:"2147483647" doc:"New number of copies."`
	}
}

// bulkUpdateItem's maximum matches the other quantity fields: it is the
// largest value the integer column holds.
type bulkUpdateItem struct {
	CardID   uuid.UUID `json:"cardId" required:"true" doc:"The Card to change. Must be unique within the request (400 otherwise)."`
	Quantity int       `json:"quantity" required:"true" minimum:"0" maximum:"2147483647" doc:"Absolute target quantity. 0 removes the Card."`
}

type bulkUpdateEntriesInput struct {
	CollectionID uuid.UUID `path:"id" doc:"Collection ID"`
	Body         struct {
		Items []bulkUpdateItem `json:"items" required:"true" minItems:"1" maxItems:"100" doc:"Changes applied in order, at most 100."`
	}
}

// listEntriesInput is GET /collections/{id}/entries: the catalog search
// filters plus pagination.
type listEntriesInput struct {
	CollectionID uuid.UUID `path:"id" doc:"Collection ID"`
	CardFilterParams
	CardIDs []string `query:"cardId,explode" maxItems:"100" doc:"Only these Cards (repeatable, at most 100), combined with the other filters; use it to fetch quantities for the Cards on a page."`
	Page    int      `query:"page" default:"1" minimum:"1" doc:"1-indexed page number."`
	Limit   int      `query:"limit" default:"24" minimum:"1" maximum:"100" doc:"Page size."`
}

// listEntryFacetsInput is GET /collections/{id}/facets: the same filters,
// without pagination.
type listEntryFacetsInput struct {
	CollectionID uuid.UUID `path:"id" doc:"Collection ID"`
	CardFilterParams
}

type cardHoldingsInput struct {
	CardID uuid.UUID `path:"cardId" doc:"Card ID"`
}

func (h *InventoryHandler) listHoldings(ctx context.Context, in cardHoldingsInput) ([]dto.CardHolding, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return nil, err
	}

	return h.inventorySvc.ListCardHoldings(ctx, dto.CardHoldingsRequest{ProfileID: profileID, CardID: in.CardID})
}

func (h *InventoryHandler) list(ctx context.Context, in listEntriesInput) ([]dto.InventoryItem, dto.PaginationMeta, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	filter, err := buildCardFilter(in.CardFilterParams)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	if filter.CardIDs, err = parseUUIDs("cardId", in.CardIDs); err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	filter.Page, filter.Limit = in.Page, in.Limit

	return h.inventorySvc.List(ctx, dto.InventoryListRequest{ProfileID: profileID, CollectionID: in.CollectionID, Filter: filter})
}

func (h *InventoryHandler) listFacets(ctx context.Context, in listEntryFacetsInput) (dto.CatalogFacets, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.CatalogFacets{}, err
	}
	filter, err := buildCardFilter(in.CardFilterParams)
	if err != nil {
		return dto.CatalogFacets{}, err
	}

	return h.inventorySvc.ListFacets(ctx, dto.InventoryListRequest{ProfileID: profileID, CollectionID: in.CollectionID, Filter: filter})
}

func (h *InventoryHandler) add(ctx context.Context, in addEntryInput) (dto.InventoryEntry, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.InventoryEntry{}, err
	}

	return h.inventorySvc.Add(ctx, dto.InventoryEntryRequest{
		ProfileID:    profileID,
		CollectionID: in.CollectionID,
		CardID:       in.Body.CardID,
		Quantity:     in.Body.Quantity,
	})
}

func (h *InventoryHandler) update(ctx context.Context, in updateEntryInput) (dto.InventoryEntry, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.InventoryEntry{}, err
	}

	return h.inventorySvc.UpdateQuantity(ctx, dto.InventoryEntryRequest{
		ProfileID:    profileID,
		CollectionID: in.CollectionID,
		CardID:       in.CardID,
		Quantity:     in.Body.Quantity,
	})
}

func (h *InventoryHandler) bulkUpdate(ctx context.Context, in bulkUpdateEntriesInput) ([]dto.InventoryChangeResult, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return nil, err
	}

	items := ezutil.MapSlice(in.Body.Items, func(it bulkUpdateItem) dto.InventoryQuantityChange {
		return dto.InventoryQuantityChange{CardID: it.CardID, Quantity: it.Quantity}
	})

	return h.inventorySvc.BulkUpdate(ctx, dto.InventoryBulkUpdateRequest{ProfileID: profileID, CollectionID: in.CollectionID, Items: items})
}

func (h *InventoryHandler) remove(ctx context.Context, in entryInput) error {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return err
	}

	return h.inventorySvc.Remove(ctx, dto.InventoryEntryLookup{ProfileID: profileID, CollectionID: in.CollectionID, CardID: in.CardID})
}

// Routes sets Secured:true only as OpenAPI metadata; the router must pass
// SessionGuard to RegisterAll.
func (h *InventoryHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.NewWithMeta(endpoint.EndpointWithMeta[listEntriesInput, []dto.InventoryItem, dto.PaginationMeta]{
			OperationID: "list-collection-entries",
			Method:      http.MethodGet,
			Path:        "/collections/{id}/entries",
			Summary:     "Search/page the Cards and quantities in one of the current user's own Collections, with the catalog search filters",
			Tags:        []string{"inventory"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.list,
		}),
		endpoint.New(endpoint.Endpoint[cardHoldingsInput, []dto.CardHolding]{
			OperationID: "list-card-holdings",
			Method:      http.MethodGet,
			Path:        "/inventory/cards/{cardId}/holdings",
			Summary:     "List the current user's own Collections that hold a Card, with the quantity in each (empty when none)",
			Tags:        []string{"inventory"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.listHoldings,
		}),
		endpoint.New(endpoint.Endpoint[listEntryFacetsInput, dto.CatalogFacets]{
			OperationID: "list-collection-facets",
			Method:      http.MethodGet,
			Path:        "/collections/{id}/facets",
			Summary:     "List each filter's available options computed only from the Cards in one of the current user's own Collections (same faceted rule as the catalog)",
			Tags:        []string{"inventory"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.listFacets,
		}),
		endpoint.New(endpoint.Endpoint[addEntryInput, dto.InventoryEntry]{
			OperationID: "add-collection-entry",
			Method:      http.MethodPost,
			Path:        "/collections/{id}/entries",
			Summary:     "Add a Card to one of the current user's own Collections",
			Tags:        []string{"inventory"},
			SuccessCode: http.StatusCreated,
			Secured:     true,
			HandlerFunc: h.add,
		}),
		endpoint.New(endpoint.Endpoint[updateEntryInput, dto.InventoryEntry]{
			OperationID: "update-collection-entry",
			Method:      http.MethodPut,
			Path:        "/collections/{id}/entries/{cardId}",
			Summary:     "Set the quantity of a Card already in one of the current user's own Collections",
			Tags:        []string{"inventory"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.update,
		}),
		endpoint.New(endpoint.Endpoint[bulkUpdateEntriesInput, []dto.InventoryChangeResult]{
			OperationID: "bulk-update-collection-entries",
			Method:      http.MethodPatch,
			Path:        "/collections/{id}/entries",
			Summary:     "Set many Card quantities in one of the current user's own Collections; items over capacity or naming an unknown Card are declined, the rest applied",
			Tags:        []string{"inventory"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.bulkUpdate,
		}),
		endpoint.NewNoBody(endpoint.NoBodyEndpoint[entryInput]{
			OperationID: "remove-collection-entry",
			Method:      http.MethodDelete,
			Path:        "/collections/{id}/entries/{cardId}",
			Summary:     "Remove a Card from one of the current user's own Collections",
			Tags:        []string{"inventory"},
			Secured:     true,
			HandlerFunc: h.remove,
		}),
	}
}
