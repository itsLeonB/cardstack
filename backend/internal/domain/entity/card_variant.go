package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// CardVariant is a specific print finish of a Card (CONTEXT.md) — the unit
// quantities are tracked against, not the Card itself. FinishID references
// the finishes lookup table (normalized out of a CHECK-constrained free-text
// column), so a new finish is a row insert, not a migration.
type CardVariant struct {
	crud.BaseEntity
	CardID   uuid.UUID `gorm:"type:uuid;not null;index"`
	FinishID uuid.UUID `gorm:"type:uuid;not null;index"`
}

func (CardVariant) TableName() string { return "card_variants" }
