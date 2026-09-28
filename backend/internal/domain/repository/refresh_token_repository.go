package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	authkit "github.com/itsLeonB/go-authkit"
	crud "github.com/itsLeonB/go-crud"
	"gorm.io/gorm"
)

// RefreshTokenRepository implements authkit.RefreshTokenStore on top of a
// generic crud.Repository[entity.RefreshToken]. Delete and DeleteBySession
// don't address a single record by primary key (authkit deletes by
// session+hash, or by session alone), so they're hand-written directly
// against the embedded repository's transaction-aware GetGormInstance
// rather than going through crud.Repository.Delete (which deletes by
// model/primary key).
type RefreshTokenRepository struct {
	crud.Repository[entity.RefreshToken]
}

// NewRefreshTokenRepository builds a RefreshTokenRepository over db.
func NewRefreshTokenRepository(db *gorm.DB) authkit.RefreshTokenStore {
	return &RefreshTokenRepository{Repository: crud.NewRepository[entity.RefreshToken](db)}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, sessionID, tokenHash string, expiresAt time.Time) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return authkit.ErrSessionNotFound
	}

	_, err = r.Insert(ctx, entity.RefreshToken{SessionID: sid, TokenHash: tokenHash, ExpiresAt: expiresAt})
	return err
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, hash string) (authkit.RefreshToken, error) {
	// Same zero-value gotcha as the other repositories: crud.Specification's
	// WHERE clause drops zero-value fields, so an empty hash would otherwise
	// match an arbitrary row instead of correctly reporting "not found".
	if hash == "" {
		return authkit.RefreshToken{}, authkit.ErrTokenNotFound
	}

	token, err := r.FindFirst(ctx, crud.Specification[entity.RefreshToken]{
		Model: entity.RefreshToken{TokenHash: hash},
	})
	if err != nil {
		return authkit.RefreshToken{}, err
	}
	if token.IsZero() {
		return authkit.RefreshToken{}, authkit.ErrTokenNotFound
	}

	return toAuthRefreshToken(token), nil
}

// Delete shadows the embedded crud.Repository[entity.RefreshToken].Delete
// (which takes a full model) to satisfy authkit.RefreshTokenStore's
// session+hash signature.
func (r *RefreshTokenRepository) Delete(ctx context.Context, sessionID, tokenHash string) error {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}

	return db.Unscoped().
		Where("session_id = ? AND token_hash = ?", sessionID, tokenHash).
		Delete(&entity.RefreshToken{}).
		Error
}

func (r *RefreshTokenRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}

	return db.Unscoped().
		Where("session_id = ?", sessionID).
		Delete(&entity.RefreshToken{}).
		Error
}

func toAuthRefreshToken(rt entity.RefreshToken) authkit.RefreshToken {
	return authkit.RefreshToken{
		ID:        rt.ID.String(),
		SessionID: rt.SessionID.String(),
		TokenHash: rt.TokenHash,
		ExpiresAt: rt.ExpiresAt,
		CreatedAt: rt.CreatedAt,
	}
}
