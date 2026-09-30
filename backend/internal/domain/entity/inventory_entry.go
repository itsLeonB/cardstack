package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// InventoryEntry is the quantity (always > 0) of one Card held in one
// Collection; (CollectionID, CardID) is unique. Removing a Card deletes the
// row rather than storing 0.
type InventoryEntry struct {
	crud.BaseEntity
	CollectionID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_inventory_entries_collection_card"`
	CardID       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_inventory_entries_collection_card"`
	Quantity     int       `gorm:"not null"`
}

func (InventoryEntry) TableName() string { return "inventory_entries" }
