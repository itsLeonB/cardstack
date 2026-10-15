package service

import (
	"context"

	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/ungerr"
)

const emptySourceMsg = "the source Slot is empty; only a Slot holding a card can be moved"

// MoveSlot moves the card in one Slot to another Slot of the same binder. An
// empty target takes the card and a filled target swaps with it. Neither moves
// a card between Cards, so no Inventory Entry changes.
func (s *binderService) MoveSlot(ctx context.Context, req dto.BinderSlotMove) error {
	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}

		layout, err := s.layouts.GetLayout(ctx, c.ID)
		if err != nil {
			return err
		}

		from, filled, err := s.slotAt(ctx, c, layout, req.From)
		if err != nil {
			return err
		}
		if !filled {
			return ungerr.BadRequestError(emptySourceMsg)
		}

		to, toFilled, err := s.slotAt(ctx, c, layout, req.To)
		if err != nil {
			return err
		}
		if req.From == req.To {
			return nil
		}

		if !toFilled {
			// The row keeps its card and takes the empty Slot's position. The
			// unique key on the position cannot clash, since the target is empty.
			from.LeafID, from.Side, from.Row, from.Column = to.LeafID, to.Side, to.Row, to.Column
			_, err = s.slots.Update(ctx, from)
			return err
		}

		// Both Slots are filled, so the two rows exchange what they hold and keep
		// their positions. Card and Unidentified swap together, so each row still
		// holds exactly one of them.
		from.CardID, to.CardID = to.CardID, from.CardID
		from.Unidentified, to.Unidentified = to.Unidentified, from.Unidentified
		if _, err := s.slots.Update(ctx, from); err != nil {
			return err
		}
		_, err = s.slots.Update(ctx, to)
		return err
	})
}

// slotAt locates the Slot at target. It returns the Slot's row when occupied,
// and otherwise a new row carrying only its position, so the caller can place a
// card there.
func (s *binderService) slotAt(ctx context.Context, c entity.Collection, layout entity.CollectionLayout, target dto.BinderSlotTarget) (entity.BinderSlot, bool, error) {
	if err := checkSlotInGrid(layout, target); err != nil {
		return entity.BinderSlot{}, false, err
	}

	leaf, side, err := s.pageLeaf(ctx, c.ID, target.Page)
	if err != nil {
		return entity.BinderSlot{}, false, err
	}

	slot, err := s.slots.FindFirst(ctx, slotSpec(c, leaf, side, target.Row, target.Column))
	if err != nil {
		return entity.BinderSlot{}, false, err
	}
	if !slot.IsZero() {
		return slot, true, nil
	}

	return entity.BinderSlot{CollectionID: c.ID, LeafID: leaf.ID, Side: side, Row: target.Row, Column: target.Column}, false, nil
}
