package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type wishlistShareEnvelope struct {
	Data struct {
		Slug string `json:"slug"`
	} `json:"data"`
}

type sharedWishlistHeaderEnvelope struct {
	Data struct {
		OwnerName string `json:"ownerName"`
	} `json:"data"`
}

type sharedWishlistEntriesEnvelope struct {
	Data []struct {
		Card struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"card"`
		WantedQuantity int  `json:"wantedQuantity"`
		OwnedQuantity  int  `json:"ownedQuantity"`
		Fulfilled      bool `json:"fulfilled"`
	} `json:"data"`
	Meta struct{ Total, Page, Limit int } `json:"meta"`
}

type sharedWishlistFacetsEnvelope struct {
	Data struct {
		Rarities   []struct{ Value string } `json:"rarities"`
		Categories []struct{ Value string } `json:"categories"`
		Tags       []struct{ Value string } `json:"tags"`
		Sets       []struct{ Value string } `json:"sets"`
	} `json:"data"`
}

// wishlistMissShape is every part of a response a caller can see, except the
// per-request headers (Date and the request id), which differ on each call.
func wishlistMissShape(resp *httptest.ResponseRecorder) string {
	headers := resp.Result().Header.Clone()
	headers.Del("Date")
	headers.Del("X-Request-Id")
	return fmt.Sprintf("%d\n%v\n%s", resp.Code, headers, resp.Body.String())
}

// addWishlistItem adds a Card to the owner's Wishlist with the given quantity.
func addWishlistItem(t *testing.T, api testAPI, token, ownerEmail string, cardID uuid.UUID, quantity int) {
	t.Helper()
	resp := api.Put("/wishlist/items/"+cardID.String(), bearer(token), map[string]any{"quantity": quantity})
	require.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
}

// TestShareWishlist turns sharing on and returns the profile slug for the link.
func TestShareWishlist(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)

	resp := api.Post("/wishlist/share", bearer(owner.token), map[string]any{})
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

	var share wishlistShareEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &share))
	require.NotEmpty(t, share.Data.Slug, "share returns the profile slug")

	// Sharing again while already shared keeps the same slug
	again := api.Post("/wishlist/share", bearer(owner.token), map[string]any{})
	require.Equal(t, http.StatusOK, again.Code, again.Body.String())
	var againShare wishlistShareEnvelope
	require.NoError(t, json.Unmarshal(again.Body.Bytes(), &againShare))
	require.Equal(t, share.Data.Slug, againShare.Data.Slug)
}

// TestUnshareWishlist turns sharing off and the link stops working.
func TestUnshareWishlist(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)

	share := api.Post("/wishlist/share", bearer(owner.token), map[string]any{})
	require.Equal(t, http.StatusOK, share.Code, share.Body.String())
	var shareData wishlistShareEnvelope
	require.NoError(t, json.Unmarshal(share.Body.Bytes(), &shareData))
	profile := api.profileSlug(t, owner.email)

	// The shared link works
	require.Equal(t, http.StatusOK, api.Get("/shared/"+profile+"/wishlist").Code)

	// Unshare
	unshare := api.Delete("/wishlist/share", bearer(owner.token))
	require.Equal(t, http.StatusNoContent, unshare.Code, unshare.Body.String())

	// The link stops working
	require.Equal(t, http.StatusNotFound, api.Get("/shared/"+profile+"/wishlist").Code)

	// Unsharing an already private wishlist still succeeds
	again := api.Delete("/wishlist/share", bearer(owner.token))
	require.Equal(t, http.StatusNoContent, again.Code, again.Body.String())
}

// TestSharedWishlistGuestView covers that a guest can view a shared wishlist
// with actual items, their Cards, wanted/owned quantities, and fulfilled status.
func TestSharedWishlistGuestView(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)

	// Seed some cards and add them to the owner's wishlist
	pokemon := seedSharedSet(t, api, true, "Pokémon", "wl-tag-"+uuid.NewString(), 2)
	trainer := seedSharedSet(t, api, true, "Trainer", "wl-tag-"+uuid.NewString(), 1)

	// Add items to the wishlist
	addWishlistItem(t, api, owner.token, owner.email, pokemon.cards[0], 3) // wanted 3, owned 0 -> not fulfilled
	addWishlistItem(t, api, owner.token, owner.email, pokemon.cards[1], 1) // wanted 1, owned 0 -> not fulfilled
	addWishlistItem(t, api, owner.token, owner.email, trainer.cards[0], 2) // wanted 2, owned 0 -> not fulfilled

	// Add some cards to master inventory so they become fulfilled
	createResp := api.Post("/collections", bearer(owner.token), map[string]any{"title": "Master Inv"})
	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	var created collectionEnvelope
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &created))
	base := "/collections/" + created.Data.ID + "/entries"
	// Add 3 of pokemon.cards[0] to inventory -> fulfills the wishlist item
	api.Post(base, bearer(owner.token), map[string]any{"cardId": pokemon.cards[0], "quantity": 3})
	// Add 2 of trainer.cards[0] to inventory -> fulfills the wishlist item
	api.Post(base, bearer(owner.token), map[string]any{"cardId": trainer.cards[0], "quantity": 2})

	share := api.Post("/wishlist/share", bearer(owner.token), map[string]any{})
	require.Equal(t, http.StatusOK, share.Code, share.Body.String())
	var shareData wishlistShareEnvelope
	require.NoError(t, json.Unmarshal(share.Body.Bytes(), &shareData))
	profile := api.profileSlug(t, owner.email)

	// Guest can get the header
	resp := api.Get("/shared/" + profile + "/wishlist")
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var header sharedWishlistHeaderEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &header))
	require.NotEmpty(t, header.Data.OwnerName, "guest sees owner name, not email")
	assert.NotContains(t, resp.Body.String(), owner.email, "response never contains owner's email")

	// Guest can list entries with actual items
	entries := api.Get("/shared/" + profile + "/wishlist/entries")
	require.Equal(t, http.StatusOK, entries.Code, entries.Body.String())
	var entriesData sharedWishlistEntriesEnvelope
	require.NoError(t, json.Unmarshal(entries.Body.Bytes(), &entriesData))
	require.Len(t, entriesData.Data, 3, "wishlist has 3 items")
	assert.NotContains(t, entries.Body.String(), owner.email, "entries response never contains owner's email")

	// Verify each item has Card details, wanted quantity, owned quantity, and fulfilled status
	found := make(map[string]struct {
		cardID, cardName string
		wanted, owned    int
		fulfilled        bool
	})
	for _, item := range entriesData.Data {
		found[item.Card.ID] = struct {
			cardID, cardName string
			wanted, owned    int
			fulfilled        bool
		}{
			cardID:    item.Card.ID,
			cardName:  item.Card.Name,
			wanted:    item.WantedQuantity,
			owned:     item.OwnedQuantity,
			fulfilled: item.Fulfilled,
		}
	}

	// Check specific items
	assert.Contains(t, found, pokemon.cards[0].String())
	assert.Equal(t, 3, found[pokemon.cards[0].String()].wanted)
	assert.Equal(t, 3, found[pokemon.cards[0].String()].owned)
	assert.True(t, found[pokemon.cards[0].String()].fulfilled, "pokemon[0] has 3 owned and 3 wanted")

	assert.Contains(t, found, pokemon.cards[1].String())
	assert.Equal(t, 1, found[pokemon.cards[1].String()].wanted)
	assert.Equal(t, 0, found[pokemon.cards[1].String()].owned)
	assert.False(t, found[pokemon.cards[1].String()].fulfilled, "pokemon[1] has 0 owned and 1 wanted")

	assert.Contains(t, found, trainer.cards[0].String())
	assert.Equal(t, 2, found[trainer.cards[0].String()].wanted)
	assert.Equal(t, 2, found[trainer.cards[0].String()].owned)
	assert.True(t, found[trainer.cards[0].String()].fulfilled, "trainer[0] has 2 owned and 2 wanted")

	// Guest can list facets
	facets := api.Get("/shared/" + profile + "/wishlist/facets")
	require.Equal(t, http.StatusOK, facets.Code, facets.Body.String())
	assert.NotContains(t, facets.Body.String(), owner.email, "facets response never contains owner's email")
}

// TestSharedWishlistEntriesUseTheOwnersFiltersAndFacetsForAGuest covers that a Guest
// is not held to the catalog's preview lock on a Shared Wishlist, so later pages
// and the rarity, category, tag, and status filters all answer.
func TestSharedWishlistEntriesUseTheOwnersFiltersAndFacetsForAGuest(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)
	tag := "wl-filter-tag-" + uuid.NewString()
	pokemon := seedSharedSet(t, api, true, "Pokémon", tag, 2)
	trainer := seedSharedSet(t, api, true, "Trainer", tag, 1)

	// Add items to wishlist
	addWishlistItem(t, api, owner.token, owner.email, pokemon.cards[0], 3) // wanted 3, owned 0 -> not fulfilled
	addWishlistItem(t, api, owner.token, owner.email, pokemon.cards[1], 1) // wanted 1, owned 0 -> not fulfilled
	addWishlistItem(t, api, owner.token, owner.email, trainer.cards[0], 2) // wanted 2, owned 0 -> not fulfilled

	// Add to master inventory to create fulfilled items
	createResp := api.Post("/collections", bearer(owner.token), map[string]any{"title": "Master Inv"})
	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	var created collectionEnvelope
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &created))
	base := "/collections/" + created.Data.ID + "/entries"
	api.Post(base, bearer(owner.token), map[string]any{"cardId": pokemon.cards[0], "quantity": 3}) // fulfills
	api.Post(base, bearer(owner.token), map[string]any{"cardId": trainer.cards[0], "quantity": 1}) // partially fulfills

	// Share the wishlist
	api.Post("/wishlist/share", bearer(owner.token), map[string]any{})
	profile := api.profileSlug(t, owner.email)
	wlBase := "/shared/" + profile + "/wishlist/entries"

	list := func(query url.Values) sharedWishlistEntriesEnvelope {
		t.Helper()
		resp := api.Get(wlBase + "?" + query.Encode())
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		var out sharedWishlistEntriesEnvelope
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
		assert.NotContains(t, resp.Body.String(), owner.email, "response never contains owner's email")
		return out
	}

	// All items
	all := list(url.Values{})
	assert.Equal(t, 3, all.Meta.Total)

	// Filter by category
	byCategory := list(url.Values{"category": {"Pokémon"}})
	require.Len(t, byCategory.Data, 2)
	assert.Equal(t, pokemon.cards[0].String(), byCategory.Data[0].Card.ID)
	assert.Equal(t, pokemon.cards[1].String(), byCategory.Data[1].Card.ID)

	// Filter by tag
	byTag := list(url.Values{"tag": {tag}})
	assert.Equal(t, 3, byTag.Meta.Total)

	// Filter by rarity
	byRarity := list(url.Values{"rarityId": {pokemon.rarity.String()}})
	require.Len(t, byRarity.Data, 2)

	// Filter by status: fulfilled
	fulfilled := list(url.Values{"status": {"fulfilled"}})
	require.Len(t, fulfilled.Data, 1)
	assert.Equal(t, pokemon.cards[0].String(), fulfilled.Data[0].Card.ID)
	assert.True(t, fulfilled.Data[0].Fulfilled)

	// Filter by status: needed
	needed := list(url.Values{"status": {"needed"}})
	require.Len(t, needed.Data, 2)
	// pokemon.cards[1] and trainer.cards[0] are not fully fulfilled
	cardIDs := make([]string, len(needed.Data))
	for i, item := range needed.Data {
		cardIDs[i] = item.Card.ID
	}
	assert.Contains(t, cardIDs, pokemon.cards[1].String())
	assert.Contains(t, cardIDs, trainer.cards[0].String())

	// Filter by status: all
	allStatus := list(url.Values{"status": {"all"}})
	assert.Equal(t, 3, allStatus.Meta.Total)

	// Test facets (use the correct base path)
	facetsBase := "/shared/" + profile + "/wishlist/facets"
	facets := api.Get(facetsBase)
	require.Equal(t, http.StatusOK, facets.Code, facets.Body.String())
	assert.NotContains(t, facets.Body.String(), owner.email, "facets never contains owner's email")
	var facetsData sharedWishlistFacetsEnvelope
	require.NoError(t, json.Unmarshal(facets.Body.Bytes(), &facetsData))
	assert.ElementsMatch(t, []string{"Pokémon", "Trainer"},
		[]string{facetsData.Data.Categories[0].Value, facetsData.Data.Categories[1].Value})
}

// TestSharedWishlistNotFound covers that a private, unshared, or nonexistent
// wishlist returns the same 404 (ADR-0025), and compares the full bodies against
// each other and against the shared Collection's not-found body.
func TestSharedWishlistNotFound(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)
	// Make a request to ensure the profile exists
	api.Get("/profile", bearer(owner.token))
	profile := api.profileSlug(t, owner.email)

	// Get the reference not-found from a shared Collection endpoint
	refPath := "/shared/" + profile + "/no-such-binder-00000"
	refResp := api.Get(refPath)
	require.Equal(t, http.StatusNotFound, refResp.Code)
	refShape := wishlistMissShape(refResp)

	// Private wishlist (never shared) returns the same 404
	private := api.Get("/shared/" + profile + "/wishlist")
	require.Equal(t, http.StatusNotFound, private.Code)
	assert.Equal(t, refShape, wishlistMissShape(private), "private wishlist not-found matches collection not-found")

	// Nonexistent profile returns 404
	nonexistent := api.Get("/shared/nonexistent-user-slug/wishlist")
	require.Equal(t, http.StatusNotFound, nonexistent.Code)
	assert.Equal(t, refShape, wishlistMissShape(nonexistent), "nonexistent profile not-found matches collection not-found")

	// After sharing and unsharing, the link is dead
	api.Post("/wishlist/share", bearer(owner.token), map[string]any{})
	api.Delete("/wishlist/share", bearer(owner.token))
	unshared := api.Get("/shared/" + profile + "/wishlist")
	require.Equal(t, http.StatusNotFound, unshared.Code)
	assert.Equal(t, refShape, wishlistMissShape(unshared), "unshared wishlist not-found matches collection not-found")

	// Test all the sub-endpoints too
	for _, suffix := range []string{"/entries", "/facets"} {
		privateSub := api.Get("/shared/" + profile + "/wishlist" + suffix)
		require.Equal(t, http.StatusNotFound, privateSub.Code)
		assert.Equal(t, refShape, wishlistMissShape(privateSub), "private wishlist not-found on %s matches", suffix)

		nonexistentSub := api.Get("/shared/nonexistent-user-slug/wishlist" + suffix)
		require.Equal(t, http.StatusNotFound, nonexistentSub.Code)
		assert.Equal(t, refShape, wishlistMissShape(nonexistentSub), "nonexistent profile not-found on %s matches", suffix)
	}
}
