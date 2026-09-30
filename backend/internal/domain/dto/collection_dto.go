package dto

import "github.com/google/uuid"

// CollectionSummary is one Collection as returned to its owner.
// MaxCardCount is the optional hard cap on the collection's summed
// Inventory Entry quantities; 0 means no limit.
type CollectionSummary struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	MaxCardCount int       `json:"maxCardCount"`
}

// CollectionRequest is the request to create or edit a Collection. Title is
// required; Description and MaxCardCount are optional.
type CollectionRequest struct {
	Title        string
	Description  string
	MaxCardCount int
}
