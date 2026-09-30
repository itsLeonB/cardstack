package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInventoryRepository_SumQuantity(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewInventoryRepository(crud.NewRepository[entity.InventoryEntry](db))
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	card1 := fixture.newCard(t, db, set.ID, nil)
	card2 := fixture.newCard(t, db, set.ID, func(c *entity.Card) { c.LocalID = "002" })

	user, err := NewUserRepository(db).Create(ctx, uniqueEmail(t), "hash")
	require.NoError(t, err)
	userID, err := uuid.Parse(user.ID)
	require.NoError(t, err)
	profile := entity.UserProfile{UserID: userID, Name: "Inventory Test"}
	require.NoError(t, db.Create(&profile).Error)
	col := entity.Collection{ProfileID: profile.ID, Title: "Binder"}
	require.NoError(t, db.Create(&col).Error)

	sum, err := repo.SumQuantity(ctx, col.ID)
	require.NoError(t, err)
	assert.Zero(t, sum)

	_, err = repo.Insert(ctx, entity.InventoryEntry{CollectionID: col.ID, CardID: card1.ID, Quantity: 2})
	require.NoError(t, err)

	// Inside a transaction the sum sees that transaction's uncommitted writes.
	err = crud.NewTransactor(db).WithinTransaction(ctx, func(ctx context.Context) error {
		if _, err := repo.Insert(ctx, entity.InventoryEntry{CollectionID: col.ID, CardID: card2.ID, Quantity: 3}); err != nil {
			return err
		}
		sum, err := repo.SumQuantity(ctx, col.ID)
		assert.Equal(t, 5, sum)
		return err
	})
	require.NoError(t, err)
}
