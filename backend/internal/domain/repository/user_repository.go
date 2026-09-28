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
// crud.Repository[entity.User], plus a second generic repository for
// entity.UserProfile — the table authkit.User.ProfileID's value actually
// points to, looked up by user_id (see entity/user_profile.go's doc
// comment). Method names don't line up
// with crud.Repository's (Create vs Insert, FindByID vs FindFirst, ...), so
// each authkit.UserStore method is hand-written, delegating to the
// embedded/held repositories' generic Insert/FindFirst/Update for the
// actual queries and translating between entity.User's uuid.UUID primary
// key and authkit.User's string ID at the boundary.
type UserRepository struct {
	crud.Repository[entity.User]
	profiles crud.Repository[entity.UserProfile]
}

// NewUserRepository builds a UserRepository over db. Returned as the
// concrete type (rather than authkit.UserStore) so callers can also use it
// as an auth.ProfileLookup — see internal/provider/repository_provider.go's
// ProvideUserStore/ProvideProfileLookup.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		Repository: crud.NewRepository[entity.User](db),
		profiles:   crud.NewRepository[entity.UserProfile](db),
	}
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

	return toAuthUser(user, ""), nil
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

	return toAuthUser(user, ""), nil
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash string) (authkit.User, error) {
	user, err := r.Insert(ctx, entity.User{Email: email, PasswordHash: passwordHash})
	if err != nil {
		return authkit.User{}, err
	}

	return toAuthUser(user, ""), nil
}

// CreateOAuth creates a new user from an OAuth login, with a user_profiles
// row for the OAuth-provided name. OAuth isn't wired up this ticket
// (Deps.OAuth stays nil, see the ticket 02 plan), so this path is never
// exercised in practice, but is implemented properly rather than stubbed
// since it's still part of the authkit.UserStore contract. avatar isn't
// modeled yet, so it's accepted but not persisted.
func (r *UserRepository) CreateOAuth(ctx context.Context, email, name, _ string) (authkit.User, error) {
	user, err := r.Insert(ctx, entity.User{Email: email, Verified: true})
	if err != nil {
		return authkit.User{}, err
	}

	profile, err := r.upsertProfile(ctx, user.ID, name)
	if err != nil {
		return authkit.User{}, err
	}

	return toAuthUser(user, profileIDString(profile)), nil
}

func (r *UserRepository) SetVerified(ctx context.Context, userID string, name, _ string) (authkit.User, error) {
	user, err := r.findByID(ctx, userID)
	if err != nil {
		return authkit.User{}, err
	}

	user.Verified = true
	updated, err := r.Update(ctx, user)
	if err != nil {
		return authkit.User{}, err
	}

	profile, err := r.upsertProfile(ctx, updated.ID, name)
	if err != nil {
		return authkit.User{}, err
	}

	return toAuthUser(updated, profileIDString(profile)), nil
}

// upsertProfile creates the user's user_profiles row on first call, or
// updates the existing row's name on a later one — found by user_id, since
// the FK runs one-directional from user_profiles to users (no column on
// users to follow). name == "" is a no-op: it returns whatever profile
// already exists (the zero value if none).
func (r *UserRepository) upsertProfile(ctx context.Context, userID uuid.UUID, name string) (entity.UserProfile, error) {
	profile, err := r.profiles.FindFirst(ctx, crud.Specification[entity.UserProfile]{
		Model: entity.UserProfile{UserID: userID},
	})
	if err != nil {
		return entity.UserProfile{}, err
	}

	if name == "" {
		return profile, nil
	}

	if profile.IsZero() {
		return r.profiles.Insert(ctx, entity.UserProfile{UserID: userID, Name: name})
	}

	if name != profile.Name {
		profile.Name = name
		return r.profiles.Update(ctx, profile)
	}

	return profile, nil
}

// FindProfileIDByUserID resolves a user's user_profiles row ID by user_id,
// so SessionGuard can put it in the request context after verifying a JWT —
// profile_id deliberately isn't embedded in the JWT itself (users is the
// auth table, user_profiles is domain data, with no FK between them; see
// entity/user_profile.go's doc comment).
func (r *UserRepository) FindProfileIDByUserID(ctx context.Context, userID string) (string, error) {
	id, err := uuid.Parse(userID)
	if err != nil || id == uuid.Nil {
		return "", authkit.ErrUserNotFound
	}

	profile, err := r.profiles.FindFirst(ctx, crud.Specification[entity.UserProfile]{
		Model: entity.UserProfile{UserID: id},
	})
	if err != nil {
		return "", err
	}
	if profile.IsZero() {
		return "", authkit.ErrUserNotFound
	}

	return profile.ID.String(), nil
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

func toAuthUser(u entity.User, profileID string) authkit.User {
	return authkit.User{
		ID:           u.ID.String(),
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Verified:     u.Verified,
		ProfileID:    profileID,
	}
}

// profileIDString returns p's ID, or "" if p is the zero value (no
// user_profiles row exists yet).
func profileIDString(p entity.UserProfile) string {
	if p.IsZero() {
		return ""
	}
	return p.ID.String()
}
