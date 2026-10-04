package service

import (
	"context"
	"strings"
	"sync"
	"time"

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

type userService struct {
	tx       crud.Transactor
	users    repository.UserRepository
	profiles crud.Repository[entity.UserProfile]
	cache    *identityCache
}

func NewUserService(tx crud.Transactor, users repository.UserRepository, profiles crud.Repository[entity.UserProfile]) UserService {
	return &userService{tx: tx, users: users, profiles: profiles, cache: newIdentityCache(time.Now)}
}

func (s *userService) ResolveCaller(ctx context.Context, identity dto.AuthIdentity) (dto.CallerIdentity, error) {
	// An empty provider or subject would drop its condition from the lookup
	// (see crud.WhereBySpec) and match any user.
	if identity.Provider == "" || identity.Subject == "" {
		return dto.CallerIdentity{}, ungerr.UnauthorizedError("invalid auth identity")
	}

	if caller, ok := s.cache.get(identity); ok {
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

	s.cache.set(identity, caller)

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

// identityCacheTTL bounds how stale a cached mapping can be, for example after
// the user or profile row is deleted.
const identityCacheTTL = time.Minute

// identityCacheMax caps the map; a full cache is emptied rather than evicted
// entry by entry, since a miss only costs one database round trip.
const identityCacheMax = 10_000

type identityKey struct{ provider, subject string }

type cachedCaller struct {
	caller    dto.CallerIdentity
	email     string
	expiresAt time.Time
}

// identityCache lets most requests skip the database. An entry also remembers
// the email it was built from, so a changed email claim misses and reaches the
// update path.
type identityCache struct {
	now     func() time.Time
	mu      sync.Mutex
	entries map[identityKey]cachedCaller
}

func newIdentityCache(now func() time.Time) *identityCache {
	return &identityCache{now: now, entries: map[identityKey]cachedCaller{}}
}

func (c *identityCache) get(identity dto.AuthIdentity) (dto.CallerIdentity, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[identityKey{identity.Provider, identity.Subject}]
	if !ok || entry.email != identity.Email || !c.now().Before(entry.expiresAt) {
		return dto.CallerIdentity{}, false
	}

	return entry.caller, true
}

func (c *identityCache) set(identity dto.AuthIdentity, caller dto.CallerIdentity) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= identityCacheMax {
		clear(c.entries)
	}
	c.entries[identityKey{identity.Provider, identity.Subject}] = cachedCaller{
		caller:    caller,
		email:     identity.Email,
		expiresAt: c.now().Add(identityCacheTTL),
	}
}
