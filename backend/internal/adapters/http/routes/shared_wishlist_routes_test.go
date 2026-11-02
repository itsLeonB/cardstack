package routes

import (
	"encoding/json"
	"net/http"
	"testing"

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
			ID string `json:"id"`
		} `json:"card"`
		WantedQuantity int  `json:"wantedQuantity"`
		OwnedQuantity  int  `json:"ownedQuantity"`
		Fulfilled      bool `json:"fulfilled"`
	} `json:"data"`
	Meta struct{ Total, Page, Limit int } `json:"meta"`
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

// TestSharedWishlistGuestView covers that a guest can view a shared wishlist.
func TestSharedWishlistGuestView(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)

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

	// Guest can list entries (empty wishlist)
	entries := api.Get("/shared/" + profile + "/wishlist/entries")
	require.Equal(t, http.StatusOK, entries.Code, entries.Body.String())
	var entriesData sharedWishlistEntriesEnvelope
	require.NoError(t, json.Unmarshal(entries.Body.Bytes(), &entriesData))
	require.Empty(t, entriesData.Data, "empty wishlist returns empty list")

	// Guest can list facets
	facets := api.Get("/shared/" + profile + "/wishlist/facets")
	require.Equal(t, http.StatusOK, facets.Code, facets.Body.String())
}

// TestSharedWishlistNotFound covers that a private, unshared, or nonexistent
// wishlist returns the same 404 (ADR-0025).
func TestSharedWishlistNotFound(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)
	// Make a request to ensure the profile exists
	api.Get("/profile", bearer(owner.token))
	profile := api.profileSlug(t, owner.email)

	// Private wishlist (never shared) returns 404
	require.Equal(t, http.StatusNotFound, api.Get("/shared/"+profile+"/wishlist").Code)

	// Nonexistent profile returns 404 (use a slug that won't match any profile)
	require.Equal(t, http.StatusNotFound, api.Get("/shared/nonexistent-user-slug/wishlist").Code)

	// After sharing and unsharing, the link is dead
	api.Post("/wishlist/share", bearer(owner.token), map[string]any{})
	api.Delete("/wishlist/share", bearer(owner.token))
	require.Equal(t, http.StatusNotFound, api.Get("/shared/"+profile+"/wishlist").Code)
}
