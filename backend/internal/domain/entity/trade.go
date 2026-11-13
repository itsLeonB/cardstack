package entity

import (
	"time"

	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// TradeStatus is a Trade's state (GLOSSARY.md: Trade), the single source of
// truth held in one column.
type TradeStatus string

const (
	TradeStatusOpen        TradeStatus = "open"
	TradeStatusNegotiating TradeStatus = "negotiating"
	TradeStatusLocked      TradeStatus = "locked"
	TradeStatusExecuted    TradeStatus = "executed"
	TradeStatusCancelled   TradeStatus = "cancelled"
)

// A Trade Item's Side names the seat whose Collection the item leaves.
const (
	SideOwner       = 0
	SideParticipant = 1
)

// Trade is a swap agreed between two profiles (GLOSSARY.md: Trade). The
// creator is OwnerProfileID; ParticipantProfileID is the second seat,
// claimed atomically by the first join (docs/adr/0028). Slug is the link's
// address, permanent for the Trade's life; a new Trade draws a new one.
type Trade struct {
	crud.BaseEntity
	Slug                 string      `gorm:"not null;uniqueIndex"`
	OwnerProfileID       uuid.UUID   `gorm:"type:uuid;not null;index"`
	ParticipantProfileID *uuid.UUID  `gorm:"type:uuid;index"`
	Status               TradeStatus `gorm:"type:text;not null;default:'open'"`
	// OwnerAcceptedAt and ParticipantAcceptedAt record each seat's
	// acceptance (spec: two nullable acceptance timestamps beside the
	// status column); the second one moves Status to locked.
	OwnerAcceptedAt       *time.Time `gorm:"type:timestamptz"`
	ParticipantAcceptedAt *time.Time `gorm:"type:timestamptz"`

	// TradeItems are loaded only via crud.Specification.PreloadRelations.
	TradeItems []TradeItem `gorm:"foreignKey:TradeID"`
}

func (Trade) TableName() string { return "trades" }

// AcceptedAt is the seat's acceptance timestamp, nil until that side
// accepts. Side is entity.SideOwner or entity.SideParticipant.
func (t Trade) AcceptedAt(side int) *time.Time {
	if side == SideOwner {
		return t.OwnerAcceptedAt
	}
	return t.ParticipantAcceptedAt
}

// TradeItem is one entry on a Trade side (GLOSSARY.md: Trade Item): a Card, a
// quantity, and the Collection it is meant to leave. Side names the seat:
// SideOwner's offer, SideParticipant's. SourceCollectionID goes NULL when its
// Collection is deleted — the item is part of the agreed deal and outlives it
// (the spec's execution falls back to the participant's other Collections).
// Postgres unique indexes treat NULLs as distinct, so the
// (trade, side, card, source) uniqueness only bites while a source exists.
type TradeItem struct {
	crud.BaseEntity
	TradeID            uuid.UUID  `gorm:"type:uuid;not null;index;uniqueIndex:idx_trade_items_trade_side_card_source"`
	Side               int        `gorm:"not null;uniqueIndex:idx_trade_items_trade_side_card_source"`
	CardID             uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_trade_items_trade_side_card_source"`
	SourceCollectionID *uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_trade_items_trade_side_card_source"`
	Quantity           int        `gorm:"not null"`

	// Card is loaded only via crud.Specification.PreloadRelations.
	Card Card `gorm:"foreignKey:CardID"`
	// SourceCollection is loaded only via crud.Specification.PreloadRelations;
	// it stays the zero value when SourceCollectionID is NULL.
	SourceCollection Collection `gorm:"foreignKey:SourceCollectionID"`
}

func (TradeItem) TableName() string { return "trade_items" }
