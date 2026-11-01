package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
)

// Operation IDs of the review routes (docs/adr/0027).
const (
	OpListReviewQueue     = "list-review-queue"
	OpGetReviewPhoto      = "get-review-photo"
	OpRequestReviewUpload = "request-review-upload"
	OpApproveReview       = "approve-review"
	OpRejectReview        = "reject-review"
)

// AdminReviewHandler holds the Admin's photo review routes. Like the other admin
// routes, RegisterRoutes mounts them behind authpkg.AdminOnly (docs/adr/0024).
type AdminReviewHandler struct {
	reviews service.ScanReviewService
}

func NewAdminReviewHandler(reviews service.ScanReviewService) *AdminReviewHandler {
	return &AdminReviewHandler{reviews: reviews}
}

type listReviewQueueInput struct {
	Page  int `query:"page" default:"1" minimum:"1" doc:"1-indexed page number."`
	Limit int `query:"limit" default:"24" minimum:"1" maximum:"100" doc:"Page size."`
}

type reviewIDInput struct {
	ID uuid.UUID `path:"id" doc:"Correction ID"`
}

type approveReviewInput struct {
	ID   uuid.UUID `path:"id" doc:"Correction ID"`
	Body struct {
		Key string `json:"key" required:"true" minLength:"1" maxLength:"200" doc:"The key the cut was uploaded to: the key the upload address returned."`
	}
}

func (h *AdminReviewHandler) listQueue(ctx context.Context, in listReviewQueueInput) ([]dto.ReviewItem, dto.PaginationMeta, error) {
	return h.reviews.ListQueue(ctx, dto.ReviewQueueRequest{Page: in.Page, Limit: in.Limit})
}

func (h *AdminReviewHandler) getPhoto(ctx context.Context, in reviewIDInput) (dto.ReviewPhoto, error) {
	return h.reviews.PhotoURL(ctx, dto.ReviewLookup{ID: in.ID})
}

func (h *AdminReviewHandler) requestUpload(ctx context.Context, in reviewIDInput) (dto.ReviewUpload, error) {
	return h.reviews.UploadURL(ctx, dto.ReviewLookup{ID: in.ID})
}

func (h *AdminReviewHandler) approve(ctx context.Context, in approveReviewInput) error {
	return h.reviews.Approve(ctx, dto.ReviewApproveRequest{ID: in.ID, ReviewerProfileID: authpkg.CallerFrom(ctx).ProfileID, Key: in.Body.Key})
}

func (h *AdminReviewHandler) reject(ctx context.Context, in reviewIDInput) error {
	return h.reviews.Reject(ctx, dto.ReviewRejectRequest{ID: in.ID, ReviewerProfileID: authpkg.CallerFrom(ctx).ProfileID})
}

// Routes returns every route AdminReviewHandler exposes, for registration via
// endpoint.RegisterAll.
func (h *AdminReviewHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.NewWithMeta(endpoint.EndpointWithMeta[listReviewQueueInput, []dto.ReviewItem, dto.PaginationMeta]{
			OperationID: OpListReviewQueue,
			Method:      http.MethodGet,
			Path:        "/admin/reviews",
			Summary:     "List the Corrections whose photos wait for review, oldest first: a switched, reprint or placed Correction with a photo and a chosen Card, not yet approved or rejected",
			Tags:        []string{"admin"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.listQueue,
		}),
		endpoint.New(endpoint.Endpoint[reviewIDInput, dto.ReviewPhoto]{
			OperationID: OpGetReviewPhoto,
			Method:      http.MethodGet,
			Path:        "/admin/reviews/{id}/photo",
			Summary:     "Sign a short-lived address that reads a queued Correction's photo (404 when the Correction is not waiting for review)",
			Tags:        []string{"admin"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.getPhoto,
		}),
		endpoint.New(endpoint.Endpoint[reviewIDInput, dto.ReviewUpload]{
			OperationID: OpRequestReviewUpload,
			Method:      http.MethodPost,
			Path:        "/admin/reviews/{id}/upload",
			Summary:     "Sign a short-lived PUT address for the Admin's cut of a queued Correction's photo, under the one key approve accepts (409 when already reviewed)",
			Tags:        []string{"admin"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.requestUpload,
		}),
		endpoint.NewNoBody(endpoint.NoBodyEndpoint[approveReviewInput]{
			OperationID: OpApproveReview,
			Method:      http.MethodPost,
			Path:        "/admin/reviews/{id}/approve",
			Summary:     "Approve a queued Correction's photo from its uploaded cut: embed it as an extra image of the Card the user chose and mark it approved, in one step. A failed embedding stores nothing and leaves it queued. 404 when the Correction is missing or not waiting for review; 409 when already reviewed; 422 when the upload is missing, not a JPEG, over the size cap or not this Correction's key",
			Tags:        []string{"admin"},
			Secured:     true,
			HandlerFunc: h.approve,
		}),
		endpoint.NewNoBody(endpoint.NoBodyEndpoint[reviewIDInput]{
			OperationID: OpRejectReview,
			Method:      http.MethodPost,
			Path:        "/admin/reviews/{id}/reject",
			Summary:     "Reject a queued Correction's photo: it leaves the queue for good and embeds nothing. 409 when it was already reviewed",
			Tags:        []string{"admin"},
			Secured:     true,
			HandlerFunc: h.reject,
		}),
	}
}
