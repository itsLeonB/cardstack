package entity

import (
	"time"

	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// ExpansionSet is a released batch of cards within a Game, scoped to one
// print edition/region (see CONTEXT.md — never assumed shared across
// regions). Code is the upstream source's own set ID (e.g. TCGDex's
// "SV1V"); LocaleID is descriptive source metadata, not part of the natural
// key, which is (GameID, Code). SeriesID optionally groups this set under a
// Series (docs/adr/0008); ReleaseDate is the upstream source's own sale date
// for the set, when known.
type ExpansionSet struct {
	crud.BaseEntity
	GameID      uuid.UUID  `gorm:"type:uuid;not null;index"`
	Code        string     `gorm:"not null"`
	Name        string     `gorm:"not null"`
	LocaleID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	SeriesID    *uuid.UUID `gorm:"type:uuid;index"`
	ReleaseDate *time.Time
}

func (ExpansionSet) TableName() string { return "expansion_sets" }
