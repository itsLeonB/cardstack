package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
)

// writeSlotRow stores one Slot's state, then brings the Inventory Entries of the
// Card it held and the Card it now holds into step with the Slot count. The
// caller holds the Collection's row lock and runs this inside its transaction.
func (s *binderService) writeSlotRow(ctx context.Context, c entity.Collection, leaf entity.BinderLeaf, side string, target dto.BinderSlotTarget, state string, cardID *uuid.UUID) error {
	slot, err := s.slots.FindFirst(ctx, crud.Specification[entity.BinderSlot]{
		Model: entity.BinderSlot{CollectionID: c.ID, LeafID: leaf.ID, Side: side, Row: target.Row, Column: target.Column},
	})
	if err != nil {
		return err
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
		return err
	}

	for _, id := range []*uuid.UUID{previous, cardID} {
		if id == nil {
			continue
		}
		if err := s.reconcileEntry(ctx, c.ID, *id); err != nil {
			return err
		}
	}

	return nil
}
