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
// and ImageURL stay first-class columns as cross-game concerns.
type Card struct {
	crud.BaseEntity
	ExpansionSetID uuid.UUID         `gorm:"type:uuid;not null;index"`
	LocalID        string            `gorm:"not null"`
	Names          datatypes.JSONMap `gorm:"not null"`
	Rarity         string            `gorm:"not null;default:''"`
	ImageURL       string            `gorm:"not null;default:''"`
	Attributes     datatypes.JSONMap `gorm:"not null"`
}

func (Card) TableName() string { return "cards" }
