package service

import (
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

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
