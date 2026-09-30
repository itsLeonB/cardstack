package dto

import "github.com/google/uuid"

// InventoryEntryRequest adds a Card to a Collection.
type InventoryEntryRequest struct {
	CardID   uuid.UUID
	Quantity int
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
