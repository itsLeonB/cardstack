package repository

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestCollection(t *testing.T, db *gorm.DB) entity.Collection {
	t.Helper()
	user, err := NewUserRepository(db).Create(context.Background(), uniqueEmail(t), "hash")
	require.NoError(t, err)
	userID, err := uuid.Parse(user.ID)
	require.NoError(t, err)
	profile := entity.UserProfile{UserID: userID, Name: "Collection Test"}
	require.NoError(t, db.Create(&profile).Error)
	col := entity.Collection{ProfileID: profile.ID, Title: "Binder"}
	require.NoError(t, db.Create(&col).Error)
	return col
}

func requireNotFound(t *testing.T, err error) {
	t.Helper()
	var appErr ungerr.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, http.StatusNotFound, appErr.HttpStatus())
}

func TestCollectionRepository_GetOwnedCollection(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewCollectionRepository(crud.NewRepository[entity.Collection](db))
	col := newTestCollection(t, db)
	other := newTestCollection(t, db)

	got, err := repo.GetOwnedCollection(ctx, col.ProfileID, col.ID, false)
	require.NoError(t, err)
	assert.Equal(t, col.ID, got.ID)

	t.Run("foreign owner", func(t *testing.T) {
		_, err := repo.GetOwnedCollection(ctx, other.ProfileID, col.ID, false)
		requireNotFound(t, err)
	})
	t.Run("missing", func(t *testing.T) {
		_, err := repo.GetOwnedCollection(ctx, col.ProfileID, uuid.New(), false)
		requireNotFound(t, err)
	})
	t.Run("nil ids", func(t *testing.T) {
		_, err := repo.GetOwnedCollection(ctx, col.ProfileID, uuid.Nil, false)
		requireNotFound(t, err)
		_, err = repo.GetOwnedCollection(ctx, uuid.Nil, col.ID, false)
		requireNotFound(t, err)
		_, err = repo.GetOwnedCollection(ctx, uuid.Nil, uuid.Nil, false)
		requireNotFound(t, err)
	})
}

// TestCollectionRepository_GetOwnedCollection_ForUpdate: inside a transaction
// the match is row-locked, so another connection can't lock it (NOWAIT fails);
// without forUpdate it can.
func TestCollectionRepository_GetOwnedCollection_ForUpdate(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewCollectionRepository(crud.NewRepository[entity.Collection](db))
	col := newTestCollection(t, db)

	tryLock := func() error {
		return db.Exec("SELECT id FROM collections WHERE id = ? FOR UPDATE NOWAIT", col.ID).Error
	}

	err := crud.NewTransactor(db).WithinTransaction(ctx, func(ctx context.Context) error {
		if _, err := repo.GetOwnedCollection(ctx, col.ProfileID, col.ID, false); err != nil {
			return err
		}
		assert.NoError(t, tryLock(), "plain read must not lock")

		if _, err := repo.GetOwnedCollection(ctx, col.ProfileID, col.ID, true); err != nil {
			return err
		}
		assert.Error(t, tryLock(), "forUpdate must hold the row lock")
		return nil
	})
	require.NoError(t, err)
}
