package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/ungerr"
)

const binderPageSlotCountMsg = "the request needs one state per Slot of the Page, in grid order"

// ReplacePage writes each Slot of the Page through the same path as a single
// Slot write, so every Inventory Entry stays equal to its Slot count. A refusal
// returns before any Slot is written, so the Page is left as it was.
func (s *binderService) ReplacePage(ctx context.Context, req dto.BinderPageReplace) (dto.BinderPageReplaceResult, error) {
	var result dto.BinderPageReplaceResult
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}

		layout, err := s.layouts.GetLayout(ctx, c.ID)
		if err != nil {
			return err
		}
		if len(req.Slots) != layout.Rows*layout.Columns {
			return ungerr.BadRequestError(binderPageSlotCountMsg)
		}

		leaf, side, err := s.pageLeaf(ctx, c.ID, req.Page)
		if err != nil {
			return err
		}

		slots := make([]dto.BinderSlotResult, 0, len(req.Slots))
		touched := make([]*uuid.UUID, 0, 2*len(req.Slots))
		for i, want := range req.Slots {
			target := dto.BinderSlotTarget{Page: req.Page, Row: i/layout.Columns + 1, Column: i%layout.Columns + 1}

			state, cardID, reason, err := s.storedSlotState(ctx, want)
			if err != nil {
				return err
			}

			previous, err := s.writeSlotRow(ctx, c, leaf, side, target, state, cardID)
			if err != nil {
				return err
			}
			touched = append(touched, previous, cardID)

			slots = append(slots, dto.BinderSlotResult{
				Row:    target.Row,
				Column: target.Column,
				State:  state,
				CardID: cardID,
				Reason: reason,
			})
		}

		if err := s.reconcileCards(ctx, c.ID, touched); err != nil {
			return err
		}

		result = dto.BinderPageReplaceResult{Slots: slots}
		return nil
	})
	if err != nil {
		return dto.BinderPageReplaceResult{}, err
	}

	return result, nil
}

// storedSlotState returns the state a requested Slot is stored in. A Card id the
// server does not know is stored Unidentified, with the reason.
func (s *binderService) storedSlotState(ctx context.Context, want dto.BinderSlotState) (string, *uuid.UUID, string, error) {
	if want.State != dto.BinderSlotCard {
		return want.State, nil, "", nil
	}

	_, err := s.resolveCard(ctx, want.CardID)
	if err == nil {
		cardID := want.CardID
		return dto.BinderSlotCard, &cardID, "", nil
	}
	if !isNotFound(err) {
		return "", nil, "", err
	}

	return dto.BinderSlotUnidentified, nil, dto.BinderSlotReasonUnknownCard, nil
}

// isNotFound reports whether err is a not-found AppError, which resolveCard
// returns for a Card it does not know.
func isNotFound(err error) bool {
	var appErr ungerr.AppError
	return errors.As(err, &appErr) && appErr.HttpStatus() == http.StatusNotFound
}
