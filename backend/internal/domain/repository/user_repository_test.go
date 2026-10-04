package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_AuthIdentityIsUnique(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(crud.NewRepository[entity.User](db))
	ctx := context.Background()
	subject := uuid.NewString()

	_, err := repo.Insert(ctx, entity.User{AuthProvider: "test", AuthSubject: subject, Email: uniqueEmail(t)})
	require.NoError(t, err)

	_, err = repo.Insert(ctx, entity.User{AuthProvider: "test", AuthSubject: subject, Email: uniqueEmail(t)})
	assert.Error(t, err, "the same provider and subject must not map to two users")

	_, err = repo.Insert(ctx, entity.User{AuthProvider: "other", AuthSubject: subject, Email: uniqueEmail(t)})
	assert.NoError(t, err, "the same subject under another provider is a different identity")
}

func TestUserRepository_EmailIsNotUnique(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(crud.NewRepository[entity.User](db))
	ctx := context.Background()
	email := uniqueEmail(t)

	_, err := repo.Insert(ctx, entity.User{AuthProvider: "test", AuthSubject: uuid.NewString(), Email: email})
	require.NoError(t, err)
	_, err = repo.Insert(ctx, entity.User{AuthProvider: "test", AuthSubject: uuid.NewString(), Email: email})
	assert.NoError(t, err, "two identities may share an email")
}

func TestUserRepository_DeletingUserCascadesToProfile(t *testing.T) {
	db := testDB(t)
	profile := newTestProfile(t, db, "Cascade")

	require.NoError(t, db.Delete(&entity.User{BaseEntity: crud.BaseEntity{ID: profile.UserID}}).Error)

	var count int64
	require.NoError(t, db.Model(&entity.UserProfile{}).Where("id = ?", profile.ID).Count(&count).Error)
	assert.Zero(t, count)
}

func TestUserRepository_LockIdentity(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(crud.NewRepository[entity.User](db))
	tx := crud.NewTransactor(db)
	subject := uuid.NewString()

	holding := make(chan struct{})
	release := make(chan struct{})
	holderDone := make(chan error, 1)
	go func() {
		holderDone <- tx.WithinTransaction(context.Background(), func(ctx context.Context) error {
			if err := repo.LockIdentity(ctx, "test", subject); err != nil {
				return err
			}
			close(holding)
			<-release
			return nil
		})
	}()
	select {
	case <-holding:
	case err := <-holderDone:
		t.Fatalf("the first transaction ended before holding the lock: %v", err)
	}

	waiterDone := make(chan error, 1)
	go func() {
		waiterDone <- tx.WithinTransaction(context.Background(), func(ctx context.Context) error {
			return repo.LockIdentity(ctx, "test", subject)
		})
	}()

	select {
	case <-waiterDone:
		t.Fatal("a second transaction acquired the identity lock while the first held it")
	case <-time.After(300 * time.Millisecond):
	}

	// A different identity is not blocked by the held lock.
	require.NoError(t, tx.WithinTransaction(context.Background(), func(ctx context.Context) error {
		return repo.LockIdentity(ctx, "test", uuid.NewString())
	}))

	close(release)
	require.NoError(t, <-holderDone)
	require.NoError(t, <-waiterDone)
}
