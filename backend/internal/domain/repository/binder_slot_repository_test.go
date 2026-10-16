package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newBinderWithLeaf creates a Collection with one Leaf, so Slots can be placed on it.
func newBinderWithLeaf(t *testing.T, db *gorm.DB, title string) (entity.Collection, entity.BinderLeaf) {
	t.Helper()
	profile := newTestProfile(t, db, title)
	col := entity.Collection{ProfileID: profile.ID, Title: title}
	require.NoError(t, db.Create(&col).Error)
	leaf := entity.BinderLeaf{CollectionID: col.ID, Position: 1}
	require.NoError(t, db.Create(&leaf).Error)
	return col, leaf
}

// placeSlotRow writes one Slot row. Create omits the Card association, as the
// Slot routes' own repository does, so only the card_id column is set.
func placeSlotRow(t *testing.T, db *gorm.DB, col entity.Collection, leaf entity.BinderLeaf, row, column int, cardID *uuid.UUID, unidentified bool) {
	t.Helper()
	slot := entity.BinderSlot{
		CollectionID: col.ID,
		LeafID:       leaf.ID,
		Side:         "front",
		Row:          row,
		Column:       column,
		CardID:       cardID,
		Unidentified: unidentified,
	}
	require.NoError(t, db.Omit("Card").Create(&slot).Error)
}

