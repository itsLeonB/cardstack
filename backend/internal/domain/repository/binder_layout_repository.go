package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

const layoutNotFoundMsg = "this Collection is not a binder"

// BinderLayoutRepository reads and writes a binder's Layout, the row that makes a
// Collection a binder.
type BinderLayoutRepository interface {
	crud.Repository[entity.CollectionLayout]
	// FindLayout returns the Collection's Layout, or the zero value when it is
	// not a binder.
	FindLayout(ctx context.Context, collectionID uuid.UUID) (entity.CollectionLayout, error)
	// GetLayout is FindLayout, with the 404 AppError for a plain Collection.
	GetLayout(ctx context.Context, collectionID uuid.UUID) (entity.CollectionLayout, error)
	IsBinder(ctx context.Context, collectionID uuid.UUID) (bool, error)
}

type binderLayoutRepository struct {
	crud.Repository[entity.CollectionLayout]
}

func NewBinderLayoutRepository(base crud.Repository[entity.CollectionLayout]) BinderLayoutRepository {
	return &binderLayoutRepository{Repository: base}
}

func (r *binderLayoutRepository) FindLayout(ctx context.Context, collectionID uuid.UUID) (entity.CollectionLayout, error) {
	// A nil collection ID would drop out of the Model and match any binder.
	if collectionID == uuid.Nil {
		return entity.CollectionLayout{}, nil
	}

	return r.FindFirst(ctx, crud.Specification[entity.CollectionLayout]{
		Model: entity.CollectionLayout{CollectionID: collectionID},
	})
}

func (r *binderLayoutRepository) GetLayout(ctx context.Context, collectionID uuid.UUID) (entity.CollectionLayout, error) {
	layout, err := r.FindLayout(ctx, collectionID)
	if err != nil {
		return entity.CollectionLayout{}, err
	}
	if layout.IsZero() {
		return entity.CollectionLayout{}, ungerr.NotFoundError(layoutNotFoundMsg)
	}

	return layout, nil
}

func (r *binderLayoutRepository) IsBinder(ctx context.Context, collectionID uuid.UUID) (bool, error) {
	layout, err := r.FindLayout(ctx, collectionID)
	if err != nil {
		return false, err
	}

	return !layout.IsZero(), nil
}
