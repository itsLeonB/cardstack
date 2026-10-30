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
	// Positions is set only in a binder's entry list: one per Slot holding the Card.
	Positions []BinderSlotTarget `json:"positions,omitempty" doc:"Where the Card sits in a binder: one position per Slot holding it, in Page order. Absent in a plain Collection."`
}

// InventoryQuantityChange is one item of a bulk update: the absolute target
// quantity for a Card (0 removes it).
type InventoryQuantityChange struct {
	CardID   uuid.UUID
	Quantity int
}

// InventoryMoveRequest moves the listed Cards, with every copy of each, from one
// of a profile's Collections into another.
type InventoryMoveRequest struct {
	ProfileID     uuid.UUID
	SourceID      uuid.UUID
	DestinationID uuid.UUID
	// CardIDs and Filter are mutually exclusive: exactly one is set. Filter
	// selects every matching Inventory Entry of the source across all pages.
	CardIDs []uuid.UUID
	Filter  *CardFilter
}

// InventoryMoveResult counts what a Move moved: Cards is the number of
// Inventory Entries, Copies the sum of their quantities.
type InventoryMoveResult struct {
	Cards  int `json:"cards" doc:"Inventory Entries moved, or counted by the count endpoint."`
	Copies int `json:"copies" doc:"Copies in those entries, summed."`
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

// CardHoldingsRequest lists which of a profile's Collections hold a Card.
type CardHoldingsRequest struct {
	ProfileID uuid.UUID
	CardID    uuid.UUID
}

// HoldingCollection identifies the Collection of a CardHolding.
type HoldingCollection struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// CardHolding is the quantity of a Card in one Collection.
type CardHolding struct {
	Collection HoldingCollection `json:"collection"`
	Quantity   int               `json:"quantity"`
}

// MasterInventoryRequest lists or facets a profile's Master Inventory,
// filtered like a catalog search (Page/Limit are ignored by facets).
type MasterInventoryRequest struct {
	ProfileID uuid.UUID
	Filter    CardFilter
}
