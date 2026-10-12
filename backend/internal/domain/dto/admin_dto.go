package dto

import (
	"time"

	"github.com/google/uuid"
)

// ExpansionSetUpsert is an Admin's write of one Expansion Set, keyed by Code
// within the Game of its Series (docs/adr/0019).
type ExpansionSetUpsert struct {
	Code        string
	Name        string
	ReleaseDate time.Time
	SeriesID    uuid.UUID
}
