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

// CreateCollectionRequest is the request to create a Collection. Title is
// required; Description and MaxCardCount are optional.
type CreateCollectionRequest struct {
	Title        string
	Description  string
	MaxCardCount int
}

// UpdateCollectionRequest is the request to edit an existing Collection's
// title/description/limit.
type UpdateCollectionRequest struct {
	Title        string
	Description  string
	MaxCardCount int
}
