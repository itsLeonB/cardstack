package entity

import (
	crud "github.com/itsLeonB/go-crud"
)

// User is the one row we own per Auth Identity: the identity provider owns
// credentials and sessions, so this holds only who the provider says the
// caller is. AuthProvider and AuthSubject are unique together.
type User struct {
	crud.BaseEntity
	AuthProvider string `gorm:"not null;uniqueIndex:idx_users_auth_identity"`
	AuthSubject  string `gorm:"not null;uniqueIndex:idx_users_auth_identity"`
	Email        string `gorm:"not null;index"`
}

func (User) TableName() string { return "users" }
