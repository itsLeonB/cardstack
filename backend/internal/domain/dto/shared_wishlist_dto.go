package dto

import "github.com/google/uuid"

// SharedWishlistLookup addresses a Shared Wishlist by its profile Slug
// (docs/adr/0025). There is one Wishlist per profile at a fixed path.
type SharedWishlistLookup struct {
	ProfileSlug string
}

// SharedWishlistSummary is a Shared Wishlist's header as any viewer sees it.
// OwnerName is the owner's display name; their email is never included.
type SharedWishlistSummary struct {
	OwnerName string `json:"ownerName" doc:"The owner's display name, never their email."`
}

// SharedWishlistListRequest pages or faceted-counts the items of a Shared
// Wishlist under the same filters as the owner's own listing.
type SharedWishlistListRequest struct {
	Lookup          SharedWishlistLookup
	Filter          CardFilter
	Status          string // all, needed, fulfilled
	ViewerProfileID uuid.UUID
	ViewerFilter    string // "have" (owned > 0) or "dont-have" (owned == 0), empty means no viewer filter
}
