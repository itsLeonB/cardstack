// Package collection holds the Collections CRUD domain (ticket 06): an
// authenticated user creates, lists, views, edits, and deletes their own
// Collections (binders/boxes/decks). CollectionRepository and
// CollectionService each keep their interface and implementation together
// in one file here, under internal/domain/collection rather than split
// across internal/domain (interface) and internal/adapters (implementation)
// — per docs/adr/0011, there's exactly one Postgres-backed implementation
// of either, ever, so that split would manufacture a seam nothing actually
// varies across.
package collection

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

// CollectionRepository is the persistence access the collection domain
// needs: creating a Collection, listing a user's own Collections, and
// looking one up by ID so the service can apply its own ownership check
// before editing/deleting it.
type CollectionRepository interface {
	Create(ctx context.Context, c entity.Collection) (entity.Collection, error)
	// ListByUser returns every Collection owned by userID, most recently
	// created first (crud.Repository's DefaultOrder).
	ListByUser(ctx context.Context, userID uuid.UUID) ([]entity.Collection, error)
	// FindByID returns the Collection with id, or ErrCollectionNotFound if
	// none exists. It does not filter by owner - CollectionService applies
	// that check itself once it has the row.
	FindByID(ctx context.Context, id uuid.UUID) (entity.Collection, error)
	Update(ctx context.Context, c entity.Collection) (entity.Collection, error)
	// Delete hard-deletes c - no soft delete (see ticket 06).
	Delete(ctx context.Context, c entity.Collection) error
}

// collectionRepository is CollectionRepository's implementation, backed by
// crud.Repository[entity.Collection] for the actual GORM/transaction
// mechanics (mirrors UserRepository's use of crud.Repository) rather than
// hand-rolled GORM calls, since every method here is a straightforward
// single-row read/write crud.Repository already covers.
type collectionRepository struct {
	repo crud.Repository[entity.Collection]
}

// NewCollectionRepository builds a CollectionRepository over db.
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
	// crud.WhereBySpec) and match an arbitrary row instead of correctly
	// reporting "not found" - mirrors UserRepository.findByID's same guard.
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
