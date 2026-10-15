package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// writeSlotRow stores one Slot's state and returns the Card the Slot held before,
// if any. It leaves Inventory Entries alone: the caller passes every Card it
// touched to reconcileCards once its writes are done, so a Page write reconciles
// each Card once.
func (s *binderService) writeSlotRow(ctx context.Context, c entity.Collection, leaf entity.BinderLeaf, side string, target dto.BinderSlotTarget, state string, cardID *uuid.UUID) (*uuid.UUID, error) {
	slot, err := s.slots.FindFirst(ctx, slotSpec(c, leaf, side, target.Row, target.Column))
	if err != nil {
		return nil, err
	}

	// An empty Slot has no row, so a zero slot means the Slot was empty before this write.
	var previous *uuid.UUID
	if !slot.IsZero() {
		previous = slot.CardID
	}

	switch {
	case state == dto.BinderSlotEmpty:
		if !slot.IsZero() {
			err = s.slots.Delete(ctx, slot)
		}
	case slot.IsZero():
		_, err = s.slots.Insert(ctx, entity.BinderSlot{
			CollectionID: c.ID,
			LeafID:       leaf.ID,
			Side:         side,
			Row:          target.Row,
			Column:       target.Column,
			CardID:       cardID,
			Unidentified: state == dto.BinderSlotUnidentified,
		})
	default:
		slot.CardID = cardID
		slot.Unidentified = state == dto.BinderSlotUnidentified
		_, err = s.slots.Update(ctx, slot)
	}
	if err != nil {
		return nil, err
	}

	return previous, nil
}

// reconcileCards brings the Inventory Entries of the given Cards into step with
// their Slot counts, once per distinct Card. Nil ids are skipped. The caller
// holds the Collection's row lock and runs this inside its transaction.
func (s *binderService) reconcileCards(ctx context.Context, collectionID uuid.UUID, cardIDs []*uuid.UUID) error {
	seen := make(map[uuid.UUID]bool, len(cardIDs))
	for _, id := range cardIDs {
		if id == nil || seen[*id] {
			continue
		}
		seen[*id] = true
		if err := s.reconcileEntry(ctx, collectionID, *id); err != nil {
			return err
		}
	}

	return nil
}

// checkSlotInGrid refuses a Slot address outside the Layout's rows and columns.
func checkSlotInGrid(layout entity.CollectionLayout, target dto.BinderSlotTarget) error {
	if target.Row < 1 || target.Row > layout.Rows || target.Column < 1 || target.Column > layout.Columns {
		return ungerr.BadRequestError(slotOutOfGridMsg)
	}

	return nil
}

// slotSpec addresses one Slot of a Page. Every field must be set, because a zero
// field drops out of the query and would widen it.
func slotSpec(c entity.Collection, leaf entity.BinderLeaf, side string, row, column int) crud.Specification[entity.BinderSlot] {
	return crud.Specification[entity.BinderSlot]{
		Model: entity.BinderSlot{CollectionID: c.ID, LeafID: leaf.ID, Side: side, Row: row, Column: column},
	}
}
