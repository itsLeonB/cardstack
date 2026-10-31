package dto

import (
	"time"

	"github.com/google/uuid"
)

// ReviewQueueRequest asks for one page of the review queue.
type ReviewQueueRequest struct {
	Page  int
	Limit int
}

// ReviewLookup names one Correction by its ID.
type ReviewLookup struct {
	ID uuid.UUID
}

// ReviewRejectRequest is an Admin's rejection of one Correction's photo.
type ReviewRejectRequest struct {
	ID                uuid.UUID
	ReviewerProfileID uuid.UUID
}

// ReviewItem is one Correction whose photo waits for the Admin's decision
// (docs/adr/0027). Its photo is read through ReviewPhoto, not returned here.
type ReviewItem struct {
	ID        uuid.UUID         `json:"id"`
	Kind      string            `json:"kind" enum:"switched,reprint,placed" doc:"What the user did: switched to another candidate, picked a print from Other prints, or placed a Card the matcher offered nothing for."`
	Source    string            `json:"source" enum:"card_scan,binder_page" doc:"Where the photo came from: a single card scan, or a binder Page cell."`
	CreatedAt time.Time         `json:"createdAt" doc:"When the Correction was recorded."`
	Card      CardSummary       `json:"card" doc:"The Card the user chose, with its catalog image."`
	Scores    []ReviewScore     `json:"scores" doc:"The scores the matcher gave, in the order it offered them."`
	Binder    *ReviewBinderCell `json:"binder,omitempty" doc:"Set for a binder_page item: the cell's place on its Page, and the Page's grid in the photo."`
}

// ReviewScore is one candidate the matcher offered, with its score.
type ReviewScore struct {
	CardID uuid.UUID `json:"cardId"`
	Score  float64   `json:"score" minimum:"0" maximum:"1"`
}

// ReviewBinderCell is a binder cell's place on its Page and the grid it was cut
// with. Corners are the grid's outer corners on the Page photo.
type ReviewBinderCell struct {
	Page        int           `json:"page" doc:"Page number in turning order, from 1."`
	SlotRow     int           `json:"slotRow" doc:"The cell's row on its Page, from 1."`
	SlotColumn  int           `json:"slotColumn" doc:"The cell's column on its Page, from 1."`
	GridRows    int           `json:"gridRows" doc:"Rows in the Page's grid, from 1."`
	GridColumns int           `json:"gridColumns" doc:"Columns in the Page's grid, from 1."`
	Corners     []ReviewPoint `json:"corners" doc:"Top-left, top-right, bottom-right, bottom-left: the grid's outer corners, each axis 0 to 1 across the photo."`
}

// ReviewPoint is a point on a photo, each axis 0 to 1 across it.
type ReviewPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ReviewPhoto is a short-lived address that reads one Correction's photo.
type ReviewPhoto struct {
	URL       string    `json:"url" doc:"Reads the photo until expiresAt. The Admin app cuts a binder cell from it."`
	ExpiresAt time.Time `json:"expiresAt"`
}
