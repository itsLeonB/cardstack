package dto

import (
	"github.com/google/uuid"
)

// ScanFeedbackBatch is the rows of one confirmed Draft Addition or Page Review.
// ProfileID is the caller's, and it is the prefix every image key must sit under.
type ScanFeedbackBatch struct {
	ProfileID uuid.UUID
	Rows      []ScanFeedbackRow
}

// ScanFeedbackRow is one Scan or cell. Detection and Corners are the client's
// JSON, kept as they arrived; the service checks their shape and size.
type ScanFeedbackRow struct {
	Source       string
	CollectionID uuid.UUID
	Kind         string
	Detection    any
	CardID       *uuid.UUID
	Rank         *int
	RankSource   *string
	ImageKey     *string
	Page         *int
	SlotRow      *int
	SlotColumn   *int
	GridRows     *int
	GridColumns  *int
	Corners      any
}
