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
