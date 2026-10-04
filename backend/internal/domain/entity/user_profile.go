package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// UserProfile holds the display info domain data hangs off (collections key
// on its ID), a separate table so users stays identity-only. Name is written
// once, when the profile is created.
type UserProfile struct {
	crud.BaseEntity
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`
	Name   string    `gorm:"not null"`
}

func (UserProfile) TableName() string { return "user_profiles" }
