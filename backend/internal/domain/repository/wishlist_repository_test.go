package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// wishlistFixture is a catalog fixture plus the two profiles a Wishlist test
// needs: the owner and someone else.
type wishlistFixture struct {
	catalogFixture
	owner entity.UserProfile
	other entity.UserProfile
}

func newWishlistFixture(t *testing.T, db *gorm.DB) wishlistFixture {
	t.Helper()

	return wishlistFixture{
		catalogFixture: newCatalogFixture(t, db),
		owner:          newTestProfile(t, db, "Wishlist Owner"),
		other:          newTestProfile(t, db, "Wishlist Other"),
	}
}

// want puts the Card on the profile's Wishlist with the wanted quantity.
func (f wishlistFixture) want(t *testing.T, db *gorm.DB, profileID, cardID uuid.UUID, wanted int) {
	t.Helper()

	require.NoError(t, db.Create(&entity.WishlistItem{ProfileID: profileID, CardID: cardID, WantedQuantity: wanted}).Error)
}

// own puts quantity copies of the Card in one of the profile's Collections.
func (f wishlistFixture) own(t *testing.T, db *gorm.DB, profileID, cardID uuid.UUID, quantity int) {
	t.Helper()

	c := entity.Collection{ProfileID: profileID, Title: uuid.NewString()}
	require.NoError(t, db.Create(&c).Error)
	require.NoError(t, db.Create(&entity.InventoryEntry{CollectionID: c.ID, CardID: cardID, Quantity: quantity}).Error)
}

func TestWishlistRepository_List(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	fixture := newWishlistFixture(t, db)
	repo := NewWishlistRepository(crud.NewRepository[entity.WishlistItem](db))
	set := fixture.newExpansionSet(t, db, nil, nil)
	fulfilled := fixture.newCard(t, db, set.ID, func(c *entity.Card) { c.LocalID = "1"; c.Category = "Trainer" })
	needed := fixture.newCard(t, db, set.ID, func(c *entity.Card) { c.LocalID = "2"; c.Category = "Pokémon" })
	unowned := fixture.newCard(t, db, set.ID, func(c *entity.Card) { c.LocalID = "3"; c.Category = "Pokémon" })

	fixture.want(t, db, fixture.owner.ID, fulfilled.ID, 2)
	fixture.want(t, db, fixture.owner.ID, needed.ID, 3)
	fixture.want(t, db, fixture.owner.ID, unowned.ID, 1)
	fixture.want(t, db, fixture.other.ID, unowned.ID, 5)
	fixture.own(t, db, fixture.owner.ID, fulfilled.ID, 4)
	fixture.own(t, db, fixture.owner.ID, needed.ID, 1)

	all, total, err := repo.List(ctx, WishlistFilter{ProfileID: fixture.owner.ID, Cards: CardFilter{Limit: 10}})
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	require.Len(t, all, 3)
	// Ordered by local id within the Set.
	assert.Equal(t, []uuid.UUID{fulfilled.ID, needed.ID, unowned.ID}, []uuid.UUID{all[0].ID, all[1].ID, all[2].ID})
	assert.Equal(t, []int{2, 3, 1}, []int{all[0].WantedQuantity, all[1].WantedQuantity, all[2].WantedQuantity})
	assert.Equal(t, []int{4, 1, 0}, []int{all[0].Quantity, all[1].Quantity, all[2].Quantity})

	// The status filter is derived from owned against wanted.
	fulfilledOnly, total, err := repo.List(ctx, WishlistFilter{ProfileID: fixture.owner.ID, Status: WishlistStatusFulfilled, Cards: CardFilter{Limit: 10}})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, fulfilledOnly, 1)
	assert.Equal(t, fulfilled.ID, fulfilledOnly[0].ID)

	neededOnly, total, err := repo.List(ctx, WishlistFilter{ProfileID: fixture.owner.ID, Status: WishlistStatusNeeded, Cards: CardFilter{Limit: 10}})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, neededOnly, 2)
	assert.Equal(t, []uuid.UUID{needed.ID, unowned.ID}, []uuid.UUID{neededOnly[0].ID, neededOnly[1].ID})

	// The shared card filters combine with the owner scope.
	filtered, total, err := repo.List(ctx, WishlistFilter{ProfileID: fixture.owner.ID, Cards: CardFilter{Categories: []string{"Pokémon"}, Limit: 10}})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, filtered, 2)

	filtered, total, err = repo.List(ctx, WishlistFilter{ProfileID: fixture.owner.ID, Status: WishlistStatusNeeded, Cards: CardFilter{Name: "Catalog Test Card", Limit: 10}})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, filtered, 2)

	// Another profile's Wishlist is never in scope, whatever the filter.
	them, total, err := repo.List(ctx, WishlistFilter{ProfileID: fixture.other.ID, Cards: CardFilter{Limit: 10}})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, them, 1)
	assert.Equal(t, unowned.ID, them[0].ID)
	assert.Equal(t, 5, them[0].WantedQuantity)
	assert.Equal(t, 0, them[0].Quantity)
}

