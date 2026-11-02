package routes

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

// wishlistItem is one row of the Wishlist list as the API renders it.
type wishlistItem struct {
	Card struct {
		ID string `json:"id"`
	} `json:"card"`
	WantedQuantity int  `json:"wantedQuantity"`
	OwnedQuantity  int  `json:"ownedQuantity"`
	Fulfilled      bool `json:"fulfilled"`
}

func (a testAPI) wishlist(t *testing.T, token string, query string) ([]wishlistItem, int) {
	t.Helper()

	resp := a.Get("/wishlist"+query, bearer(token))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var list struct {
		Data []wishlistItem `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &list))

	return list.Data, list.Meta.Total
}

// newUnpublishedCard inserts a Card of an unpublished Expansion Set, which no
// user-facing list may show.
func newUnpublishedCard(t *testing.T, api testAPI) uuid.UUID {
	t.Helper()

	s := uuid.NewString()
	game := entity.Game{Slug: "wl-" + s, Name: "Wl " + s}
	require.NoError(t, api.db.Create(&game).Error)
	locale := entity.Locale{Code: "wl-" + s}
	require.NoError(t, api.db.Create(&locale).Error)
	rarity := entity.Rarity{GameID: game.ID, Code: "WL-" + s, Name: "Rarity " + s}
	require.NoError(t, api.db.Create(&rarity).Error)
	set := entity.ExpansionSet{CreatedBy: entity.SourcePokemonAsia, UpdatedBy: entity.SourcePokemonAsia, GameID: game.ID, Code: "wl-set-" + s, Name: "Wl Set " + s, LocaleID: locale.ID}
	require.NoError(t, api.db.Create(&set).Error)
	// The column defaults to true, so an unpublished Set is made by an update.
	require.NoError(t, api.db.Model(&entity.ExpansionSet{}).Where("id = ?", set.ID).Update("published", false).Error)
	card := entity.Card{CreatedBy: entity.SourcePokemonAsia, UpdatedBy: entity.SourcePokemonAsia, ExpansionSetID: set.ID, LocalID: "1", Name: "Wl Card " + s, Category: "Pokémon",
		Tags: datatypes.JSONSlice[string]{}, RarityID: rarity.ID, Attributes: datatypes.JSONMap{}}
	require.NoError(t, api.db.Create(&card).Error)

	return card.ID
}

// TestWishlistFlow covers the whole Wishlist through the HTTP boundary: set and
// replace a quantity, the derived status and its filter, remove, Clear
// fulfilled, the unpublished-set rule, cross-user isolation, and that no
// Wishlist write ever moves Master Inventory.
func TestWishlistFlow(t *testing.T) {
	api := newTestAPI(t)
	cards := newTestCards(t, 3)
	owner := api.newUserToken(t)
	other := api.newUserToken(t)

	// Private: no token is refused on every verb.
	assert.Equal(t, http.StatusUnauthorized, api.Get("/wishlist").Code)
	assert.Equal(t, http.StatusUnauthorized, api.Put("/wishlist/items/"+cards[0].String(), map[string]any{"quantity": 1}).Code)
	assert.Equal(t, http.StatusUnauthorized, api.Delete("/wishlist/items/"+cards[0].String()).Code)
	assert.Equal(t, http.StatusUnauthorized, api.Delete("/wishlist/fulfilled").Code)

	// Setting a quantity creates the item.
	for _, c := range []struct {
		cardID   uuid.UUID
		quantity int
	}{{cards[0], 2}, {cards[1], 2}, {cards[2], 1}} {
		resp := api.Put("/wishlist/items/"+c.cardID.String(), bearer(owner), map[string]any{"quantity": c.quantity})
		require.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
	}

	// The owner now has 2 of the first Card and 1 of the second; the third is
	// only wanted. Master Inventory must not move.
	createResp := api.Post("/collections", bearer(owner), map[string]any{"title": "Box"})
	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	var created collectionEnvelope
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &created))
	base := "/collections/" + created.Data.ID + "/entries"
	require.Equal(t, http.StatusCreated, api.Post(base, bearer(owner), map[string]any{"cardId": cards[0], "quantity": 2}).Code)
	require.Equal(t, http.StatusCreated, api.Post(base, bearer(owner), map[string]any{"cardId": cards[1], "quantity": 1}).Code)

	items, total := api.wishlist(t, owner, "")
	assert.Equal(t, 3, total)
	require.Len(t, items, 3)
	byID := map[string]wishlistItem{}
	for _, item := range items {
		byID[item.Card.ID] = item
	}
	assert.Equal(t, 2, byID[cards[0].String()].WantedQuantity)
	assert.Equal(t, 2, byID[cards[0].String()].OwnedQuantity)
	assert.True(t, byID[cards[0].String()].Fulfilled)
	assert.Equal(t, 1, byID[cards[1].String()].OwnedQuantity)
	assert.False(t, byID[cards[1].String()].Fulfilled)
	assert.Equal(t, 0, byID[cards[2].String()].OwnedQuantity)
	assert.False(t, byID[cards[2].String()].Fulfilled)

	// The status filter is derived from owned against wanted.
	fulfilled, total := api.wishlist(t, owner, "?status=fulfilled")
	assert.Equal(t, 1, total)
	require.Len(t, fulfilled, 1)
	assert.Equal(t, cards[0].String(), fulfilled[0].Card.ID)

	needed, total := api.wishlist(t, owner, "?status=needed")
	assert.Equal(t, 2, total)
	require.Len(t, needed, 2)

	// The shared card filters combine with the status.
	filtered, total := api.wishlist(t, owner, "?status=needed&cardId="+cards[1].String())
	assert.Equal(t, 1, total)
	require.Len(t, filtered, 1)
	assert.Equal(t, cards[1].String(), filtered[0].Card.ID)

	// Setting the quantity again replaces it instead of adding a second item.
	resp := api.Put("/wishlist/items/"+cards[0].String(), bearer(owner), map[string]any{"quantity": 3})
	require.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
	items, total = api.wishlist(t, owner, "")
	assert.Equal(t, 3, total)
	for _, item := range items {
		if item.Card.ID == cards[0].String() {
			assert.Equal(t, 3, item.WantedQuantity)
			assert.False(t, item.Fulfilled, "3 wanted against 2 owned is no longer fulfilled")
		}
	}

	// A wishlist write never moves Master Inventory: still the two owned Cards.
	resp = api.Get("/inventory/cards", bearer(owner))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var master struct {
		Data []struct {
			Card     struct{ ID string } `json:"card"`
			Quantity int                 `json:"quantity"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &master))
	owned := map[string]int{}
	for _, item := range master.Data {
		owned[item.Card.ID] = item.Quantity
	}
	assert.Equal(t, map[string]int{cards[0].String(): 2, cards[1].String(): 1}, owned)

	// Another user sees and changes nothing of it.
	theirItems, theirTotal := api.wishlist(t, other, "")
	assert.Equal(t, 0, theirTotal)
	assert.Empty(t, theirItems)
	assert.Equal(t, http.StatusNotFound, api.Delete("/wishlist/items/"+cards[0].String(), bearer(other)).Code)
	require.Equal(t, http.StatusNoContent, api.Put("/wishlist/items/"+cards[2].String(), bearer(other), map[string]any{"quantity": 1}).Code)
	_, ownerTotal := api.wishlist(t, owner, "")
	assert.Equal(t, 3, ownerTotal)
	_, otherTotal := api.wishlist(t, other, "")
	assert.Equal(t, 1, otherTotal)

	// Removing an item removes only that item.
	assert.Equal(t, http.StatusNoContent, api.Delete("/wishlist/items/"+cards[2].String(), bearer(owner)).Code)
	assert.Equal(t, http.StatusNotFound, api.Delete("/wishlist/items/"+cards[2].String(), bearer(owner)).Code)
	_, total = api.wishlist(t, owner, "")
	assert.Equal(t, 2, total)

	// Clear fulfilled removes the fulfilled item only, and says how many went.
	resp = api.Delete("/wishlist/fulfilled", bearer(owner))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var cleared struct {
		Data struct {
			Removed int `json:"removed"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &cleared))
	assert.Equal(t, 0, cleared.Data.Removed, "the only fulfilled item was replaced by a larger wanted quantity above")
	assert.Equal(t, http.StatusOK, api.Delete("/wishlist/fulfilled", bearer(owner)).Code)
}

// TestWishlistClearFulfilled covers Clear fulfilled on its own: a fulfilled item
// goes, a still-needed one stays, and another profile keeps its own item.
func TestWishlistClearFulfilled(t *testing.T) {
	api := newTestAPI(t)
	cards := newTestCards(t, 2)
	owner := api.newUserToken(t)
	other := api.newUserToken(t)

	for _, c := range []struct {
		cardID   uuid.UUID
		quantity int
	}{{cards[0], 1}, {cards[1], 2}} {
		require.Equal(t, http.StatusNoContent, api.Put("/wishlist/items/"+c.cardID.String(), bearer(owner), map[string]any{"quantity": c.quantity}).Code)
		require.Equal(t, http.StatusNoContent, api.Put("/wishlist/items/"+c.cardID.String(), bearer(other), map[string]any{"quantity": c.quantity}).Code)
	}

	createResp := api.Post("/collections", bearer(owner), map[string]any{"title": "Box"})
	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	var created collectionEnvelope
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &created))
	require.Equal(t, http.StatusCreated, api.Post("/collections/"+created.Data.ID+"/entries", bearer(owner), map[string]any{"cardId": cards[0], "quantity": 1}).Code)

	resp := api.Delete("/wishlist/fulfilled", bearer(owner))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var cleared struct {
		Data struct {
			Removed int `json:"removed"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &cleared))
	assert.Equal(t, 1, cleared.Data.Removed)

	items, total := api.wishlist(t, owner, "")
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, cards[1].String(), items[0].Card.ID)

	// The other profile owns nothing, so its items are all still needed.
	_, total = api.wishlist(t, other, "")
	assert.Equal(t, 2, total)
}

