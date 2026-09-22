package entity

import (
	crud "github.com/itsLeonB/go-crud"
)

// User is the GORM row backing authkit.UserStore. It carries a uuid.UUID
// primary key (via crud.BaseEntity, PG18 native uuidv7()) rather than
// authkit.User's string ID — the repository adapter converts between the
// two at the authkit.UserStore boundary.
type User struct {
	crud.BaseEntity
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	Verified     bool   `gorm:"not null;default:false"`
}

func (User) TableName() string { return "users" }
