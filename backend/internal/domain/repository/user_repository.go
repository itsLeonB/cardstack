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
