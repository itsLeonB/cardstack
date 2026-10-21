package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/apperr"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"gorm.io/datatypes"
)

// MaxScanPhotoBytes caps a kept photo at the size /scan/match accepts, so a
// photo kept for a Correction is one the matcher could read (docs/adr/0022).
const MaxScanPhotoBytes = 2 << 20

// MaxScanJSONBytes caps a Detection's JSON and a Page's corners.
const MaxScanJSONBytes = 64 << 10

// jpegSignature is what every JPEG starts with: the SOI marker and the next
// marker's start.
var jpegSignature = []byte{0xFF, 0xD8, 0xFF}

const (
	feedbackConfirmationKeyMsg    = "a confirmation carries no photo: only a Correction may carry an image key"
	feedbackRowMsg                = "a feedback row's kind, Card, rank, source or fields do not agree with each other"
	feedbackJSONTooLargeMsg       = "a Detection or corners JSON is over the size cap"
	feedbackPhotoOutsidePrefixMsg = "an image key is outside the caller's own prefix"
	feedbackPhotoMissingMsg       = "an image key names no uploaded photo"
	feedbackPhotoTooLargeMsg      = "a scan photo is over the size cap"
	feedbackPhotoNotJPEGMsg       = "a scan photo is not a JPEG"
)

// ScanFeedbackService records the rows of a confirmed Draft Addition or Page
// Review (docs/adr/0022).
type ScanFeedbackService interface {
	Record(ctx context.Context, req dto.ScanFeedbackBatch) error
}

type scanFeedbackService struct {
	transactor  crud.Transactor
	collections repository.CollectionRepository
	feedback    crud.Repository[entity.ScanFeedback]
	photos      objectstore.ObjectStore
}

// NewScanFeedbackService builds the service over the private photo bucket. A nil
// photos refuses every batch, since no photo can be checked.
func NewScanFeedbackService(transactor crud.Transactor, collections repository.CollectionRepository, feedback crud.Repository[entity.ScanFeedback], photos objectstore.ObjectStore) ScanFeedbackService {
	return &scanFeedbackService{transactor: transactor, collections: collections, feedback: feedback, photos: photos}
}

func (s *scanFeedbackService) Record(ctx context.Context, req dto.ScanFeedbackBatch) error {
	if s.photos == nil {
		return apperr.WithCode(apperr.ServiceUnavailableError(scanPhotoNotConfiguredMsg), apperr.CodePhotoStorageNotConfigured)
	}

	rows := make([]entity.ScanFeedback, len(req.Rows))
	for i, r := range req.Rows {
		row, err := toScanFeedback(req.ProfileID, r)
		if err != nil {
			return err
		}
		rows[i] = row
	}

	owned := map[uuid.UUID]bool{}
	for _, row := range rows {
		if owned[row.CollectionID] {
			continue
		}
		if _, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, row.CollectionID, false); err != nil {
			return err
		}
		owned[row.CollectionID] = true
	}

	if err := s.checkPhotos(ctx, req.ProfileID, rows); err != nil {
		return err
	}

	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		for _, row := range rows {
			if _, err := s.feedback.Insert(ctx, row); err != nil {
				return err
			}
		}
		return nil
	})
}

