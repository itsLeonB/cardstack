package dto

import "github.com/google/uuid"

// InventoryListRequest lists the contents of one of a profile's Collections.
type InventoryListRequest struct {
	ProfileID    uuid.UUID
	CollectionID uuid.UUID
}

// InventoryEntryRequest sets a Card's quantity in a Collection: adds the Card
// (Add) or replaces its quantity (UpdateQuantity).
type InventoryEntryRequest struct {
	ProfileID    uuid.UUID
	CollectionID uuid.UUID
	CardID       uuid.UUID
	Quantity     int
}

// InventoryEntryLookup addresses one Card's entry in a Collection.
type InventoryEntryLookup struct {
	ProfileID    uuid.UUID
	CollectionID uuid.UUID
	CardID       uuid.UUID
}

// InventoryEntry is a written Inventory Entry: a Card ID and its quantity.
type InventoryEntry struct {
	CardID   uuid.UUID `json:"cardId"`
	Quantity int       `json:"quantity"`
}

// InventoryItem is one Card + quantity pair of a Collection's contents.
type InventoryItem struct {
	Card     CardSummary `json:"card"`
	Quantity int         `json:"quantity"`
}
