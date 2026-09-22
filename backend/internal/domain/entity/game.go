package entity

import (
	crud "github.com/itsLeonB/go-crud"
)

// Game is a distinct trading card game (Pokémon TCG, Riftbound, ...). See
// docs/adr/0001: game-specific attributes attach per-Game via
// Card.Attributes, not as fixed columns here.
type Game struct {
	crud.BaseEntity
	Slug string `gorm:"uniqueIndex;not null"`
	Name string `gorm:"not null"`
}

func (Game) TableName() string { return "games" }
