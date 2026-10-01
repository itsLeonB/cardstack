package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// InventoryRepository is crud.Repository[entity.InventoryEntry] plus the one
// aggregate go-crud has no equivalent for.
type InventoryRepository interface {
	crud.Repository[entity.InventoryEntry]
	// SumQuantity returns the Collection's summed quantity (0 when empty).
	// It runs in the transaction carried by ctx, if any.
	SumQuantity(ctx context.Context, collectionID uuid.UUID) (int, error)
	// ListHoldings returns the profile's Collections that hold the Card, with
	// the Card's quantity in each, ordered by Collection title. The ownership
	// filter is part of the query.
	ListHoldings(ctx context.Context, profileID, cardID uuid.UUID) ([]CardHolding, error)
}

// CardHolding is one Collection's holding of a Card.
type CardHolding struct {
	CollectionID    uuid.UUID
	CollectionTitle string
	Quantity        int
}

type inventoryRepository struct {
	crud.Repository[entity.InventoryEntry]
}

func NewInventoryRepository(base crud.Repository[entity.InventoryEntry]) InventoryRepository {
	return &inventoryRepository{Repository: base}
}

func (r *inventoryRepository) SumQuantity(ctx context.Context, collectionID uuid.UUID) (int, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return 0, err
	}

	var sum int
	err = db.Model(&entity.InventoryEntry{}).
		Where("collection_id = ?", collectionID).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&sum).
		Error
	if err != nil {
		return 0, ungerr.Wrap(err, "summing inventory quantity")
	}

	return sum, nil
}

func (r *inventoryRepository) ListHoldings(ctx context.Context, profileID, cardID uuid.UUID) ([]CardHolding, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var holdings []CardHolding
	err = db.Table("inventory_entries AS ie").
		Select("c.id AS collection_id, c.title AS collection_title, ie.quantity").
		Joins("JOIN collections c ON c.id = ie.collection_id").
		Where("c.profile_id = ? AND ie.card_id = ? AND ie.quantity > 0", profileID, cardID).
		Order("c.title, c.id").
		Scan(&holdings).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "listing card holdings")
	}

	return holdings, nil
}
