package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// UserService maps the Auth Identity a verified token carries to the user and
// profile we own.
type UserService interface {
	// ResolveCaller returns the identity's user and profile, creating both in
	// one transaction on first use. The profile name comes from the identity's
	// name, falling back to the local part of its email, and is never changed
	// afterwards; the stored email is updated only when it differs.
	ResolveCaller(ctx context.Context, identity dto.AuthIdentity) (dto.CallerIdentity, error)
	// Profile returns a profile's name and Slug, for the caller's own profile.
	Profile(ctx context.Context, lookup dto.ProfileLookup) (dto.ProfileSummary, error)
}

// IdentityCache lets most requests skip the database. A hit must be for the
// same email the identity carries now, so a changed email claim misses and
// reaches the update path. Its implementation is infrastructure (ADR-0011).
type IdentityCache interface {
	Get(identity dto.AuthIdentity) (dto.CallerIdentity, bool)
	Set(identity dto.AuthIdentity, caller dto.CallerIdentity)
}

type userService struct {
	tx       crud.Transactor
	users    repository.UserRepository
	profiles crud.Repository[entity.UserProfile]
	cache    IdentityCache
}

func NewUserService(tx crud.Transactor, users repository.UserRepository, profiles crud.Repository[entity.UserProfile], cache IdentityCache) UserService {
	return &userService{tx: tx, users: users, profiles: profiles, cache: cache}
}

func (s *userService) ResolveCaller(ctx context.Context, identity dto.AuthIdentity) (dto.CallerIdentity, error) {
	// An empty provider or subject would drop its condition from the lookup
	// (see crud.WhereBySpec) and match any user.
	if identity.Provider == "" || identity.Subject == "" {
		return dto.CallerIdentity{}, ungerr.UnauthorizedError("invalid auth identity")
	}

	if caller, ok := s.cache.Get(identity); ok {
		return caller, nil
	}

	var caller dto.CallerIdentity
	var err error
	for range maxProvisionAttempts {
		// A failed insert aborts the Postgres transaction, so a retry has to
		// rerun the whole transaction, not just the insert.
		err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
			var err error
			caller, err = s.provision(ctx, identity)
			return err
		})
		if !isUniqueViolation(err) {
			break
		}
	}
	if err != nil {
		return dto.CallerIdentity{}, err
	}

	s.cache.Set(identity, caller)

	return caller, nil
}

func (s *userService) Profile(ctx context.Context, lookup dto.ProfileLookup) (dto.ProfileSummary, error) {
	// A nil ID would drop its condition from the lookup (see crud.WhereBySpec)
	// and match any profile.
	if lookup.ProfileID == uuid.Nil {
		return dto.ProfileSummary{}, ungerr.NotFoundError("profile not found")
	}

	profile, err := s.profiles.FindFirst(ctx, crud.Specification[entity.UserProfile]{
		Model: entity.UserProfile{BaseEntity: crud.BaseEntity{ID: lookup.ProfileID}},
	})
	if err != nil {
		return dto.ProfileSummary{}, err
	}
	if profile.IsZero() {
		return dto.ProfileSummary{}, ungerr.NotFoundError("profile not found")
	}

	return dto.ProfileSummary{Name: profile.Name, Slug: profile.Slug}, nil
}

func (s *userService) provision(ctx context.Context, identity dto.AuthIdentity) (dto.CallerIdentity, error) {
	if err := s.users.LockIdentity(ctx, identity.Provider, identity.Subject); err != nil {
		return dto.CallerIdentity{}, err
	}

	user, err := s.users.FindFirst(ctx, crud.Specification[entity.User]{
		Model: entity.User{AuthProvider: identity.Provider, AuthSubject: identity.Subject},
	})
	if err != nil {
		return dto.CallerIdentity{}, err
	}

	switch {
	case user.IsZero():
		user, err = s.users.Insert(ctx, entity.User{AuthProvider: identity.Provider, AuthSubject: identity.Subject, Email: identity.Email})
	case user.Email != identity.Email:
		user.Email = identity.Email
		user, err = s.users.Update(ctx, user)
	}
	if err != nil {
		return dto.CallerIdentity{}, err
	}

	profile, err := s.profiles.FindFirst(ctx, crud.Specification[entity.UserProfile]{
		Model: entity.UserProfile{UserID: user.ID},
	})
	if err != nil {
		return dto.CallerIdentity{}, err
	}
	if profile.IsZero() {
		// The Slug embeds the id, so the id is made here rather than by the
		// column default.
		id, idErr := uuid.NewV7()
		if idErr != nil {
			return dto.CallerIdentity{}, ungerr.Wrap(idErr, "generating profile id")
		}
		name := profileName(identity)
		profile, err = s.profiles.Insert(ctx, entity.UserProfile{
			BaseEntity: crud.BaseEntity{ID: id},
			UserID:     user.ID,
			Name:       name,
			Slug:       mapper.SlugOf(name, "user", id),
		})
		if err != nil {
			return dto.CallerIdentity{}, err
		}
	}

	return dto.CallerIdentity{UserID: user.ID, ProfileID: profile.ID}, nil
}

// profileName is the name claim, or the email's local part for accounts with
// no name (plain email signups).
func profileName(identity dto.AuthIdentity) string {
	if name := strings.TrimSpace(identity.Name); name != "" {
		return name
	}
	local, _, _ := strings.Cut(identity.Email, "@")
	return local
}

// maxProvisionAttempts bounds the retries after a unique violation. A profile's
// slug can clash with another's; each attempt makes a fresh id and slug.
const maxProvisionAttempts = 3

// isUniqueViolation reports whether err carries SQLSTATE 23505, the only unique
// violation provision's insert can raise besides its id, which a retry also
// fixes. ungerr.Wrap has no Unwrap method, so the chain is walked with ungerr.Unwrap.
func isUniqueViolation(err error) bool {
	for {
		if coded, ok := err.(interface{ SQLState() string }); ok {
			return coded.SQLState() == "23505"
		}
		if _, wrapped := err.(*ungerr.UnknownError); !wrapped {
			return false
		}
		err = ungerr.Unwrap(err)
	}
}
