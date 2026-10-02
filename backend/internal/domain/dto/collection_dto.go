package dto

import "github.com/google/uuid"

// CollectionSummary is one Collection as returned to its owner.
// MaxCardCount is the optional hard cap on the collection's summed
// Inventory Entry quantities; 0 means no limit. CardCount is the summed
// quantity of the collection's Inventory Entries (the same measure the limit
// caps, not distinct cards); 0 when empty.
type CollectionSummary struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	MaxCardCount int       `json:"maxCardCount"`
	CardCount    int       `json:"cardCount"`
}

// CollectionRequest is the request to create or edit a Collection. ID is set
// only for an edit. Title is required; Description and MaxCardCount are
// optional.
type CollectionRequest struct {
	ProfileID    uuid.UUID
	ID           uuid.UUID
	Title        string
	Description  string
	MaxCardCount int
}

// CollectionListRequest lists a profile's Collections.
type CollectionListRequest struct {
	ProfileID uuid.UUID
}

// CollectionLookup addresses one of a profile's Collections.
type CollectionLookup struct {
	ProfileID uuid.UUID
	ID        uuid.UUID
}
