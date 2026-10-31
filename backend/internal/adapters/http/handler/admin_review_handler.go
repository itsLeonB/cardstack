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
	OpListReviewQueue = "list-review-queue"
	OpGetReviewPhoto  = "get-review-photo"
	OpRejectReview    = "reject-review"
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

func (h *AdminReviewHandler) listQueue(ctx context.Context, in listReviewQueueInput) ([]dto.ReviewItem, dto.PaginationMeta, error) {
	return h.reviews.ListQueue(ctx, dto.ReviewQueueRequest{Page: in.Page, Limit: in.Limit})
}

func (h *AdminReviewHandler) getPhoto(ctx context.Context, in reviewIDInput) (dto.ReviewPhoto, error) {
	return h.reviews.PhotoURL(ctx, dto.ReviewLookup{ID: in.ID})
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
