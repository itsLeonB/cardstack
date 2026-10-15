package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// BinderSlotRepository reads a binder's occupied Slots. An empty Slot has no row.
type BinderSlotRepository interface {
	crud.Repository[entity.BinderSlot]
	// ListSlotsOnPage returns the occupied Slots of one face of a Leaf, with
	// their Cards, Rarities and Expansion Sets loaded.
	ListSlotsOnPage(ctx context.Context, leafID uuid.UUID, side string) ([]entity.BinderSlot, error)
	HasSlotsOnPage(ctx context.Context, leafID uuid.UUID, side string) (bool, error)
	// CountSlots returns, across the whole binder, how many Slots hold a Card and
	// how many are Unidentified. It is raw GORM because go-crud has no aggregate.
	CountSlots(ctx context.Context, collectionID uuid.UUID) (cards, unidentified int, err error)
}

type binderSlotRepository struct {
	crud.Repository[entity.BinderSlot]
}

func NewBinderSlotRepository(base crud.Repository[entity.BinderSlot]) BinderSlotRepository {
	return &binderSlotRepository{Repository: base}
}

// pageSpec scopes a query to one face of a Leaf. Both fields must be set, or
// the Model would match every Page.
func pageSpec(leafID uuid.UUID, side string) crud.Specification[entity.BinderSlot] {
	return crud.Specification[entity.BinderSlot]{
		Model: entity.BinderSlot{LeafID: leafID, Side: side},
	}
}

func (r *binderSlotRepository) ListSlotsOnPage(ctx context.Context, leafID uuid.UUID, side string) ([]entity.BinderSlot, error) {
	if leafID == uuid.Nil || side == "" {
		return nil, nil
	}

	spec := pageSpec(leafID, side)
	spec.PreloadRelations = []string{"Card", "Card.Rarity", "Card.ExpansionSet"}
	return r.FindAll(ctx, spec)
}

func (r *binderSlotRepository) HasSlotsOnPage(ctx context.Context, leafID uuid.UUID, side string) (bool, error) {
	if leafID == uuid.Nil || side == "" {
		return false, nil
	}

	slot, err := r.FindFirst(ctx, pageSpec(leafID, side))
	if err != nil {
		return false, err
	}

	return !slot.IsZero(), nil
}

func (r *binderSlotRepository) CountSlots(ctx context.Context, collectionID uuid.UUID) (int, int, error) {
	if collectionID == uuid.Nil {
		return 0, 0, nil
	}

	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return 0, 0, err
	}

	var counts struct {
		Cards        int64
		Unidentified int64
	}
	err = db.Model(&entity.BinderSlot{}).
		Select("COUNT(card_id) AS cards, COUNT(*) FILTER (WHERE unidentified) AS unidentified").
		Where("collection_id = ?", collectionID).
		Scan(&counts).
		Error
	if err != nil {
		return 0, 0, ungerr.Wrap(err, "counting binder slots")
	}

	return int(counts.Cards), int(counts.Unidentified), nil
}
