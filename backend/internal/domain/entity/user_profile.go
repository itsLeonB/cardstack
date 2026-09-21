package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// UserProfile holds the display info authkit.User.ProfileID references —
// a separate table rather than a text column on users, since authkit
// itself only ever treats ProfileID as an opaque ID it hands back and
// forth (see UserRepository's SetVerified/CreateOAuth, the only writers).
type UserProfile struct {
	crud.BaseEntity
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`
	Name   string    `gorm:"not null"`
}

func (UserProfile) TableName() string { return "user_profiles" }
