package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
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

	profile := entity.UserProfile{UserID: userID, Name: "Inventory Test"}
	require.NoError(t, db.Create(&profile).Error)
	c := entity.Collection{ProfileID: profile.ID, Title: "Binder"}
	require.NoError(t, db.Create(&c).Error)
	return c
}

func TestInventoryRepository_EntryLifecycle(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewInventoryRepository(db)
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	card1 := fixture.newCard(t, db, set.ID, nil)
	card2 := fixture.newCard(t, db, set.ID, func(c *entity.Card) { c.LocalID = "002" })
	col := newTestCollection(t, db)

	exists, err := repo.CardExists(ctx, card1.ID)
	require.NoError(t, err)
	assert.True(t, exists)
	exists, err = repo.CardExists(ctx, uuid.New())
	require.NoError(t, err)
	assert.False(t, exists)

	sum, err := repo.SumQuantity(ctx, col.ID)
	require.NoError(t, err)
	assert.Zero(t, sum)
	missing, err := repo.FindEntry(ctx, col.ID, card1.ID)
	require.NoError(t, err)
	assert.True(t, missing.IsZero())

	saved, err := repo.SaveEntry(ctx, entity.InventoryEntry{CollectionID: col.ID, CardID: card1.ID, Quantity: 2})
	require.NoError(t, err)
	assert.False(t, saved.IsZero())
	_, err = repo.SaveEntry(ctx, entity.InventoryEntry{CollectionID: col.ID, CardID: card2.ID, Quantity: 3})
	require.NoError(t, err)

	// Saving again for the same (collection, card) sets the quantity in place.
	updated, err := repo.SaveEntry(ctx, entity.InventoryEntry{CollectionID: col.ID, CardID: card1.ID, Quantity: 5})
	require.NoError(t, err)
	assert.Equal(t, saved.ID, updated.ID)

	found, err := repo.FindEntry(ctx, col.ID, card1.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, found.Quantity)
	sum, err = repo.SumQuantity(ctx, col.ID)
	require.NoError(t, err)
	assert.Equal(t, 8, sum)

	items, err := repo.ListItems(ctx, col.ID)
	require.NoError(t, err)
	require.Len(t, items, 2)
	qty := map[uuid.UUID]int{}
	for _, it := range items {
		qty[it.ID] = it.Quantity
		assert.Equal(t, fixture.rarity.Code, it.RarityCode)
	}
	assert.Equal(t, map[uuid.UUID]int{card1.ID: 5, card2.ID: 3}, qty)

	require.NoError(t, repo.DeleteEntry(ctx, col.ID, card1.ID))
	found, err = repo.FindEntry(ctx, col.ID, card1.ID)
	require.NoError(t, err)
	assert.True(t, found.IsZero())

	// Deleting the Collection cascades to its entries.
	require.NoError(t, db.Delete(&col).Error)
	sum, err = repo.SumQuantity(ctx, col.ID)
	require.NoError(t, err)
	assert.Zero(t, sum)
}
