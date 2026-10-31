package service

import (
	"context"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/core/apperr"
	"github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ezutil/v2"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// reviewPhotoTTL is how long a photo address works: long enough to load and cut
// the photo, short enough that a leaked address is nearly useless.
const reviewPhotoTTL = 5 * time.Minute

const (
	reviewedMsg          = "this Correction is already reviewed"
	reviewNotPendingMsg  = "this Correction is not waiting for review"
	reviewRejectedStatus = "rejected"
)

// ScanReviewService is the Admin's review of the photos kept from Corrections
// (docs/adr/0027). Approval is added with the embedding it stores.
type ScanReviewService interface {
	// ListQueue returns one page of the Corrections waiting for review, oldest
	// first, and the queue's total.
	ListQueue(ctx context.Context, req dto.ReviewQueueRequest) ([]dto.ReviewItem, dto.PaginationMeta, error)
	// PhotoURL signs a short-lived address for a queued Correction's photo. A
	// Correction that is not waiting for review is a 404, so the private bucket
	// is read only for rows the queue lists.
	PhotoURL(ctx context.Context, req dto.ReviewLookup) (dto.ReviewPhoto, error)
	// Reject marks a pending Correction rejected. It embeds nothing and deletes
	// no photo (docs/adr/0022). A Correction already decided is a 409.
	Reject(ctx context.Context, req dto.ReviewRejectRequest) error
}

type scanReviewService struct {
	transactor crud.Transactor
	reviews    repository.ScanReviewRepository
	photos     objectstore.ObjectStore
	images     mapper.ImageHost
}

// NewScanReviewService builds the review service over the private photo bucket
// and the image host the chosen Cards' catalog images come from. A nil photos
// refuses every photo address, since none can be signed.
func NewScanReviewService(transactor crud.Transactor, reviews repository.ScanReviewRepository, photos objectstore.ObjectStore, images mapper.ImageHost) ScanReviewService {
	return &scanReviewService{transactor: transactor, reviews: reviews, photos: photos, images: images}
}

func (s *scanReviewService) ListQueue(ctx context.Context, req dto.ReviewQueueRequest) ([]dto.ReviewItem, dto.PaginationMeta, error) {
	page, limit := normalizePagination(req.Page, req.Limit)

	rows, total, err := s.reviews.ListPending(ctx, page, limit)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	items, err := ezutil.MapSliceWithError(rows, func(row entity.ScanFeedback) (dto.ReviewItem, error) {
		return mapper.ToReviewItem(s.images, row)
	})
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	return items, dto.PaginationMeta{Total: int(total), Page: page, Limit: limit}, nil
}

func (s *scanReviewService) PhotoURL(ctx context.Context, req dto.ReviewLookup) (dto.ReviewPhoto, error) {
	if s.photos == nil {
		return dto.ReviewPhoto{}, apperr.WithCode(apperr.ServiceUnavailableError(scanPhotoNotConfiguredMsg), apperr.CodePhotoStorageNotConfigured)
	}

	row, err := s.reviews.GetReviewable(ctx, req.ID, false)
	if err != nil {
		return dto.ReviewPhoto{}, err
	}
	if row.ReviewStatus != nil {
		return dto.ReviewPhoto{}, ungerr.NotFoundError(reviewNotPendingMsg)
	}

	url, expiresAt, err := s.photos.PresignGet(ctx, *row.ImageKey, reviewPhotoTTL)
	if err != nil {
		return dto.ReviewPhoto{}, err
	}
	return dto.ReviewPhoto{URL: url, ExpiresAt: expiresAt}, nil
}

func (s *scanReviewService) Reject(ctx context.Context, req dto.ReviewRejectRequest) error {
	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		row, err := s.reviews.GetReviewable(ctx, req.ID, true)
		if err != nil {
			return err
		}
		if row.ReviewStatus != nil {
			return ungerr.ConflictError(reviewedMsg)
		}

		status := reviewRejectedStatus
		reviewedAt := time.Now()
		row.ReviewStatus = &status
		row.ReviewedBy = &req.ReviewerProfileID
		row.ReviewedAt = &reviewedAt

		_, err = s.reviews.Update(ctx, row)
		return err
	})
}
