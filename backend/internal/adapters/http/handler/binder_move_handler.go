package handler

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
)

type binderSlotMoveInput struct {
	ID   uuid.UUID `path:"id" doc:"Collection ID"`
	Body binderSlotMoveBody
}

type binderSlotMoveBody struct {
	From binderSlotTargetBody `json:"from" required:"true" doc:"The filled Slot to move the card from."`
	To   binderSlotTargetBody `json:"to" required:"true" doc:"The Slot to move the card to. If it holds a card, the two cards swap."`
}

type binderSlotTargetBody struct {
	Page   int `json:"page" required:"true" minimum:"1" doc:"Page number in turning order, counted from 1."`
	Row    int `json:"row" required:"true" minimum:"1" doc:"Row of the Slot on its Page, counted from 1."`
	Column int `json:"column" required:"true" minimum:"1" doc:"Column of the Slot on its Page, counted from 1."`
}

func (b binderSlotTargetBody) target() dto.BinderSlotTarget {
	return dto.BinderSlotTarget{Page: b.Page, Row: b.Row, Column: b.Column}
}

// moveSlot returns the Pages the move touched, so the client does not need a
// second read for the source Page.
func (h *BinderHandler) moveSlot(ctx context.Context, in binderSlotMoveInput) (dto.BinderSlotMoveResult, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.BinderSlotMoveResult{}, err
	}

	err = h.binderSvc.MoveSlot(ctx, dto.BinderSlotMove{
		ProfileID:    profileID,
		CollectionID: in.ID,
		From:         in.Body.From.target(),
		To:           in.Body.To.target(),
	})
	if err != nil {
		return dto.BinderSlotMoveResult{}, err
	}

	from, err := h.binderSvc.GetPage(ctx, dto.BinderPageLookup{ProfileID: profileID, CollectionID: in.ID, Page: in.Body.From.Page})
	if err != nil {
		return dto.BinderSlotMoveResult{}, err
	}
	to, err := h.binderSvc.GetPage(ctx, dto.BinderPageLookup{ProfileID: profileID, CollectionID: in.ID, Page: in.Body.To.Page})
	if err != nil {
		return dto.BinderSlotMoveResult{}, err
	}

	return dto.BinderSlotMoveResult{From: from, To: to}, nil
}
