package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	"github.com/itsLeonB/ezutil/v2"
)

type ScanFeedbackHandler struct {
	feedback service.ScanFeedbackService
}

func NewScanFeedbackHandler(feedback service.ScanFeedbackService) *ScanFeedbackHandler {
	return &ScanFeedbackHandler{feedback: feedback}
}

type scanFeedbackInput struct {
	Body struct {
		Rows []scanFeedbackRowInput `json:"rows" required:"true" minItems:"1" maxItems:"100" doc:"One row per confirmed Scan or Page Review cell. Up to 100 in a batch."`
	}
}

type scanFeedbackRowInput struct {
	Source       string     `json:"source" required:"true" enum:"card_scan,binder_page" doc:"Where the photo came from: a single card scan, or a binder Page Review cell."`
	CollectionID uuid.UUID  `json:"collectionId" required:"true" doc:"The Collection the Scan or cell was aimed at."`
	Kind         string     `json:"kind" required:"true" enum:"confirmed,switched,reprint,empty,unidentified,placed" doc:"What the user did with the Detection."`
	Detection    any        `json:"detection,omitempty" doc:"The Detection as /scan/match returned it, kept as is. Up to 64 KiB of JSON."`
	CardID       *uuid.UUID `json:"cardId,omitempty" doc:"The Card chosen. Absent for empty and unidentified, and for a confirmation of either."`
	Rank         *int       `json:"rank,omitempty" doc:"The chosen Card's position in the list the matcher offered, or its position in the Reprints list counted across pages. Required with cardId."`
	RankSource   *string    `json:"rankSource,omitempty" enum:"top,reprint" doc:"Whether the rank counts the top list or the Reprints list. Required with cardId."`
	ImageKey     *string    `json:"imageKey,omitempty" doc:"The photo's key from /scan/photo-uploads. Only a Correction may carry one."`
	Page         *int       `json:"page,omitempty" doc:"A binder cell's Page number, from 1. Required for a binder_page row and refused otherwise."`
	SlotRow      *int       `json:"slotRow,omitempty" doc:"A binder cell's row on its Page, from 1."`
	SlotColumn   *int       `json:"slotColumn,omitempty" doc:"A binder cell's column on its Page, from 1."`
	GridRows     *int       `json:"gridRows,omitempty" doc:"Rows in the Page's grid, from 1, used to cut the photo."`
	GridColumns  *int       `json:"gridColumns,omitempty" doc:"Columns in the Page's grid, from 1, used to cut the photo."`
	Corners      any        `json:"corners,omitempty" doc:"The four corners of the Page's grid in the photo, as the client cut it. Up to 64 KiB of JSON."`
}

func (h *ScanFeedbackHandler) record(ctx context.Context, in scanFeedbackInput) error {
	rows := ezutil.MapSlice(in.Body.Rows, func(r scanFeedbackRowInput) dto.ScanFeedbackRow {
		return dto.ScanFeedbackRow{
			Source:       r.Source,
			CollectionID: r.CollectionID,
			Kind:         r.Kind,
			Detection:    r.Detection,
			CardID:       r.CardID,
			Rank:         r.Rank,
			RankSource:   r.RankSource,
			ImageKey:     r.ImageKey,
			Page:         r.Page,
			SlotRow:      r.SlotRow,
			SlotColumn:   r.SlotColumn,
			GridRows:     r.GridRows,
			GridColumns:  r.GridColumns,
			Corners:      r.Corners,
		}
	})
	return h.feedback.Record(ctx, dto.ScanFeedbackBatch{ProfileID: authpkg.CallerFrom(ctx).ProfileID, Rows: rows})
}

func (h *ScanFeedbackHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.NewNoBody(endpoint.NoBodyEndpoint[scanFeedbackInput]{
			OperationID: "record-scan-feedback",
			Method:      http.MethodPost,
			Path:        "/scan/feedback",
			Summary:     "Record one batch of confirmed scan choices, up to 100 rows, with the photos of its Corrections, which are checked first (a JPEG under the caller's prefix, under the size cap) and deleted when they fail. 503 with code photo_storage_not_configured when the private photo bucket is not configured",
			Tags:        []string{"scan"},
			Secured:     true,
			HandlerFunc: h.record,
		}),
	}
}
