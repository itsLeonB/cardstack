package service

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/apperr"
	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ezutil/v2"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"github.com/pgvector/pgvector-go"
)

// reviewPhotoTTL is how long a photo address works: long enough to load and cut
// the photo, short enough that a leaked address is nearly useless.
const reviewPhotoTTL = 5 * time.Minute

const (
	reviewedMsg          = "this Correction is already reviewed"
	reviewNotPendingMsg  = "this Correction is not waiting for review"
	reviewCardGoneMsg    = "the Card this Correction chose no longer exists, so it can only be rejected"
	reviewKeyMsg         = "the key is not the one this Correction's upload address returned"
	reviewMissingMsg     = "no photo was uploaded under the key"
	reviewTooLargeMsg    = "the uploaded photo is over the size cap"
	reviewNotJPEGMsg     = "the uploaded photo is not a JPEG"
	reviewRejectedStatus = "rejected"
	reviewApprovedStatus = "approved"
)

// reviewedKey is the one object key a Correction's cut is uploaded to and
// approved from. The server picks it, so an Admin cannot write elsewhere in the
// bucket (docs/adr/0027).
func reviewedKey(id uuid.UUID) string {
	return "reviewed/" + id.String() + ".jpg"
}

// ScanReviewService is the Admin's review of the photos kept from Corrections
// (docs/adr/0027).
type ScanReviewService interface {
	// ListQueue returns one page of the Corrections waiting for review, oldest
	// first, and the queue's total.
	ListQueue(ctx context.Context, req dto.ReviewQueueRequest) ([]dto.ReviewItem, dto.PaginationMeta, error)
	// PhotoURL signs a short-lived address for a queued Correction's photo. A
	// Correction that is not waiting for review is a 404, so the private bucket
	// is read only for rows the queue lists.
	PhotoURL(ctx context.Context, req dto.ReviewLookup) (dto.ReviewPhoto, error)
	// UploadURL signs a short-lived PUT address for the Admin's cut of a queued
	// Correction's photo, under the one key Approve accepts for it. A Correction
	// already decided is a 409, so an approved cut is never overwritten.
	UploadURL(ctx context.Context, req dto.ReviewLookup) (dto.ReviewUpload, error)
	// Approve embeds the cut uploaded to req.Key as an extra image of the Card
	// the Correction chose, and records the approval in the same transaction. A
	// failed embedding stores nothing and leaves the Correction pending.
	Approve(ctx context.Context, req dto.ReviewApproveRequest) error
	// Reject marks a pending Correction rejected. It embeds nothing and deletes
	// no photo (docs/adr/0022). A Correction already decided is a 409.
	Reject(ctx context.Context, req dto.ReviewRejectRequest) error
}

type scanReviewService struct {
	transactor crud.Transactor
	reviews    repository.ScanReviewRepository
	embeddings repository.EmbeddingRepository
	photos     objectstore.ObjectStore
	images     mapper.ImageHost
	embedder   embedding.ImageEmbedder
	model      string
}

