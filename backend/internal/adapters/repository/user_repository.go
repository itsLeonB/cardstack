package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	authkit "github.com/itsLeonB/go-authkit"
	crud "github.com/itsLeonB/go-crud"
	"gorm.io/gorm"
)

// UserRepository implements authkit.UserStore on top of a generic
// crud.Repository[entity.User]. Its method names don't line up with
// crud.Repository's (Create vs Insert, FindByID vs FindFirst, ...), so each
// authkit.UserStore method is hand-written, delegating to the embedded
// repository's generic Insert/FindFirst/Update for the actual query and
// translating between entity.User's uuid.UUID primary key and authkit.User's
// string ID at the boundary.
type UserRepository struct {
	crud.Repository[entity.User]
}

// NewUserRepository builds a UserRepository over db.
func NewUserRepository(db *gorm.DB) authkit.UserStore {
	return &UserRepository{Repository: crud.NewRepository[entity.User](db)}
}

// findByID looks up a user by ID, parsing userID and rejecting uuid.Nil up
// front. go-crud's crud.Specification builds its WHERE clause from
// non-zero struct fields (see WhereBySpec), so a zero-value ID here would
// silently drop the ID filter entirely and match an arbitrary row instead
// of correctly reporting "not found" — a real uuidv7()-generated primary
// key is never uuid.Nil, so this only ever rejects a malformed/absent ID.
func (r *UserRepository) findByID(ctx context.Context, userID string) (entity.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil || id == uuid.Nil {
		return entity.User{}, authkit.ErrUserNotFound
	}

	user, err := r.FindFirst(ctx, crud.Specification[entity.User]{
		Model: entity.User{BaseEntity: crud.BaseEntity{ID: id}},
	})
	if err != nil {
		return entity.User{}, err
	}
	if user.IsZero() {
		return entity.User{}, authkit.ErrUserNotFound
	}

	return user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, userID string) (authkit.User, error) {
	user, err := r.findByID(ctx, userID)
	if err != nil {
		return authkit.User{}, err
	}

	return toAuthUser(user), nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (authkit.User, error) {
	// Same zero-value gotcha as findByID: an empty email would otherwise
	// match an arbitrary row instead of correctly reporting "not found".
	if email == "" {
		return authkit.User{}, authkit.ErrUserNotFound
	}

	user, err := r.FindFirst(ctx, crud.Specification[entity.User]{
		Model: entity.User{Email: email},
	})
	if err != nil {
		return authkit.User{}, err
	}
	if user.IsZero() {
		return authkit.User{}, authkit.ErrUserNotFound
	}

	return toAuthUser(user), nil
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash string) (authkit.User, error) {
	user, err := r.Insert(ctx, entity.User{Email: email, PasswordHash: passwordHash})
	if err != nil {
		return authkit.User{}, err
	}

	return toAuthUser(user), nil
}

// CreateOAuth creates a new user from an OAuth login. OAuth isn't wired up
// this ticket (Deps.OAuth stays nil, see the ticket 02 plan), so this path
// is never exercised in practice, but is implemented properly rather than
// stubbed since it's still part of the authkit.UserStore contract. avatar
// isn't modeled on entity.User yet, so it's accepted but not persisted.
func (r *UserRepository) CreateOAuth(ctx context.Context, email, name, _ string) (authkit.User, error) {
	user, err := r.Insert(ctx, entity.User{Email: email, Verified: true, ProfileID: name})
	if err != nil {
		return authkit.User{}, err
	}

	return toAuthUser(user), nil
}

func (r *UserRepository) SetVerified(ctx context.Context, userID string, name, _ string) (authkit.User, error) {
	user, err := r.findByID(ctx, userID)
	if err != nil {
		return authkit.User{}, err
	}

	user.Verified = true
	if name != "" {
		user.ProfileID = name
	}

	updated, err := r.Update(ctx, user)
	if err != nil {
		return authkit.User{}, err
	}

	return toAuthUser(updated), nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	user, err := r.findByID(ctx, userID)
	if err != nil {
		return err
	}

	user.PasswordHash = passwordHash
	_, err = r.Update(ctx, user)
	return err
}

func (r *UserRepository) Exists(ctx context.Context, userID string) error {
	_, err := r.findByID(ctx, userID)
	return err
}

func toAuthUser(u entity.User) authkit.User {
	return authkit.User{
		ID:           u.ID.String(),
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Verified:     u.Verified,
		ProfileID:    u.ProfileID,
	}
}
