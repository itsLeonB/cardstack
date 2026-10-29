package dto

// SharedCollectionLookup addresses a Shared Collection by its profile Slug and
// its Collection Slug (docs/adr/0025).
type SharedCollectionLookup struct {
	ProfileSlug    string
	CollectionSlug string
}

// SharedCollectionSummary is a Shared Collection's header as any viewer sees
// it. OwnerName is the owner's display name; their email is never included.
type SharedCollectionSummary struct {
	OwnerName   string `json:"ownerName" doc:"The owner's display name, never their email."`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// SharedCollectionListRequest pages or faceted-counts the entries of a Shared
// Collection under the same filters as the owner's own listing.
type SharedCollectionListRequest struct {
	Lookup SharedCollectionLookup
	Filter CardFilter
}
