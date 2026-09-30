package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// Collection is a profile-owned, named grouping of Cards with quantities (a
// binder, box, or deck; see CONTEXT.md's Collection entry). MaxCardCount is
// an optional hard cap on the collection's summed Inventory Entry
// quantities - nil means no limit. It's only a validation ceiling for later
// Inventory Entry writes (a later ticket); nothing here tracks quantities.
type Collection struct {
	crud.BaseEntity
	ProfileID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Title        string    `gorm:"not null"`
	Description  string    `gorm:"not null;default:''"`
	MaxCardCount *int
}

func (Collection) TableName() string { return "collections" }
