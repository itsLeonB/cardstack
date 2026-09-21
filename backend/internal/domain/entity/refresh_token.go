package entity

import (
	"time"

	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// RefreshToken is the GORM row backing authkit.RefreshTokenStore. TokenHash
// stores the SHA-256 hash of the raw refresh token authkit hands to
// clients — the raw value is never persisted.
type RefreshToken struct {
	crud.BaseEntity
	SessionID uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenHash string    `gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"not null"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
