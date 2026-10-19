package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// CollectionLayout makes a Collection a binder: Rows and Columns are the grid
// of every Page. A Collection without one is a plain box or deck.
type CollectionLayout struct {
	crud.BaseEntity
	CollectionID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Rows         int       `gorm:"column:rows_per_page;not null"`
	Columns      int       `gorm:"column:columns_per_page;not null"`
}

func (CollectionLayout) TableName() string { return "collection_layouts" }

// MaxBinderGridSide is the most rows or columns a Page can have. Layouts are
// validated against it, and a Replace Page body caps its Slot list at
// MaxBinderSlotsPerPage.
const MaxBinderGridSide = 6

// MaxBinderSlotsPerPage is the Slot count of the largest Page.
const MaxBinderSlotsPerPage = MaxBinderGridSide * MaxBinderGridSide

// SlotsPerPage is the Slot count of one Page of this Layout.
func (l CollectionLayout) SlotsPerPage() int { return l.Rows * l.Columns }

// BinderLeaf is one physical sheet of a binder. Position is its number in turning
// order (from 1), so the Pages after it are the ones numbered higher.
type BinderLeaf struct {
	crud.BaseEntity
	CollectionID uuid.UUID `gorm:"type:uuid;not null"`
	Position     int       `gorm:"not null"`
}

func (BinderLeaf) TableName() string { return "binder_leaves" }

// BinderSlot is one occupied pocket on a Page. Empty Slots have no row.
// Exactly one of CardID and Unidentified is set. Side is "front" or "back".
type BinderSlot struct {
	crud.BaseEntity
	CollectionID uuid.UUID  `gorm:"type:uuid;not null"`
	LeafID       uuid.UUID  `gorm:"type:uuid;not null"`
	Side         string     `gorm:"not null"`
	Row          int        `gorm:"column:slot_row;not null"`
	Column       int        `gorm:"column:slot_column;not null"`
	CardID       *uuid.UUID `gorm:"type:uuid"`
	Unidentified bool       `gorm:"not null;default:false"`

	// Card and Leaf are loaded only via crud.Specification.PreloadRelations.
	// Leaf gives the Slot's Page, since a Page's number follows its Leaf's Position.
	Card Card       `gorm:"foreignKey:CardID"`
	Leaf BinderLeaf `gorm:"foreignKey:LeafID"`
}

func (BinderSlot) TableName() string { return "binder_slots" }
