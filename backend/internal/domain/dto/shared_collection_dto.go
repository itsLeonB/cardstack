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
	OwnerName   string        `json:"ownerName" doc:"The owner's display name, never their email."`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Binder      *BinderHeader `json:"binder,omitempty" doc:"The binder's Layout and Unidentified count. Absent for a plain Collection."`
}

// SharedCollectionListRequest pages or faceted-counts the entries of a Shared
// Collection under the same filters as the owner's own listing.
type SharedCollectionListRequest struct {
	Lookup SharedCollectionLookup
	Filter CardFilter
}

// SharedCollectionPageRequest addresses one Page of a Shared binder.
type SharedCollectionPageRequest struct {
	Lookup SharedCollectionLookup
	Page   int
}
