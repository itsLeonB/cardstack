package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
)

type InventoryHandler struct {
	inventorySvc service.InventoryService
}

func NewInventoryHandler(inventorySvc service.InventoryService) *InventoryHandler {
	return &InventoryHandler{inventorySvc: inventorySvc}
}

type collectionEntriesInput struct {
	CollectionID uuid.UUID `path:"id" doc:"Collection ID"`
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

func (h *InventoryHandler) list(ctx context.Context, in collectionEntriesInput) ([]dto.InventoryItem, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return nil, err
	}

	return h.inventorySvc.List(ctx, profileID, in.CollectionID)
}

func (h *InventoryHandler) add(ctx context.Context, in addEntryInput) (dto.InventoryEntry, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.InventoryEntry{}, err
	}

	return h.inventorySvc.Add(ctx, profileID, in.CollectionID, dto.InventoryEntryRequest{CardID: in.Body.CardID, Quantity: in.Body.Quantity})
}

func (h *InventoryHandler) update(ctx context.Context, in updateEntryInput) (dto.InventoryEntry, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.InventoryEntry{}, err
	}

	return h.inventorySvc.UpdateQuantity(ctx, profileID, in.CollectionID, in.CardID, in.Body.Quantity)
}

func (h *InventoryHandler) remove(ctx context.Context, in entryInput) error {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return err
	}

	return h.inventorySvc.Remove(ctx, profileID, in.CollectionID, in.CardID)
}

// Routes sets Secured:true only as OpenAPI metadata; the router must pass
// SessionGuard to RegisterAll.
func (h *InventoryHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.NewList(endpoint.ListEndpoint[collectionEntriesInput, dto.InventoryItem]{
			OperationID: "list-collection-entries",
			Method:      http.MethodGet,
			Path:        "/collections/{id}/entries",
			Summary:     "List the Cards and quantities in one of the current user's own Collections",
			Tags:        []string{"inventory"},
			Secured:     true,
			HandlerFunc: h.list,
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
