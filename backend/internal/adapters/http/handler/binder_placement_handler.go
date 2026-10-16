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

type BinderPlacementHandler struct {
	slotSvc service.BinderSlotService
}

func NewBinderPlacementHandler(slotSvc service.BinderSlotService) *BinderPlacementHandler {
	return &BinderPlacementHandler{slotSvc: slotSvc}
}

type placeCardsItem struct {
	CardID   uuid.UUID `json:"cardId" required:"true" doc:"The Card to place. Must be unique within the request (400 otherwise)."`
	Quantity int       `json:"quantity" required:"true" minimum:"1" maximum:"2147483647" doc:"Copies to place, one empty Slot each."`
}

type placeCardsInput struct {
	CollectionID uuid.UUID `path:"id" doc:"Collection ID"`
	Body         struct {
		Items []placeCardsItem `json:"items" required:"true" minItems:"1" maxItems:"100" doc:"Cards to place, in order, at most 100."`
	}
}

func (h *BinderPlacementHandler) placeCards(ctx context.Context, in placeCardsInput) ([]dto.BinderPlacementResult, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return nil, err
	}

	items := ezutil.MapSlice(in.Body.Items, func(it placeCardsItem) dto.BinderPlacementItem {
		return dto.BinderPlacementItem{CardID: it.CardID, Quantity: it.Quantity}
	})
	return h.slotSvc.PlaceCards(ctx, dto.BinderPlacementRequest{
		ProfileID:    profileID,
		CollectionID: in.CollectionID,
		Items:        items,
	})
}

func (h *BinderPlacementHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[placeCardsInput, []dto.BinderPlacementResult]{
			OperationID: "place-binder-cards",
			Method:      http.MethodPost,
			Path:        "/collections/{id}/binder/placements",
			Summary:     "Place Cards in a binder's first empty Slots in Page order, one Slot per copy; copies with no empty Slot are declined",
			Tags:        []string{"binder"},
			Secured:     true,
			HandlerFunc: h.placeCards,
		}),
	}
}