func TestBinderSlotRepository_CountCardSlots(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewBinderSlotRepository(crud.NewRepository[entity.BinderSlot](db))
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	card := fixture.newCard(t, db, set.ID, nil)
	other := fixture.newCard(t, db, set.ID, func(c *entity.Card) { c.LocalID = "002" })

	binder, leaf := newBinderWithLeaf(t, db, "Slot Count Test")
	placeSlotRow(t, db, binder, leaf, 1, 1, &card.ID, false)
	placeSlotRow(t, db, binder, leaf, 1, 2, &card.ID, false)
	placeSlotRow(t, db, binder, leaf, 2, 1, &other.ID, false)
	placeSlotRow(t, db, binder, leaf, 2, 2, nil, true)

	count, err := repo.CountCardSlots(ctx, binder.ID, card.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "every Slot holding the Card counts, one per Slot")

	count, err = repo.CountCardSlots(ctx, binder.ID, other.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count, err = repo.CountCardSlots(ctx, binder.ID, uuid.New())
	require.NoError(t, err)
	assert.Zero(t, count, "a Card with no Slots counts zero")

	count, err = repo.CountCardSlots(ctx, binder.ID, uuid.Nil)
	require.NoError(t, err)
	assert.Zero(t, count, "a nil Card id must not match the Unidentified Slots")

	// Counts are scoped to one Collection, so the same Card in another binder is not included.
	elsewhere, elsewhereLeaf := newBinderWithLeaf(t, db, "Slot Count Other")
	placeSlotRow(t, db, elsewhere, elsewhereLeaf, 1, 1, &card.ID, false)
	count, err = repo.CountCardSlots(ctx, binder.ID, card.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	count, err = repo.CountCardSlots(ctx, elsewhere.ID, card.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Inside a transaction the count sees that transaction's uncommitted Slot.
	err = crud.NewTransactor(db).WithinTransaction(ctx, func(ctx context.Context) error {
		if _, err := repo.Insert(ctx, entity.BinderSlot{
			CollectionID: binder.ID,
			LeafID:       leaf.ID,
			Side:         "front",
			Row:          2,
			Column:       3,
			CardID:       &card.ID,
		}); err != nil {
			return err
		}
		count, err := repo.CountCardSlots(ctx, binder.ID, card.ID)
		assert.NoError(t, err)
		assert.Equal(t, 3, count)
		return err
	})
	require.NoError(t, err)
}

func TestBinderSlotRepository_ListOccupiedSlots(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewBinderSlotRepository(crud.NewRepository[entity.BinderSlot](db))
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	card := fixture.newCard(t, db, set.ID, nil)
	other := fixture.newCard(t, db, set.ID, func(c *entity.Card) { c.LocalID = "002" })
	unlisted := fixture.newCard(t, db, set.ID, func(c *entity.Card) { c.LocalID = "003" })

	binder, leaf := newBinderWithLeaf(t, db, "List Occupied Test")
	placeSlotRow(t, db, binder, leaf, 1, 1, &card.ID, false)
	placeSlotRow(t, db, binder, leaf, 1, 2, &other.ID, false)
	placeSlotRow(t, db, binder, leaf, 2, 1, nil, true)
	placeSlotRow(t, db, binder, leaf, 2, 2, &unlisted.ID, false)

	slots, err := repo.ListOccupiedSlots(ctx, binder.ID, []uuid.UUID{card.ID})
	require.NoError(t, err)
	require.Len(t, slots, 1, "only the Slots of the listed Card come back")
	assert.Equal(t, card.ID, *slots[0].CardID)
	assert.Equal(t, 1, slots[0].Row)
	assert.Equal(t, 1, slots[0].Column)
	assert.Equal(t, 1, slots[0].Leaf.Position, "the Leaf is preloaded so the Page can be computed")

	slots, err = repo.ListOccupiedSlots(ctx, binder.ID, []uuid.UUID{card.ID, other.ID, unlisted.ID})
	require.NoError(t, err)
	assert.Len(t, slots, 3, "the Unidentified Slot holds no Card, so it never matches")

	slots, err = repo.ListOccupiedSlots(ctx, binder.ID, nil)
	require.NoError(t, err)
	assert.Empty(t, slots, "an entries page with no Cards loads nothing")

	elsewhere, elsewhereLeaf := newBinderWithLeaf(t, db, "List Occupied Other")
	placeSlotRow(t, db, elsewhere, elsewhereLeaf, 1, 1, &card.ID, false)
	slots, err = repo.ListOccupiedSlots(ctx, binder.ID, []uuid.UUID{card.ID})
	require.NoError(t, err)
	assert.Len(t, slots, 1, "Slots of another Collection are not listed")
}

func TestBinderSlotRepository_ListSlotsAt(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewBinderSlotRepository(crud.NewRepository[entity.BinderSlot](db))
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	card := fixture.newCard(t, db, set.ID, nil)

	binder, leaf := newBinderWithLeaf(t, db, "List Slots At Test")
	placeSlotRow(t, db, binder, leaf, 1, 1, &card.ID, false)
	placeSlotRow(t, db, binder, leaf, 1, 2, &card.ID, false)
	placeSlotRow(t, db, binder, leaf, 2, 1, nil, true)

	slots, err := repo.ListSlotsAt(ctx, binder.ID, []BinderSlotAddress{
		{LeafID: leaf.ID, Side: "front", Row: 1, Column: 1},
		{LeafID: leaf.ID, Side: "front", Row: 2, Column: 1},
		{LeafID: leaf.ID, Side: "front", Row: 2, Column: 2},
	})
	require.NoError(t, err)
	require.Len(t, slots, 2, "only the addressed occupied Slots come back; the empty address has no row")
	positions := make([][2]int, 0, len(slots))
	for _, slot := range slots {
		positions = append(positions, [2]int{slot.Row, slot.Column})
	}
	assert.ElementsMatch(t, [][2]int{{1, 1}, {2, 1}}, positions, "the unaddressed Slot (1,2) is not loaded")
}

func TestBinderSlotRepository_ListAllOccupiedSlots(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewBinderSlotRepository(crud.NewRepository[entity.BinderSlot](db))
	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	card := fixture.newCard(t, db, set.ID, nil)

	binder, leaf := newBinderWithLeaf(t, db, "List All Test")
	placeSlotRow(t, db, binder, leaf, 1, 1, &card.ID, false)
	placeSlotRow(t, db, binder, leaf, 1, 2, nil, true)

	slots, err := repo.ListAllOccupiedSlots(ctx, binder.ID)
	require.NoError(t, err)
	assert.Len(t, slots, 2, "Unidentified Slots are occupied too")
	for _, slot := range slots {
		assert.Equal(t, 1, slot.Leaf.Position)
	}
}

// TestBinderSlotRepository_NilIDsMatchNothing: a nil Collection or a nil Card id
// is refused before the query, so it cannot match the Unidentified Slots, whose
// card_id is NULL.
func TestBinderSlotRepository_NilIDsMatchNothing(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewBinderSlotRepository(crud.NewRepository[entity.BinderSlot](db))

	binder, leaf := newBinderWithLeaf(t, db, "Nil Guard Test")
	placeSlotRow(t, db, binder, leaf, 1, 1, nil, true)

	slots, err := repo.ListOccupiedSlots(ctx, uuid.Nil, []uuid.UUID{uuid.Nil})
	require.NoError(t, err)
	assert.Empty(t, slots)

	slots, err = repo.ListOccupiedSlots(ctx, binder.ID, []uuid.UUID{uuid.Nil})
	require.NoError(t, err)
	assert.Empty(t, slots, "a nil Card id must not match the Unidentified Slot")

	slots, err = repo.ListAllOccupiedSlots(ctx, uuid.Nil)
	require.NoError(t, err)
	assert.Empty(t, slots, "a nil Collection id must not list every binder's Slots")
}
