package repository

import (
	"context"
	"errors"
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

	require.NoError(t, repo.InsertEntry(ctx, entity.InventoryEntry{CollectionID: col.ID, CardID: card1.ID, Quantity: 2}))
	require.NoError(t, repo.InsertEntry(ctx, entity.InventoryEntry{CollectionID: col.ID, CardID: card2.ID, Quantity: 3}))

	// A second insert for the same (collection, card) is refused, not merged.
	err = repo.InsertEntry(ctx, entity.InventoryEntry{CollectionID: col.ID, CardID: card1.ID, Quantity: 9})
	assert.ErrorIs(t, err, ErrEntryExists)

	require.NoError(t, repo.UpdateQuantity(ctx, col.ID, card1.ID, 5))
	assert.ErrorIs(t, repo.UpdateQuantity(ctx, col.ID, uuid.New(), 1), ErrEntryNotFound)

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
	// Update never resurrects a deleted entry.
	assert.ErrorIs(t, repo.UpdateQuantity(ctx, col.ID, card1.ID, 1), ErrEntryNotFound)

	// Deleting the Collection cascades to its entries.
	require.NoError(t, db.Delete(&col).Error)
	sum, err = repo.SumQuantity(ctx, col.ID)
	require.NoError(t, err)
	assert.Zero(t, sum)
}

func TestInventoryRepository_WithLockedCollection(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewInventoryRepository(db)
	col := newTestCollection(t, db)

	var got entity.Collection
	require.NoError(t, repo.WithLockedCollection(ctx, col.ProfileID, col.ID, func(_ InventoryRepository, c entity.Collection) error {
		got = c
		return nil
	}))
	assert.Equal(t, col.ID, got.ID)

	// Another profile's collection is indistinguishable from a missing one.
	called := false
	err := repo.WithLockedCollection(ctx, uuid.New(), col.ID, func(InventoryRepository, entity.Collection) error {
		called = true
		return nil
	})
	assert.ErrorIs(t, err, ErrCollectionNotFound)
	assert.False(t, called)

	// An error from fn rolls the transaction back.
	fixture := newCatalogFixture(t, db)
	card := fixture.newCard(t, db, fixture.newExpansionSet(t, db, nil, nil).ID, nil)
	boom := errors.New("boom")
	err = repo.WithLockedCollection(ctx, col.ProfileID, col.ID, func(tx InventoryRepository, _ entity.Collection) error {
		require.NoError(t, tx.InsertEntry(ctx, entity.InventoryEntry{CollectionID: col.ID, CardID: card.ID, Quantity: 1}))
		return boom
	})
	assert.ErrorIs(t, err, boom)
	sum, err := repo.SumQuantity(ctx, col.ID)
	require.NoError(t, err)
	assert.Zero(t, sum)
}
