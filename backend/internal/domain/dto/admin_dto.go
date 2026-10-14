package dto

import (
	"time"

	"github.com/google/uuid"
)

// ExpansionSetCreate is an Admin's entry of a new Expansion Set. Its Code is
// unique within the Game of its Series and, once created, never changes
// (docs/adr/0019).
type ExpansionSetCreate struct {
	Code        string
	Name        string
	ReleaseDate time.Time
	SeriesID    uuid.UUID
}

// ExpansionSetUpdate replaces what an Admin can edit of the Expansion Set with
// ID: its name, release date and Series, which must be of the set's Game. The
// code stays as it is.
type ExpansionSetUpdate struct {
	ID          uuid.UUID
	Name        string
	ReleaseDate time.Time
	SeriesID    uuid.UUID
}
