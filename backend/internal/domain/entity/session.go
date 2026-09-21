package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// Session is the GORM row backing authkit.SessionStore.
type Session struct {
	crud.BaseEntity
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`
}

func (Session) TableName() string { return "sessions" }
