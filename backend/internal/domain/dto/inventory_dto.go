package dto

import "github.com/google/uuid"

// InventoryListRequest lists or facets the contents of one of a profile's
// Collections, filtered like a catalog search (Page/Limit are ignored by
// facets).
type InventoryListRequest struct {
	ProfileID    uuid.UUID
	CollectionID uuid.UUID
	Filter       CardFilter
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

// InventoryQuantityChange is one item of a bulk update: the absolute target
// quantity for a Card (0 removes it).
type InventoryQuantityChange struct {
	CardID   uuid.UUID
	Quantity int
}

// InventoryBulkUpdateRequest applies Items, in order, to one Collection.
type InventoryBulkUpdateRequest struct {
	ProfileID    uuid.UUID
	CollectionID uuid.UUID
	Items        []InventoryQuantityChange
}

// InventoryChangeResult is the outcome of one bulk update item. Quantity is
// the Card's resulting quantity (0 when removed or absent); Reason and
// Message are set only when Status is "declined".
type InventoryChangeResult struct {
	CardID   uuid.UUID `json:"cardId"`
	Quantity int       `json:"quantity"`
	Status   string    `json:"status" enum:"applied,removed,declined" doc:"applied: set or created; removed: deleted, or already absent; declined: not applied."`
	Reason   string    `json:"reason,omitempty" enum:"capacity_exceeded,card_not_found" doc:"Why the item was declined."`
	Message  string    `json:"message,omitempty" doc:"Human-readable reason for a declined item."`
}

const (
	InventoryStatusApplied  = "applied"
	InventoryStatusRemoved  = "removed"
	InventoryStatusDeclined = "declined"

	InventoryReasonCapacityExceeded = "capacity_exceeded"
	InventoryReasonCardNotFound     = "card_not_found"
)
