package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// Rarity is a per-Game lookup row for an upstream source's rarity code (e.g.
// "SSR", "MUR") paired with a readable name — a raw code alone doesn't
// "curate" (see docs/adr/0009). Rows are find-or-created during ingestion,
// not seeded: the code roster rotates per print era without any code ever
// changing meaning. The natural key is (GameID, Code).
type Rarity struct {
	crud.BaseEntity
	GameID uuid.UUID `gorm:"type:uuid;not null;index"`
	Code   string    `gorm:"not null"`
	Name   string    `gorm:"not null"`
}

func (Rarity) TableName() string { return "rarities" }
