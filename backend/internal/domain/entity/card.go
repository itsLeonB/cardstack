package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
	"gorm.io/datatypes"
)

// Card is a specific printed card design within exactly one ExpansionSet
// (see CONTEXT.md); number and rarity are scoped to that set, not shared
// across regions. LocalID is the upstream source's own per-set card number
// (e.g. pokemon-card.com's collector number "001" out of "001/126") —
// stored as text since it's zero-padded and some sets carry non-numeric
// IDs (promos). Category is the card's single closed-vocabulary type
// (Pokémon/Trainer/Energi, per docs/adr/0007); Tags is a multi-valued,
// game-specific tag list (e.g. Trainer/Energy subtype) also per ADR-0007.
// RarityID references the rarities lookup table (docs/adr/0009) rather than
// a bare code column. Attributes holds game-specific data (HP, types,
// attacks, ...) per docs/adr/0001. Raw is the full upstream response as-is,
// kept so a future need for another field doesn't require re-ingesting
// historical cards. It's TEXT, not JSONB: Postgres's JSONB type re-
// serializes on write (reformats whitespace, normalizes numbers, drops
// duplicate keys), so it can't actually guarantee the stored bytes match
// what the upstream source sent — TEXT stores exactly what it's given.
// Nothing queries into Raw's structure (that's what Attributes is for), so
// JSONB's query operators aren't needed here.
type Card struct {
	crud.BaseEntity
	ExpansionSetID uuid.UUID                   `gorm:"type:uuid;not null;index"`
	LocalID        string                      `gorm:"not null"`
	Name           string                      `gorm:"not null"`
	Category       string                      `gorm:"not null;index"`
	Illustrator    string                      `gorm:"not null;default:''"`
	Tags           datatypes.JSONSlice[string] `gorm:"not null;type:jsonb"`
	RarityID       uuid.UUID                   `gorm:"type:uuid;not null;index"`
	ImageURL       string                      `gorm:"not null;default:''"`
	Attributes     datatypes.JSONMap           `gorm:"not null"`
	Raw            string                      `gorm:"not null;default:''"`
}

func (Card) TableName() string { return "cards" }