// NewScanReviewService builds the review service over the private photo bucket
// and the image host the chosen Cards' catalog images come from. It embeds an
// approved photo with embedder and model, the matcher's own, so the photo is
// compared like any other query. A nil photos refuses every photo address and
// every approval, since no photo can be read or signed.
func NewScanReviewService(
	transactor crud.Transactor,
	reviews repository.ScanReviewRepository,
	embeddings repository.EmbeddingRepository,
	photos objectstore.ObjectStore,
	images mapper.ImageHost,
	embedder embedding.ImageEmbedder,
	model string,
) ScanReviewService {
	return &scanReviewService{
		transactor: transactor,
		reviews:    reviews,
		embeddings: embeddings,
		photos:     photos,
		images:     images,
		embedder:   embedder,
		model:      model,
	}
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

func (s *scanReviewService) UploadURL(ctx context.Context, req dto.ReviewLookup) (dto.ReviewUpload, error) {
	if s.photos == nil {
		return dto.ReviewUpload{}, apperr.WithCode(apperr.ServiceUnavailableError(scanPhotoNotConfiguredMsg), apperr.CodePhotoStorageNotConfigured)
	}

	row, err := s.reviews.GetReviewable(ctx, req.ID, false)
	if err != nil {
		return dto.ReviewUpload{}, err
	}
	if row.ReviewStatus != nil {
		return dto.ReviewUpload{}, ungerr.ConflictError(reviewedMsg)
	}

	key := reviewedKey(req.ID)
	url, expiresAt, err := s.photos.PresignPut(ctx, key, scanPhotoContentType, scanPhotoUploadTTL)
	if err != nil {
		return dto.ReviewUpload{}, err
	}
	return dto.ReviewUpload{Key: key, URL: url, ExpiresAt: expiresAt}, nil
}

func (s *scanReviewService) Approve(ctx context.Context, req dto.ReviewApproveRequest) error {
	if s.photos == nil {
		return apperr.WithCode(apperr.ServiceUnavailableError(scanPhotoNotConfiguredMsg), apperr.CodePhotoStorageNotConfigured)
	}

	// An unlocked look first, so a decided or unknown Correction is answered
	// before any bucket or embedder call. The transaction checks again under a lock.
	row, err := s.reviews.GetReviewable(ctx, req.ID, false)
	if err != nil {
		return err
	}
	if err := s.checkApprovable(ctx, row); err != nil {
		return err
	}
	if req.Key != reviewedKey(req.ID) {
		return ungerr.UnprocessableEntityError(reviewKeyMsg)
	}

	// The embedder is a network call, so it runs outside the transaction and
	// holds no row lock. A failure here stores nothing.
	vector, err := s.embedUpload(ctx, req.Key)
	if err != nil {
		return err
	}

	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		locked, err := s.reviews.GetReviewable(ctx, req.ID, true)
		if err != nil {
			return err
		}
		if err := s.checkApprovable(ctx, locked); err != nil {
			return err
		}

		stored := entity.CardEmbedding{
			CardID:    *locked.CardID,
			Model:     s.model,
			Source:    "correction:" + locked.ID.String(),
			Embedding: pgvector.NewVector(vector),
		}
		if err := s.embeddings.Upsert(ctx, []entity.CardEmbedding{stored}); err != nil {
			return err
		}
		return s.decide(ctx, locked, reviewApprovedStatus, req.ReviewerProfileID)
	})
}

// checkApprovable refuses a Correction that is decided, or whose chosen Card has
// gone. Both are checked again under the row lock in Approve.
func (s *scanReviewService) checkApprovable(ctx context.Context, row entity.ScanFeedback) error {
	if row.ReviewStatus != nil {
		return ungerr.ConflictError(reviewedMsg)
	}
	live, err := s.reviews.CardExists(ctx, *row.CardID)
	if err != nil {
		return err
	}
	if !live {
		return ungerr.ConflictError(reviewCardGoneMsg)
	}
	return nil
}

// embedUpload checks the object uploaded under key and embeds it. The object
// must be within the size cap and start with a JPEG signature. A refused object
// stays in the bucket: photos are kept until the operator deletes them
// (docs/adr/0022).
func (s *scanReviewService) embedUpload(ctx context.Context, key string) ([]float32, error) {
	size, err := s.photos.Size(ctx, key)
	if errors.Is(err, objectstore.ErrNotFound) {
		return nil, ungerr.UnprocessableEntityError(reviewMissingMsg)
	}
	if err != nil {
		return nil, err
	}
	if size > MaxFeedbackPhotoBytes {
		return nil, ungerr.UnprocessableEntityError(reviewTooLargeMsg)
	}

	data, err := s.photos.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(data, jpegSignature) {
		return nil, ungerr.UnprocessableEntityError(reviewNotJPEGMsg)
	}
	return s.embedder.Embed(ctx, jpegMIME, data)
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
		return s.decide(ctx, row, reviewRejectedStatus, req.ReviewerProfileID)
	})
}

// decide records the Admin's decision on a pending Correction. The caller holds
// the row's lock and has checked that the Correction is pending.
func (s *scanReviewService) decide(ctx context.Context, row entity.ScanFeedback, status string, reviewer uuid.UUID) error {
	reviewedAt := time.Now()
	row.ReviewStatus = &status
	row.ReviewedBy = &reviewer
	row.ReviewedAt = &reviewedAt

	_, err := s.reviews.Update(ctx, row)
	return err
}
