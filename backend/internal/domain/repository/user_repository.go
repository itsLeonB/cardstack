package repository

import (
	"context"

	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// UserRepository is crud.Repository[entity.User] plus the lock that makes
// first-use provisioning safe under concurrent first requests.
type UserRepository interface {
	crud.Repository[entity.User]
	// LockIdentity blocks until no other transaction holds the same Auth
	// Identity's lock, and holds it until the transaction in ctx ends. Use it
	// inside crud.Transactor.WithinTransaction with the callback's ctx, before
	// checking whether the user exists: there is no row to lock yet, so two
	// first requests would otherwise both miss and both insert.
	LockIdentity(ctx context.Context, provider, subject string) error
	// AdoptByEmail rewrites the auth provider and subject of the one user whose
	// email equals email, ignoring case, and returns it. It returns the zero
	// User, changing nothing, when no user or several users have that email.
	// Call it after LockIdentity, inside the same transaction.
	AdoptByEmail(ctx context.Context, provider, subject, email string) (entity.User, error)
}

type userRepository struct {
	crud.Repository[entity.User]
}

func NewUserRepository(base crud.Repository[entity.User]) UserRepository {
	return &userRepository{Repository: base}
}

func (r *userRepository) LockIdentity(ctx context.Context, provider, subject string) error {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}

	// The unit-separator keeps ("a", "bc") and ("ab", "c") from sharing a lock.
	err = db.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", provider+"\x1f"+subject).Error
	if err != nil {
		return ungerr.Wrap(err, "locking auth identity")
	}
	return nil
}

func (r *userRepository) AdoptByEmail(ctx context.Context, provider, subject, email string) (entity.User, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return entity.User{}, err
	}

	// One statement, so the count and the rewrite see the same rows.
	var adopted []entity.User
	err = db.Raw(`UPDATE users SET auth_provider = ?, auth_subject = ?, updated_at = now()
WHERE lower(email) = lower(?) AND (SELECT count(*) FROM users WHERE lower(email) = lower(?)) = 1
RETURNING *`, provider, subject, email, email).Scan(&adopted).Error
	if err != nil {
		return entity.User{}, ungerr.Wrap(err, "adopting user by email")
	}
	if len(adopted) == 0 {
		return entity.User{}, nil
	}
	return adopted[0], nil
}
