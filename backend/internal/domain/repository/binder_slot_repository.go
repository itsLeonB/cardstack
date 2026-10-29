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
	// their Cards, Rarities and Expansion Sets loaded. Without includeUnpublished,
	// a Slot holding a Card of an unpublished Expansion Set is left out, so it
	// reads as empty.
	ListSlotsOnPage(ctx context.Context, leafID uuid.UUID, side string, includeUnpublished bool) ([]entity.BinderSlot, error)
	HasSlotsOnPage(ctx context.Context, leafID uuid.UUID, side string) (bool, error)
	// CountSlots returns, across the whole binder, how many Slots hold a Card and
	// how many are Unidentified. Without includeUnpublished, Slots holding a Card
	// of an unpublished Expansion Set are not counted. It is raw GORM because
	// go-crud has no aggregate.
	CountSlots(ctx context.Context, collectionID uuid.UUID, includeUnpublished bool) (cards, unidentified int, err error)
	// CountCardSlots returns how many Slots of one Collection hold the Card. It is
	// raw GORM because go-crud has no aggregate.
	CountCardSlots(ctx context.Context, collectionID, cardID uuid.UUID) (int, error)
	// ListOccupiedSlots returns the Slots of one Collection that hold one of the
	// Cards, with their Leaves loaded so each Slot's Page can be computed. Other
	// Slots, Unidentified ones included, are left out, so an entries page does not
	// load the whole binder.
	ListOccupiedSlots(ctx context.Context, collectionID uuid.UUID, cardIDs []uuid.UUID) ([]entity.BinderSlot, error)
	// ListAllOccupiedSlots returns every occupied Slot of one Collection, Unidentified
	// ones included, with their Leaves loaded so each Slot's Page can be computed.
	ListAllOccupiedSlots(ctx context.Context, collectionID uuid.UUID) ([]entity.BinderSlot, error)
	// ListSlotsAt returns the occupied Slots among addresses, so a write loads the
	// Slots it touches and no others. An empty address has no row and is absent.
	ListSlotsAt(ctx context.Context, collectionID uuid.UUID, addresses []BinderSlotAddress) ([]entity.BinderSlot, error)
}

// BinderSlotAddress is where a Slot sits: its Leaf, face, row and column.
type BinderSlotAddress struct {
	LeafID uuid.UUID
	Side   string
	Row    int
	Column int
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

func (r *binderSlotRepository) ListSlotsOnPage(ctx context.Context, leafID uuid.UUID, side string, includeUnpublished bool) ([]entity.BinderSlot, error) {
	if leafID == uuid.Nil || side == "" {
		return nil, nil
	}

	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	// go-crud's Specification matches Model fields by equality only, so the
	// visibility condition, an OR over a subquery, is written as raw GORM.
	query := db.Preload("Card").Preload("Card.Rarity").Preload("Card.ExpansionSet").
		Where("leaf_id = ? AND side = ?", leafID, side)
	if !includeUnpublished {
		query = query.Where("card_id IS NULL OR card_id IN (?)", visibleCardIDs(db))
	}

	var slots []entity.BinderSlot
	err = query.Find(&slots).Error
	if err != nil {
		return nil, ungerr.Wrap(err, "listing Slots on a Page")
	}

	return slots, nil
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

func (r *binderSlotRepository) CountSlots(ctx context.Context, collectionID uuid.UUID, includeUnpublished bool) (int, int, error) {
	if collectionID == uuid.Nil {
		return 0, 0, nil
	}

	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return 0, 0, err
	}

	query := db.Model(&entity.BinderSlot{}).
		Select("COUNT(card_id) AS cards, COUNT(*) FILTER (WHERE unidentified) AS unidentified").
		Where("collection_id = ?", collectionID)
	if !includeUnpublished {
		query = query.Where("card_id IS NULL OR card_id IN (?)", visibleCardIDs(db))
	}

	var counts struct {
		Cards        int64
		Unidentified int64
	}
	err = query.Scan(&counts).Error
	if err != nil {
		return 0, 0, ungerr.Wrap(err, "counting binder slots")
	}

	return int(counts.Cards), int(counts.Unidentified), nil
}

func (r *binderSlotRepository) CountCardSlots(ctx context.Context, collectionID, cardID uuid.UUID) (int, error) {
	if collectionID == uuid.Nil || cardID == uuid.Nil {
		return 0, nil
	}

	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return 0, err
	}

	var count int64
	err = db.Model(&entity.BinderSlot{}).
		Where("collection_id = ? AND card_id = ?", collectionID, cardID).
		Count(&count).
		Error
	if err != nil {
		return 0, ungerr.Wrap(err, "counting Slots holding a Card")
	}

	return int(count), nil
}

func (r *binderSlotRepository) ListOccupiedSlots(ctx context.Context, collectionID uuid.UUID, cardIDs []uuid.UUID) ([]entity.BinderSlot, error) {
	if collectionID == uuid.Nil || len(cardIDs) == 0 {
		return nil, nil
	}

	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var slots []entity.BinderSlot
	err = db.Preload("Leaf").
		Where("collection_id = ? AND card_id IN ?", collectionID, cardIDs).
		Find(&slots).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "listing Slots holding Cards")
	}

	return slots, nil
}

func (r *binderSlotRepository) ListAllOccupiedSlots(ctx context.Context, collectionID uuid.UUID) ([]entity.BinderSlot, error) {
	if collectionID == uuid.Nil {
		return nil, nil
	}

	return r.FindAll(ctx, crud.Specification[entity.BinderSlot]{
		Model:            entity.BinderSlot{CollectionID: collectionID},
		PreloadRelations: []string{"Leaf"},
	})
}

func (r *binderSlotRepository) ListSlotsAt(ctx context.Context, collectionID uuid.UUID, addresses []BinderSlotAddress) ([]entity.BinderSlot, error) {
	if collectionID == uuid.Nil || len(addresses) == 0 {
		return nil, nil
	}

	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	// go-crud has no tuple match, so the address is matched with a row-value IN.
	keys := make([][]any, 0, len(addresses))
	for _, address := range addresses {
		keys = append(keys, []any{address.LeafID, address.Side, address.Row, address.Column})
	}

	var slots []entity.BinderSlot
	err = db.Where("collection_id = ?", collectionID).
		Where("(leaf_id, side, slot_row, slot_column) IN ?", keys).
		Find(&slots).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "listing Slots at addresses")
	}

	return slots, nil
}
