package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"gorm.io/gorm"
)

// ErrCollectionNotFound is a 404 AppError: also returned for another user's
// Collection so its existence isn't leaked.
var ErrCollectionNotFound = ungerr.NotFoundError("collection not found")

type CollectionRepository interface {
	Create(ctx context.Context, c entity.Collection) (entity.Collection, error)
	// ListByUser returns most recently created first (crud.Repository's DefaultOrder).
	ListByUser(ctx context.Context, userID uuid.UUID) ([]entity.Collection, error)
	// FindByID does not filter by owner; CollectionService checks ownership.
	FindByID(ctx context.Context, id uuid.UUID) (entity.Collection, error)
	Update(ctx context.Context, c entity.Collection) (entity.Collection, error)
	// Delete is a hard delete.
	Delete(ctx context.Context, c entity.Collection) error
}

type collectionRepository struct {
	repo crud.Repository[entity.Collection]
}

func NewCollectionRepository(db *gorm.DB) CollectionRepository {
	return &collectionRepository{repo: crud.NewRepository[entity.Collection](db)}
}

func (r *collectionRepository) Create(ctx context.Context, c entity.Collection) (entity.Collection, error) {
	created, err := r.repo.Insert(ctx, c)
	if err != nil {
		return entity.Collection{}, ungerr.Wrap(err, "error inserting collection")
	}

	return created, nil
}

func (r *collectionRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]entity.Collection, error) {
	collections, err := r.repo.FindAll(ctx, crud.Specification[entity.Collection]{
		Model: entity.Collection{UserID: userID},
	})
	if err != nil {
		return nil, ungerr.Wrap(err, "error listing collections")
	}

	return collections, nil
}

func (r *collectionRepository) FindByID(ctx context.Context, id uuid.UUID) (entity.Collection, error) {
	// A zero-value id would otherwise drop the WHERE clause entirely (see
	// crud.WhereBySpec) and match an arbitrary row instead of reporting
	// "not found".
	if id == uuid.Nil {
		return entity.Collection{}, ErrCollectionNotFound
	}

	c, err := r.repo.FindFirst(ctx, crud.Specification[entity.Collection]{
		Model: entity.Collection{BaseEntity: crud.BaseEntity{ID: id}},
	})
	if err != nil {
		return entity.Collection{}, ungerr.Wrap(err, "error finding collection")
	}
	if c.IsZero() {
		return entity.Collection{}, ErrCollectionNotFound
	}

	return c, nil
}

func (r *collectionRepository) Update(ctx context.Context, c entity.Collection) (entity.Collection, error) {
	updated, err := r.repo.Update(ctx, c)
	if err != nil {
		return entity.Collection{}, ungerr.Wrap(err, "error updating collection")
	}

	return updated, nil
}

func (r *collectionRepository) Delete(ctx context.Context, c entity.Collection) error {
	if err := r.repo.Delete(ctx, c); err != nil {
		return ungerr.Wrap(err, "error deleting collection")
	}

	return nil
}
