package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// CardVariant is a specific print finish of a Card (CONTEXT.md) — the unit
// quantities are tracked against, not the Card itself. Finish is free text
// with a DB CHECK constraint (see the catalog_schema migration) rather than
// a Postgres ENUM, so adding a new finish later is a one-line migration.
type CardVariant struct {
	crud.BaseEntity
	CardID uuid.UUID `gorm:"type:uuid;not null;index"`
	Finish string    `gorm:"not null"`
}

func (CardVariant) TableName() string { return "card_variants" }
