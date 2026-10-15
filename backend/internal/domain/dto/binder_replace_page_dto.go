package dto

import "github.com/google/uuid"

// BinderSlotReasonUnknownCard is why a Slot asked for a Card the server does not
// know is stored Unidentified instead.
const BinderSlotReasonUnknownCard = "unknown_card"

// BinderPageReplace sets every Slot of one Page at once. Slots are in grid order
// (row by row, left to right) and must match the binder's Slot count per Page.
type BinderPageReplace struct {
	ProfileID    uuid.UUID
	CollectionID uuid.UUID
	Page         int
	Slots        []BinderSlotState
}

// BinderSlotState is the state one Slot is asked to hold. CardID is read only
// when State is BinderSlotCard.
type BinderSlotState struct {
	State  string
	CardID uuid.UUID
}

// BinderPageReplaceResult reports what each Slot of the Page holds now, in grid order.
type BinderPageReplaceResult struct {
	Slots []BinderSlotResult `json:"slots" doc:"What each Slot of the Page holds after the replace, in grid order."`
}

// BinderSlotResult is one Slot's stored state. Reason is set only when the Slot
// asked for a Card the server does not know and was stored Unidentified instead.
type BinderSlotResult struct {
	Row    int        `json:"row" doc:"Row of the Slot on its Page, from 1."`
	Column int        `json:"column" doc:"Column of the Slot on its Page, from 1."`
	State  string     `json:"state" enum:"empty,card,unidentified" doc:"The state the Slot was stored in."`
	CardID *uuid.UUID `json:"cardId,omitempty" doc:"The Card stored in the Slot, set only when State is card."`
	Reason string     `json:"reason,omitempty" doc:"Why the Slot was stored differently from the request: unknown_card when the Card id is not known."`
}
