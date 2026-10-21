package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
	"gorm.io/datatypes"
)

// ScanFeedback is one confirmed Scan or Page Review cell: what the matcher said
// (Detection) and what the user chose (CardID, with its rank). Kind is the
// Correction kind, or confirmed (docs/adr/0022). Its Page, slot and grid fields
// are set only for a binder_page row.
type ScanFeedback struct {
	crud.BaseEntity
	ProfileID      uuid.UUID `gorm:"type:uuid;not null"`
	CollectionID   uuid.UUID `gorm:"type:uuid;not null"`
	Source         string    `gorm:"not null"`
	Kind           string    `gorm:"not null"`
	Detection      datatypes.JSON
	CardID         *uuid.UUID `gorm:"type:uuid"`
	CardRank       *int       `gorm:"column:card_rank"`
	CardRankSource *string    `gorm:"column:card_rank_source"`
	ImageKey       *string
	Page           *int
	SlotRow        *int `gorm:"column:slot_row"`
	SlotColumn     *int `gorm:"column:slot_column"`
	GridRows       *int
	GridColumns    *int
	Corners        datatypes.JSON
}

func (ScanFeedback) TableName() string { return "scan_feedback" }
