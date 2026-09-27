package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// Series optionally groups ExpansionSets released under one print era (e.g.
// "Scarlet & Violet") within a Game (docs/adr/0008). Code is locally derived
// (a slugified Series name), not source-provided; the natural key is
// (GameID, Code).
type Series struct {
	crud.BaseEntity
	GameID uuid.UUID `gorm:"type:uuid;not null;index"`
	Code   string    `gorm:"not null"`
	Name   string    `gorm:"not null"`
}

func (Series) TableName() string { return "series" }
