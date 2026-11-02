package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// Wishlist is a profile's own Wishlist state (GLOSSARY.md: Wishlist). One row
// per profile. Its items live in wishlist_items. Shared is off by default; when
// true, the Wishlist is reachable at /s/<profile slug>/wishlist (ADR-0025).
type Wishlist struct {
	crud.BaseEntity
	ProfileID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Shared    bool      `gorm:"not null;default:false"`
}

func (Wishlist) TableName() string { return "wishlists" }