// TestWishlistRefusesUnpublishedAndUnknownCards: a Card of an unpublished
// Expansion Set cannot be added and is never listed, unknown Cards are 404, and
// a wanted quantity below 1 is refused before any write.
func TestWishlistRefusesUnpublishedAndUnknownCards(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	hidden := newUnpublishedCard(t, api)
	published := newTestCards(t, 1)[0]

	resp := api.Put("/wishlist/items/"+hidden.String(), bearer(owner), map[string]any{"quantity": 1})
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	resp = api.Put("/wishlist/items/"+uuid.NewString(), bearer(owner), map[string]any{"quantity": 1})
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	resp = api.Put("/wishlist/items/"+published.String(), bearer(owner), map[string]any{"quantity": 0})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())

	_, total := api.wishlist(t, owner, "")
	assert.Equal(t, 0, total)
}

// TestWishlistFacets: the facet options come only from the Wishlist's own Cards.
func TestWishlistFacets(t *testing.T) {
	api := newTestAPI(t)
	cards := newTestCards(t, 1)
	owner := api.newUserToken(t)

	require.Equal(t, http.StatusNoContent, api.Put("/wishlist/items/"+cards[0].String(), bearer(owner), map[string]any{"quantity": 1}).Code)
	// A published Card the profile does not want must not contribute a facet.
	newTestCards(t, 1)

	resp := api.Get("/wishlist/facets", bearer(owner))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var facets struct {
		Data struct {
			ExpansionSets []struct {
				ID string `json:"id"`
			} `json:"expansionSets"`
			Categories []struct {
				Value string `json:"value"`
			} `json:"categories"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &facets))
	require.Len(t, facets.Data.ExpansionSets, 1)
	require.Len(t, facets.Data.Categories, 1)
	assert.Equal(t, "Pokémon", facets.Data.Categories[0].Value)
}