func TestWishlistRepository_List_ExcludesUnpublishedSets(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	fixture := newWishlistFixture(t, db)
	repo := NewWishlistRepository(crud.NewRepository[entity.WishlistItem](db))
	hidden := fixture.newExpansionSet(t, db, nil, nil)
	require.NoError(t, db.Model(&entity.ExpansionSet{}).Where("id = ?", hidden.ID).Update("published", false).Error)
	card := fixture.newCard(t, db, hidden.ID, nil)
	fixture.want(t, db, fixture.owner.ID, card.ID, 1)

	results, total, err := repo.List(ctx, WishlistFilter{ProfileID: fixture.owner.ID, Cards: CardFilter{Limit: 10}})
	require.NoError(t, err)
	assert.EqualValues(t, 0, total)
	assert.Empty(t, results)
}

// TestWishlistRepository_Constraints: the table itself refuses a second item
// for the same (profile, Card) and a wanted quantity below 1, whatever the
// service does.
func TestWishlistRepository_Constraints(t *testing.T) {
	db := testDB(t)
	fixture := newWishlistFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	card := fixture.newCard(t, db, set.ID, nil)

	fixture.want(t, db, fixture.owner.ID, card.ID, 2)
	assert.Error(t, db.Create(&entity.WishlistItem{ProfileID: fixture.owner.ID, CardID: card.ID, WantedQuantity: 4}).Error)
	assert.Error(t, db.Create(&entity.WishlistItem{ProfileID: fixture.owner.ID, CardID: card.ID, WantedQuantity: 0}).Error)

	// Two profiles can want the same Card.
	fixture.want(t, db, fixture.other.ID, card.ID, 1)
}

func TestWishlistRepository_DeleteFulfilled(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	fixture := newWishlistFixture(t, db)
	repo := NewWishlistRepository(crud.NewRepository[entity.WishlistItem](db))
	set := fixture.newExpansionSet(t, db, nil, nil)
	done := fixture.newCard(t, db, set.ID, func(c *entity.Card) { c.LocalID = "1" })
	stillNeeded := fixture.newCard(t, db, set.ID, func(c *entity.Card) { c.LocalID = "2" })
	theirs := fixture.newCard(t, db, set.ID, func(c *entity.Card) { c.LocalID = "3" })

	fixture.want(t, db, fixture.owner.ID, done.ID, 2)
	fixture.want(t, db, fixture.owner.ID, stillNeeded.ID, 2)
	fixture.own(t, db, fixture.owner.ID, done.ID, 2)
	fixture.own(t, db, fixture.owner.ID, stillNeeded.ID, 1)
	// Fulfilled for someone else, who is out of scope.
	fixture.want(t, db, fixture.other.ID, theirs.ID, 1)
	fixture.own(t, db, fixture.other.ID, theirs.ID, 1)

	removed, err := repo.DeleteFulfilled(ctx, fixture.owner.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)

	remaining, total, err := repo.List(ctx, WishlistFilter{ProfileID: fixture.owner.ID, Cards: CardFilter{Limit: 10}})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, remaining, 1)
	assert.Equal(t, stillNeeded.ID, remaining[0].ID)

	// A second Clear fulfilled finds nothing left.
	removed, err = repo.DeleteFulfilled(ctx, fixture.owner.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, removed)

	// The other profile's fulfilled item is untouched.
	_, total, err = repo.List(ctx, WishlistFilter{ProfileID: fixture.other.ID, Cards: CardFilter{Limit: 10}})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
}

func TestWishlistRepository_ListFacets(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	fixture := newWishlistFixture(t, db)
	repo := NewWishlistRepository(crud.NewRepository[entity.WishlistItem](db))
	set := fixture.newExpansionSet(t, db, nil, nil)
	wanted := fixture.newCard(t, db, set.ID, func(c *entity.Card) { c.Category = "Trainer"; c.Tags = datatypes.JSONSlice[string]{"Item"} })
	fixture.newCard(t, db, set.ID, func(c *entity.Card) {
		c.LocalID = "2"
		c.Category = "Energi"
		c.Tags = datatypes.JSONSlice[string]{"Energi Dasar"}
	})
	fixture.want(t, db, fixture.owner.ID, wanted.ID, 1)

	facets, err := repo.ListFacets(ctx, WishlistFilter{ProfileID: fixture.owner.ID})
	require.NoError(t, err)
	require.Len(t, facets.Categories, 1)
	assert.Equal(t, "Trainer", facets.Categories[0].Value)
	require.Len(t, facets.Tags, 1)
	assert.Equal(t, "Item", facets.Tags[0].Value)
	require.Len(t, facets.ExpansionSets, 1)
	assert.Equal(t, set.ID, facets.ExpansionSets[0].ID)
}
