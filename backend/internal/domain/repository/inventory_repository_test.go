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

	profile := newTestProfile(t, db, "Inventory Test")
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

// TestInventoryRepository_Constraints covers the schema guarantees the
// service relies on now that it no longer maps 23505: (collection, card) is
// unique, and deleting a Collection deletes its entries.
func TestInventoryRepository_Constraints(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewInventoryRepository(crud.NewRepository[entity.InventoryEntry](db))
	fixture := newCatalogFixture(t, db)
	card := fixture.newCard(t, db, fixture.newExpansionSet(t, db, nil, nil).ID, nil)

	profile := newTestProfile(t, db, "Inventory Test")
	col := entity.Collection{ProfileID: profile.ID, Title: "Binder"}
	require.NoError(t, db.Create(&col).Error)

	_, err := repo.Insert(ctx, entity.InventoryEntry{CollectionID: col.ID, CardID: card.ID, Quantity: 1})
	require.NoError(t, err)
	_, err = repo.Insert(ctx, entity.InventoryEntry{CollectionID: col.ID, CardID: card.ID, Quantity: 2})
	assert.Error(t, err, "duplicate (collection, card) must be rejected")

	require.NoError(t, db.Delete(&col).Error)
	left, err := repo.FindAll(ctx, crud.Specification[entity.InventoryEntry]{Model: entity.InventoryEntry{CollectionID: col.ID}})
	require.NoError(t, err)
	assert.Empty(t, left)
}

func TestInventoryRepository_ListHoldings(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewInventoryRepository(crud.NewRepository[entity.InventoryEntry](db))
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	card := fixture.newCard(t, db, set.ID, nil)
	otherCard := fixture.newCard(t, db, set.ID, func(c *entity.Card) { c.LocalID = "002" })

	newProfile := func() entity.UserProfile { return newTestProfile(t, db, "Holdings Test") }
	newCollection := func(p entity.UserProfile, title string, cardID uuid.UUID, qty int) entity.Collection {
		c := entity.Collection{ProfileID: p.ID, Title: title}
		require.NoError(t, db.Create(&c).Error)
		_, err := repo.Insert(ctx, entity.InventoryEntry{CollectionID: c.ID, CardID: cardID, Quantity: qty})
		require.NoError(t, err)
		return c
	}

	me, them := newProfile(), newProfile()
	zebra := newCollection(me, "Zebra", card.ID, 2)
	apple := newCollection(me, "Apple", card.ID, 5)
	newCollection(me, "Other card", otherCard.ID, 1)
	newCollection(them, "Theirs", card.ID, 9)

	got, err := repo.ListHoldings(ctx, me.ID, card.ID)
	require.NoError(t, err)
	assert.Equal(t, []CardHolding{
		{CollectionID: apple.ID, CollectionTitle: "Apple", Quantity: 5},
		{CollectionID: zebra.ID, CollectionTitle: "Zebra", Quantity: 2},
	}, got)

	// Held nowhere by this profile, or an unknown Card: empty, not an error.
	got, err = repo.ListHoldings(ctx, newProfile().ID, card.ID)
	require.NoError(t, err)
	assert.Empty(t, got)
	got, err = repo.ListHoldings(ctx, me.ID, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, got)
}
