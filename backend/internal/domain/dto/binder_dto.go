package dto

import "github.com/google/uuid"

const (
	BinderSideFront = "front"
	BinderSideBack  = "back"

	BinderSlotEmpty        = "empty"
	BinderSlotCard         = "card"
	BinderSlotUnidentified = "unidentified"

	BinderPlacementBefore = "before"
	BinderPlacementAfter  = "after"
)

// BinderLayout is a binder's shape. Capacity is its Slot count, so it is
// LeafCount × 2 × Rows × Columns.
type BinderLayout struct {
	Rows      int `json:"rows" doc:"Rows of Slots on each Page."`
	Columns   int `json:"columns" doc:"Columns of Slots on each Page."`
	LeafCount int `json:"leafCount" doc:"Number of Leaves; each Leaf has a front and a back Page."`
	Capacity  int `json:"capacity" doc:"Number of Slots in the binder."`
}

// BinderSlot is one pocket on a Page. Card is set only when State is "card".
type BinderSlot struct {
	Row    int          `json:"row" doc:"Row of the pocket on its Page, from 1."`
	Column int          `json:"column" doc:"Column of the pocket on its Page, from 1."`
	State  string       `json:"state" enum:"empty,card,unidentified" doc:"empty: no card; card: holds Card; unidentified: holds a card the app could not identify."`
	Card   *CardSummary `json:"card" doc:"The Card in the pocket, set only when State is card."`
}

// BinderPage is one Page of a binder in turning order, with its Slots in grid
// order (row by row, left to right). CardTotal counts Slots holding a Card plus
// Unidentified Slots, across the whole binder.
type BinderPage struct {
	Page              int          `json:"page" doc:"Page number in turning order, from 1."`
	Leaf              int          `json:"leaf" doc:"Number of the Leaf this Page belongs to, from 1."`
	Side              string       `json:"side" enum:"front,back" doc:"Which face of the Leaf this Page is."`
	Layout            BinderLayout `json:"layout"`
	CardTotal         int          `json:"cardTotal" doc:"Slots holding a Card plus Unidentified Slots, across the whole binder."`
	UnidentifiedCount int          `json:"unidentifiedCount" doc:"Unidentified Slots, across the whole binder."`
	Slots             []BinderSlot `json:"slots"`
}

// BinderLayoutRequest makes a Collection a binder.
type BinderLayoutRequest struct {
	ProfileID    uuid.UUID
	CollectionID uuid.UUID
	Rows         int
	Columns      int
	LeafCount    int
}

// BinderGridRequest changes a binder's rows and columns.
type BinderGridRequest struct {
	ProfileID    uuid.UUID
	CollectionID uuid.UUID
	Rows         int
	Columns      int
}

// BinderLeafRequest adds a Leaf before or after the Leaf numbered Leaf.
type BinderLeafRequest struct {
	ProfileID    uuid.UUID
	CollectionID uuid.UUID
	Leaf         int
	Placement    string
}

// BinderLeafLookup addresses one Leaf of a binder by its number.
type BinderLeafLookup struct {
	ProfileID    uuid.UUID
	CollectionID uuid.UUID
	Leaf         int
}

// BinderPageLookup addresses one Page of a binder by its number.
type BinderPageLookup struct {
	ProfileID    uuid.UUID
	CollectionID uuid.UUID
	Page         int
}

// BinderSlotTarget addresses one Slot by its Page, row and column. It is also the
// JSON shape of a Card's position in a binder's entry list.
type BinderSlotTarget struct {
	Page   int `json:"page" doc:"Page number in turning order, from 1."`
	Row    int `json:"row" doc:"Row of the pocket on its Page, from 1."`
	Column int `json:"column" doc:"Column of the pocket on its Page, from 1."`
}

// BinderSlotWrite sets one Slot to empty, to a Card (CardID) or to Unidentified.
// State is one of BinderSlotEmpty, BinderSlotCard or BinderSlotUnidentified.
type BinderSlotWrite struct {
	ProfileID    uuid.UUID
	CollectionID uuid.UUID
	Target       BinderSlotTarget
	State        string
	CardID       uuid.UUID
}

// BinderSlotChange is one Slot's new state inside a Page or placement write. State
// is one of the BinderSlot constants, and CardID is read only when it is card.
type BinderSlotChange struct {
	Target BinderSlotTarget
	State  string
	CardID uuid.UUID
}