// toScanFeedback checks one row against the rules of docs/adr/0022 and turns it
// into the entity. The kind decides whether a Card is needed, the Card whether a
// rank is, and the source which binder fields are set.
func toScanFeedback(profileID uuid.UUID, r dto.ScanFeedbackRow) (entity.ScanFeedback, error) {
	if r.Kind == "confirmed" && r.ImageKey != nil {
		return entity.ScanFeedback{}, ungerr.BadRequestError(feedbackConfirmationKeyMsg)
	}

	hasCard := r.CardID != nil
	switch r.Kind {
	case "empty", "unidentified":
		if hasCard {
			return entity.ScanFeedback{}, ungerr.BadRequestError(feedbackRowMsg)
		}
	case "switched", "reprint", "placed":
		if !hasCard {
			return entity.ScanFeedback{}, ungerr.BadRequestError(feedbackRowMsg)
		}
	}

	hasRank := r.Rank != nil && r.RankSource != nil
	noRank := r.Rank == nil && r.RankSource == nil
	if (hasCard && !hasRank) || (!hasCard && !noRank) || (r.Rank != nil && *r.Rank < 1) {
		return entity.ScanFeedback{}, ungerr.BadRequestError(feedbackRowMsg)
	}

	if r.Source == "binder_page" {
		if !positive(r.Page) || !positive(r.SlotRow) || !positive(r.SlotColumn) || !positive(r.GridRows) || !positive(r.GridColumns) || r.Corners == nil {
			return entity.ScanFeedback{}, ungerr.BadRequestError(feedbackRowMsg)
		}
	} else if r.Page != nil || r.SlotRow != nil || r.SlotColumn != nil || r.GridRows != nil || r.GridColumns != nil || r.Corners != nil {
		return entity.ScanFeedback{}, ungerr.BadRequestError(feedbackRowMsg)
	}

	detection, err := scanJSON(r.Detection)
	if err != nil {
		return entity.ScanFeedback{}, err
	}
	corners, err := scanJSON(r.Corners)
	if err != nil {
		return entity.ScanFeedback{}, err
	}

	return entity.ScanFeedback{
		ProfileID:      profileID,
		CollectionID:   r.CollectionID,
		Source:         r.Source,
		Kind:           r.Kind,
		Detection:      detection,
		CardID:         r.CardID,
		CardRank:       r.Rank,
		CardRankSource: r.RankSource,
		ImageKey:       r.ImageKey,
		Page:           r.Page,
		SlotRow:        r.SlotRow,
		SlotColumn:     r.SlotColumn,
		GridRows:       r.GridRows,
		GridColumns:    r.GridColumns,
		Corners:        corners,
	}, nil
}

// scanJSON encodes a Detection or corners as they arrived. A nil one stores NULL.
func scanJSON(v any) (datatypes.JSON, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, ungerr.Wrapf(err, "encoding scan feedback JSON")
	}
	if len(b) > MaxScanJSONBytes {
		return nil, ungerr.BadRequestError(feedbackJSONTooLargeMsg)
	}
	return b, nil
}

func positive(p *int) bool {
	return p != nil && *p >= 1
}

// checkPhotos checks each distinct image key once, before any row is stored. A
// key outside the caller's prefix is refused without touching its object, which
// another profile owns. A photo that fails a later check sits under the caller's
// prefix, so it is deleted.
func (s *scanFeedbackService) checkPhotos(ctx context.Context, profileID uuid.UUID, rows []entity.ScanFeedback) error {
	checked := map[string]bool{}
	for _, row := range rows {
		if row.ImageKey == nil || checked[*row.ImageKey] {
			continue
		}
		key := *row.ImageKey
		checked[key] = true

		if !strings.HasPrefix(key, profileID.String()+"/") {
			return ungerr.BadRequestError(feedbackPhotoOutsidePrefixMsg)
		}
		size, err := s.photos.Size(ctx, key)
		if errors.Is(err, objectstore.ErrNotFound) {
			return ungerr.BadRequestError(feedbackPhotoMissingMsg)
		}
		if err != nil {
			return err
		}
		if size > MaxScanPhotoBytes {
			return s.refusePhoto(ctx, key, feedbackPhotoTooLargeMsg)
		}
		head, err := s.photos.ReadHead(ctx, key, len(jpegSignature))
		if errors.Is(err, objectstore.ErrNotFound) {
			return ungerr.BadRequestError(feedbackPhotoMissingMsg)
		}
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(head, jpegSignature) {
			return s.refusePhoto(ctx, key, feedbackPhotoNotJPEGMsg)
		}
	}
	return nil
}

// refusePhoto deletes a photo that failed its check and refuses the batch. A
// failed delete is logged rather than returned: the refusal stands, and the
// object is an orphan the operator accepts (docs/adr/0022).
func (s *scanFeedbackService) refusePhoto(ctx context.Context, key, msg string) error {
	if err := s.photos.Delete(ctx, key); err != nil {
		logger.Errorf("deleting refused scan photo %q: %v", key, err)
	}
	return ungerr.BadRequestError(msg)
}
