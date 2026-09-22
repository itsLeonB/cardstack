package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
	"gorm.io/datatypes"
)

// Card is a specific printed card design within exactly one ExpansionSet
// (see CONTEXT.md); number and rarity are scoped to that set, not shared
// across regions. LocalID is the upstream source's own per-set card number
// (e.g. TCGDex's localId "008") — stored as text since it's zero-padded and
// some sets carry non-numeric IDs (promos). Names holds a per-locale
// display-name map (e.g. {"id": "...", "ja": "..."}) since no single
// upstream call returns every locale at once. Attributes holds
// game-specific data (HP, types, attacks, ...) per docs/adr/0001; Rarity
// and ImageURL stay first-class columns as cross-game concerns. Raw is the
// full upstream response as-is, kept so a future need for another field
// doesn't require re-ingesting historical cards. It's TEXT, not JSONB:
// Postgres's JSONB type re-serializes on write (reformats whitespace,
// normalizes numbers, drops duplicate keys), so it can't actually guarantee
// the stored bytes match what TCGDex sent — TEXT stores exactly what it's
// given. Nothing queries into Raw's structure (that's what Attributes is
// for), so JSONB's query operators aren't needed here.
type Card struct {
	crud.BaseEntity
	ExpansionSetID uuid.UUID         `gorm:"type:uuid;not null;index"`
	LocalID        string            `gorm:"not null"`
	Names          datatypes.JSONMap `gorm:"not null"`
	Rarity         string            `gorm:"not null;default:''"`
	ImageURL       string            `gorm:"not null;default:''"`
	Attributes     datatypes.JSONMap `gorm:"not null"`
	Raw            string            `gorm:"not null;default:''"`
}

func (Card) TableName() string { return "cards" }
