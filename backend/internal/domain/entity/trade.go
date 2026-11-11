package entity

import (
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

	// TradeItems are loaded only via crud.Specification.PreloadRelations.
	TradeItems []TradeItem `gorm:"foreignKey:TradeID"`
}

func (Trade) TableName() string { return "trades" }

// TradeItem is one entry on a Trade side (GLOSSARY.md: Trade Item): a Card, a
// quantity, and the Collection it is meant to leave. Side is 0 for the owner's
// offer, 1 for the participant's.
type TradeItem struct {
	crud.BaseEntity
	TradeID            uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_trade_items_trade_side_card_source"`
	Side               int       `gorm:"not null;uniqueIndex:idx_trade_items_trade_side_card_source"`
	CardID             uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_trade_items_trade_side_card_source"`
	SourceCollectionID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_trade_items_trade_side_card_source"`
	Quantity           int       `gorm:"not null"`

	// Card is loaded only via crud.Specification.PreloadRelations.
	Card Card `gorm:"foreignKey:CardID"`
	// SourceCollection is loaded only via crud.Specification.PreloadRelations.
	SourceCollection Collection `gorm:"foreignKey:SourceCollectionID"`
}

func (TradeItem) TableName() string { return "trade_items" }
