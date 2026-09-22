package entity

import (
	crud "github.com/itsLeonB/go-crud"
)

// Locale is a shared reference row for an upstream source's locale code
// (e.g. "id", "ja"). Normalized out of expansion_sets.locale so it's a
// lookup FK rather than repeated free text.
type Locale struct {
	crud.BaseEntity
	Code string `gorm:"uniqueIndex;not null"`
}

func (Locale) TableName() string { return "locales" }
