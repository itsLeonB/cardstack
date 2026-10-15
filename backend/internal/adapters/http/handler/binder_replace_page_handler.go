package handler

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
)

type binderPageReplaceInput struct {
	ID   uuid.UUID `path:"id" doc:"Collection ID"`
	Page int       `path:"page" minimum:"1" doc:"Page number in turning order, counted from 1."`
	Body binderPageReplaceBody
}

// The maxItems cap is the largest Layout, 6 × 6 Slots (dto.BinderMaxSlotsPerPage).
// A struct tag cannot read a constant, so binder_replace_page_handler_test.go
// fails if the two drift apart.
type binderPageReplaceBody struct {
	Slots []binderPageSlotBody `json:"slots" required:"true" maxItems:"36" doc:"One state per Slot in grid order: row by row, left to right. The count must match the binder's Layout."`
}

type binderPageSlotBody struct {
	State  string    `json:"state" required:"true" enum:"empty,card,unidentified" doc:"empty: no card; card: holds the Card in cardId; unidentified: holds a card not yet identified."`
	CardID uuid.UUID `json:"cardId,omitempty" doc:"The Card, read only when state is card. An id the server does not know is stored as unidentified and reported for that Slot."`
}

// replacePage returns what each Slot holds after the replace, so the client can
// show the result without a second read.
func (h *BinderHandler) replacePage(ctx context.Context, in binderPageReplaceInput) (dto.BinderPageReplaceResult, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.BinderPageReplaceResult{}, err
	}

	slots := make([]dto.BinderSlotState, len(in.Body.Slots))
	for i, slot := range in.Body.Slots {
		slots[i] = dto.BinderSlotState{State: slot.State, CardID: slot.CardID}
	}

	return h.binderSvc.ReplacePage(ctx, dto.BinderPageReplace{
		ProfileID:    profileID,
		CollectionID: in.ID,
		Page:         in.Page,
		Slots:        slots,
	})
}
