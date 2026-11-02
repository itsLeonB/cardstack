package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// WishlistItem is one Card a profile wants, with the quantity wanted
// (GLOSSARY.md: Wishlist). Unique per profile and Card. It never contributes to
// Master Inventory: owned quantity is read from inventory_entries, never here.
type WishlistItem struct {
	crud.BaseEntity
	ProfileID      uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_wishlist_items_profile_card"`
	CardID         uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_wishlist_items_profile_card"`
	WantedQuantity int       `gorm:"not null"`

	// Card is loaded only via crud.Specification.PreloadRelations.
	Card Card `gorm:"foreignKey:CardID"`
}

func (WishlistItem) TableName() string { return "wishlist_items" }
