package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// reviewNotFoundMsg is the message of the 404 AppError, also returned for a
// Correction that is not reviewable, so the review routes do not reveal rows
// the queue never lists.
const reviewNotFoundMsg = "correction not found"

// reviewableWhere makes a row a Correction whose photo the Admin can review: an
// eligible kind, a photo and the chosen Card (docs/adr/0027). A confirmation
// stores no photo, and Empty and Unidentified name no Card.
const reviewableWhere = "kind IN ('switched', 'reprint', 'placed') AND image_key IS NOT NULL AND card_id IS NOT NULL"

// ScanReviewRepository is crud.Repository[entity.ScanFeedback] plus the review
// queue's reads. The review columns are written through the embedded base.
type ScanReviewRepository interface {
	crud.Repository[entity.ScanFeedback]
	// ListPending returns one page of the review queue, oldest first, with each
	// Card loaded with its Rarity and Expansion Set, and the queue's total.
	ListPending(ctx context.Context, page, limit int) ([]entity.ScanFeedback, int64, error)
	// GetReviewable returns the Correction with id whether or not it is reviewed,
	// or the 404 AppError when it is missing or not reviewable. forUpdate
	// row-locks it; use it inside crud.Transactor.WithinTransaction.
	GetReviewable(ctx context.Context, id uuid.UUID, forUpdate bool) (entity.ScanFeedback, error)
}

type scanReviewRepository struct {
	crud.Repository[entity.ScanFeedback]
}

// NewScanReviewRepository builds a ScanReviewRepository over base's database.
func NewScanReviewRepository(base crud.Repository[entity.ScanFeedback]) ScanReviewRepository {
	return &scanReviewRepository{Repository: base}
}

// pendingReviews is the queue's scope: reviewable and not yet decided.
func pendingReviews(db *gorm.DB) *gorm.DB {
	return db.Model(&entity.ScanFeedback{}).Where(reviewableWhere).Where("review_status IS NULL")
}

func (r *scanReviewRepository) ListPending(ctx context.Context, page, limit int) ([]entity.ScanFeedback, int64, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, 0, err
	}

	var total int64
	if err := pendingReviews(db).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []entity.ScanFeedback
	err = pendingReviews(db).
		Preload("Card").
		Preload("Card.Rarity").
		Preload("Card.ExpansionSet").
		Order("created_at ASC, id ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&rows).
		Error
	return rows, total, err
}

func (r *scanReviewRepository) GetReviewable(ctx context.Context, id uuid.UUID, forUpdate bool) (entity.ScanFeedback, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return entity.ScanFeedback{}, err
	}

	q := db.Model(&entity.ScanFeedback{}).Where("id = ?", id).Where(reviewableWhere)
	if forUpdate {
		q = q.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate})
	}

	var row entity.ScanFeedback
	err = q.Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.ScanFeedback{}, ungerr.NotFoundError(reviewNotFoundMsg)
	}
	return row, err
}
