package mapper

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/ezutil/v2"
	"github.com/itsLeonB/ungerr"
	"gorm.io/datatypes"
)

// storedDetection reads only the part of a stored Detection the review shows: the
// candidates and their scores. The rest is kept as the matcher returned it.
type storedDetection struct {
	Candidates []storedCandidate `json:"candidates"`
}

type storedCandidate struct {
	Score float64 `json:"score"`
	Card  struct {
		ID uuid.UUID `json:"id"`
	} `json:"card"`
}

// ToReviewItem converts a queued Correction, with its Card preloaded with its
// Rarity and Expansion Set, into the item the Admin reviews.
func ToReviewItem(host ImageHost, row entity.ScanFeedback) (dto.ReviewItem, error) {
	if row.Card == nil {
		return dto.ReviewItem{}, ungerr.Wrapf(errors.New("chosen Card not preloaded"), "mapping review item %s", row.ID)
	}

	item := dto.ReviewItem{
		ID:        row.ID,
		Kind:      row.Kind,
		Source:    row.Source,
		CreatedAt: row.CreatedAt,
		Card:      ToBinderCardSummary(host, *row.Card),
		Scores:    toReviewScores(row.ID, row.Detection),
	}

	if row.Source != "binder_page" {
		return item, nil
	}
	item.Binder = &dto.ReviewBinderCell{
		Page:        derefInt(row.Page),
		SlotRow:     derefInt(row.SlotRow),
		SlotColumn:  derefInt(row.SlotColumn),
		GridRows:    derefInt(row.GridRows),
		GridColumns: derefInt(row.GridColumns),
		Corners:     toReviewCorners(row.ID, row.Corners),
	}
	return item, nil
}

// toReviewScores reads a stored Detection's candidates. A value it cannot read is
// logged and shown empty, so one bad row does not fail the whole queue: the
// Admin can still reject it.
func toReviewScores(id uuid.UUID, detection datatypes.JSON) []dto.ReviewScore {
	var d storedDetection
	if len(detection) == 0 {
		return []dto.ReviewScore{}
	}
	if err := json.Unmarshal(detection, &d); err != nil {
		logger.Errorf("review item %s: reading its Detection: %v", id, err)
		return []dto.ReviewScore{}
	}
	return ezutil.MapSlice(d.Candidates, func(c storedCandidate) dto.ReviewScore {
		return dto.ReviewScore{CardID: c.Card.ID, Score: c.Score}
	})
}

// toReviewCorners reads a binder cell's Page corners, logged and left empty when
// unreadable for the same reason as toReviewScores.
func toReviewCorners(id uuid.UUID, corners datatypes.JSON) []dto.ReviewPoint {
	var points []dto.ReviewPoint
	if err := json.Unmarshal(corners, &points); err != nil {
		logger.Errorf("review item %s: reading its Page corners: %v", id, err)
		return []dto.ReviewPoint{}
	}
	return points
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
