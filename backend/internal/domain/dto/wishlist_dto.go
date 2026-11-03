package dto

import "github.com/google/uuid"

// The status filter's values (GLOSSARY.md: Wishlist). Fulfilled means the owner
// has at least the wanted quantity in Master Inventory; it is derived on read.
const (
	WishlistStatusAll       = "all"
	WishlistStatusNeeded    = "needed"
	WishlistStatusFulfilled = "fulfilled"
)

// WishlistListRequest lists or facets a profile's Wishlist, filtered like a
// catalog search (Page/Limit are ignored by facets).
type WishlistListRequest struct {
	ProfileID uuid.UUID
	Status    string
	Filter    CardFilter
}

// WishlistItemRequest sets one Card's wanted quantity.
type WishlistItemRequest struct {
	ProfileID uuid.UUID
	CardID    uuid.UUID
	Quantity  int
}

// WishlistItemLookup addresses one Card on a profile's Wishlist.
type WishlistItemLookup struct {
	ProfileID uuid.UUID
	CardID    uuid.UUID
}

// WishlistLookup addresses a profile's whole Wishlist.
type WishlistLookup struct {
	ProfileID uuid.UUID
}

// WishlistItem is one Card on a Wishlist: the Card, the quantity wanted, and
// the quantity the owner has in Master Inventory next to it.
type WishlistItem struct {
	Card           CardSummary `json:"card"`
	WantedQuantity int         `json:"wantedQuantity"`
	OwnedQuantity  int         `json:"ownedQuantity"`
	Fulfilled      bool        `json:"fulfilled" doc:"Whether the owner has at least the wanted quantity in Master Inventory. Derived on read, never stored."`
}

// WishlistClearResult says how many items Clear fulfilled removed.
type WishlistClearResult struct {
	Removed int `json:"removed"`
}

// WishlistPresetRequest is the query for previewing or applying a Wishlist Preset
// (GLOSSARY.md: Wishlist Preset): published Cards of the chosen Expansion Sets,
// optionally narrowed by rarity, category and tag, of which the user owns none in
// Master Inventory.
type WishlistPresetRequest struct {
	ProfileID       uuid.UUID
	ExpansionSetIDs []uuid.UUID
	Filter          CardFilter
	Quantity        int
}

// WishlistPresetResult is the response to previewing a Wishlist Preset: how many
// Cards would be added, and which ones. Apply returns the same shape.
type WishlistPresetResult struct {
	Count int           `json:"count"`
	Cards []CardSummary `json:"cards"`
}

// WishlistSummary is the owner's Wishlist metadata, including its shared status.
type WishlistSummary struct {
	Shared bool `json:"shared" doc:"Whether the Wishlist is shared publicly. False when the profile has no Wishlist row yet."`
}

// WishlistShareResult is the response to sharing a Wishlist: the profile's Slug
// for the shared link.
type WishlistShareResult struct {
	Slug string `json:"slug" doc:"The profile's Slug for the shared link at /s/{slug}/wishlist."`
}
