package dto

import "github.com/google/uuid"

// BinderSlotMove moves the card in From to To, on any Page of the same binder.
// An empty To takes the card; a filled To swaps the two Slots.
type BinderSlotMove struct {
	ProfileID    uuid.UUID
	CollectionID uuid.UUID
	From         BinderSlotTarget
	To           BinderSlotTarget
}

// BinderSlotMoveResult is the two Pages a move touched, read after the move.
// They are the same Page when both Slots are on one Page.
type BinderSlotMoveResult struct {
	From BinderPage `json:"from" doc:"The Page the card left, read after the move."`
	To   BinderPage `json:"to" doc:"The Page the card went to, read after the move."`
}
