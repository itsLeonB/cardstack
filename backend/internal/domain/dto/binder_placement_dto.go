package dto

import "github.com/google/uuid"

const (
	BinderPlacementReasonNotEnoughEmptySlots = "not_enough_empty_slots"
)

// BinderPlacementItem is one Card and the copies to place, one empty Slot each.
type BinderPlacementItem struct {
	CardID   uuid.UUID
	Quantity int
}

// BinderPlacementRequest places Items, in order, into a binder's first empty Slots.
type BinderPlacementRequest struct {
	ProfileID    uuid.UUID
	CollectionID uuid.UUID
	Items        []BinderPlacementItem
}

// BinderPlacementResult is the outcome of one item: Placed copies took Slots and
// Declined copies did not. Reason and Message are set only when Declined is above zero.
type BinderPlacementResult struct {
	CardID   uuid.UUID `json:"cardId"`
	Placed   int       `json:"placed"`
	Declined int       `json:"declined"`
	Reason   string    `json:"reason,omitempty" enum:"not_enough_empty_slots,card_not_found" doc:"Why copies were declined."`
	Message  string    `json:"message,omitempty" doc:"Human-readable reason for declined copies."`
}
