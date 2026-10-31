package repository

import (
	"context"
	"strings"
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
	case err := <-waiterDone:
		t.Fatalf("a second transaction returned while the first held the identity lock: %v", err)
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

func TestUserRepository_AdoptByEmail(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(crud.NewRepository[entity.User](db))
	ctx := context.Background()

	t.Run("rewrites the one user with the email, ignoring case, and keeps its profile", func(t *testing.T) {
		profile := newTestProfile(t, db, "Copied")
		var copied entity.User
		require.NoError(t, db.First(&copied, "id = ?", profile.UserID).Error)
		subject := uuid.NewString()

		adopted, err := repo.AdoptByEmail(ctx, "dev", subject, strings.ToUpper(copied.Email))

		require.NoError(t, err)
		assert.Equal(t, copied.ID, adopted.ID)
		assert.Equal(t, "dev", adopted.AuthProvider)
		assert.Equal(t, subject, adopted.AuthSubject)
		var stored entity.User
		require.NoError(t, db.First(&stored, "auth_provider = ? AND auth_subject = ?", "dev", subject).Error)
		assert.Equal(t, copied.ID, stored.ID)
		var profiles int64
		require.NoError(t, db.Model(&entity.UserProfile{}).Where("user_id = ?", copied.ID).Count(&profiles).Error)
		assert.EqualValues(t, 1, profiles, "the profile stays attached to the same user")
	})

	t.Run("adopts nothing when no user has the email", func(t *testing.T) {
		adopted, err := repo.AdoptByEmail(ctx, "dev", uuid.NewString(), uniqueEmail(t))

		require.NoError(t, err)
		assert.True(t, adopted.IsZero())
	})

	t.Run("adopts nothing when several users have the email", func(t *testing.T) {
		email := uniqueEmail(t)
		for _, e := range []string{email, strings.ToUpper(email)} {
			_, err := repo.Insert(ctx, entity.User{AuthProvider: "test", AuthSubject: uuid.NewString(), Email: e})
			require.NoError(t, err)
		}
		subject := uuid.NewString()

		adopted, err := repo.AdoptByEmail(ctx, "dev", subject, email)

		require.NoError(t, err)
		assert.True(t, adopted.IsZero())
		var rewritten int64
		require.NoError(t, db.Model(&entity.User{}).Where("auth_subject = ?", subject).Count(&rewritten).Error)
		assert.Zero(t, rewritten)
	})
}
