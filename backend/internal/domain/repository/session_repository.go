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

// SessionRepository implements authkit.SessionStore on top of a generic
// crud.Repository[entity.Session]. Delete and Touch take a bare session-ID
// string (per authkit.SessionStore), which shadows/hand-writes over the
// embedded repository's model-shaped Delete and its lack of a Touch method.
type SessionRepository struct {
	crud.Repository[entity.Session]
}

// NewSessionRepository builds a SessionRepository over db.
func NewSessionRepository(db *gorm.DB) authkit.SessionStore {
	return &SessionRepository{Repository: crud.NewRepository[entity.Session](db)}
}

func (r *SessionRepository) Create(ctx context.Context, userID string) (authkit.Session, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return authkit.Session{}, authkit.ErrUserNotFound
	}

	session, err := r.Insert(ctx, entity.Session{UserID: id})
	if err != nil {
		return authkit.Session{}, err
	}

	return toAuthSession(session), nil
}

// parseSessionID parses id and rejects uuid.Nil up front. go-crud's
// crud.Specification builds its WHERE clause from non-zero struct fields
// (see WhereBySpec), so a zero-value ID would silently drop the ID filter
// entirely and match an arbitrary row instead of correctly reporting "not
// found" — a real uuidv7()-generated primary key is never uuid.Nil, so this
// only ever rejects a malformed/absent ID.
func parseSessionID(id string) (uuid.UUID, error) {
	sessionID, err := uuid.Parse(id)
	if err != nil || sessionID == uuid.Nil {
		return uuid.Nil, authkit.ErrSessionNotFound
	}
	return sessionID, nil
}

func (r *SessionRepository) GetByID(ctx context.Context, id string) (authkit.Session, error) {
	sessionID, err := parseSessionID(id)
	if err != nil {
		return authkit.Session{}, err
	}

	session, err := r.FindFirst(ctx, crud.Specification[entity.Session]{
		Model: entity.Session{BaseEntity: crud.BaseEntity{ID: sessionID}},
	})
	if err != nil {
		return authkit.Session{}, err
	}
	if session.IsZero() {
		return authkit.Session{}, authkit.ErrSessionNotFound
	}

	return toAuthSession(session), nil
}

// Delete shadows the embedded crud.Repository[entity.Session].Delete (which
// takes a full model) to satisfy authkit.SessionStore's bare-ID signature.
func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	sessionID, err := parseSessionID(id)
	if err != nil {
		return err
	}

	return r.Repository.Delete(ctx, entity.Session{BaseEntity: crud.BaseEntity{ID: sessionID}})
}

// Touch updates the session's last-activity timestamp. go-crud has no
// generic "touch" operation, so this issues its own update through the
// embedded repository's transaction-aware GetGormInstance.
func (r *SessionRepository) Touch(ctx context.Context, id string) error {
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return authkit.ErrSessionNotFound
	}

	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}

	return db.Model(&entity.Session{}).
		Where("id = ?", sessionID).
		Update("updated_at", time.Now()).
		Error
}

func toAuthSession(s entity.Session) authkit.Session {
	return authkit.Session{ID: s.ID.String(), UserID: s.UserID.String()}
}
