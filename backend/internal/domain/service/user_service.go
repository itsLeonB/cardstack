package service

import (
	"context"
	"strings"

	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
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
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		var err error
		caller, err = s.provision(ctx, identity)
		return err
	})
	if err != nil {
		return dto.CallerIdentity{}, err
	}

	s.cache.Set(identity, caller)

	return caller, nil
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
		profile, err = s.profiles.Insert(ctx, entity.UserProfile{UserID: user.ID, Name: profileName(identity)})
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
