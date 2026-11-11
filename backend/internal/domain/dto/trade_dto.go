package dto

import (
	"github.com/google/uuid"
)

// TradeCreateRequest starts a Trade: the caller takes the owner seat.
type TradeCreateRequest struct {
	ProfileID uuid.UUID
}

// TradeSlug is the answer to creating a Trade: the link's address.
type TradeSlug struct {
	Slug string `json:"slug" doc:"The Trade's link address: /trades/{slug}."`
}

// TradeLookup addresses a Trade by its Slug.
type TradeLookup struct {
	Slug string
}

// TradeJoinRequest claims the second seat of a Trade.
type TradeJoinRequest struct {
	ProfileID uuid.UUID
	Slug      string
}

// TradeView is a Trade as a signed-in reader sees it: its state, both seats'
// display names, and the Trade Items on each side. Emails and Auth Identities
// are never part of it (ADR-0028).
type TradeView struct {
	Slug          string          `json:"slug"`
	Status        string          `json:"status" doc:"open, negotiating, locked, executed or cancelled."`
	OwnerName     string          `json:"ownerName" doc:"The owner's display name, never their email."`
	OtherName     *string         `json:"otherName,omitempty" doc:"The second seat's display name once joined; absent while the seat is empty."`
	OwnerItems    []TradeItemView `json:"ownerItems"`
	OtherItems    []TradeItemView `json:"otherItems"`
	IsOwner       bool            `json:"isOwner" doc:"Whether the caller holds the owner seat."`
	IsParticipant bool            `json:"isParticipant" doc:"Whether the caller holds either seat."`
}

// TradeItemView is one Trade Item as the room shows it: the Card, the
// quantity, and the source Collection it is meant to leave.
type TradeItemView struct {
	Card       CardSummary `json:"card"`
	Quantity   int         `json:"quantity"`
	Collection string      `json:"collection" doc:"The title of the Collection the Card is meant to leave."`
}
