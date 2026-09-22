package entity

import (
	crud "github.com/itsLeonB/go-crud"
)

// Finish is a shared reference row for a card print finish (e.g. "holo",
// "reverse"). Normalized out of card_variants.finish's CHECK constraint into
// a lookup table so a new finish is a row insert, not a migration.
type Finish struct {
	crud.BaseEntity
	Code string `gorm:"uniqueIndex;not null"`
}

func (Finish) TableName() string { return "finishes" }
