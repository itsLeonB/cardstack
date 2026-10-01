package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// collectionNotFoundMsg is the message of the 404 AppError, also returned for
// another profile's Collection so its existence isn't leaked. Each return
// site builds its own ungerr.NotFoundError so ungerr records that line.
const collectionNotFoundMsg = "collection not found"

// CollectionRepository is crud.Repository[entity.Collection] plus the
// owner-scoped lookup every Collection operation starts with.
type CollectionRepository interface {
	crud.Repository[entity.Collection]
	// GetOwnedCollection returns the profile's Collection, or the 404
	// AppError when it is missing or another profile's. It never returns a
	// zero value with a nil error. forUpdate row-locks the match; use it
	// inside crud.Transactor.WithinTransaction with the callback's ctx.
	GetOwnedCollection(ctx context.Context, profileID, collectionID uuid.UUID, forUpdate bool) (entity.Collection, error)
}

type collectionRepository struct {
	crud.Repository[entity.Collection]
}

func NewCollectionRepository(base crud.Repository[entity.Collection]) CollectionRepository {
	return &collectionRepository{Repository: base}
}

// GetOwnedCollection filters by owner in the query itself. A zero-value id or
// profileID would otherwise drop that condition (see crud.WhereBySpec) and
// match any of the profile's (or anyone's) collections, hence the guard.
func (r *collectionRepository) GetOwnedCollection(ctx context.Context, profileID, collectionID uuid.UUID, forUpdate bool) (entity.Collection, error) {
	if collectionID == uuid.Nil || profileID == uuid.Nil {
		return entity.Collection{}, ungerr.NotFoundError(collectionNotFoundMsg)
	}

	c, err := r.FindFirst(ctx, crud.Specification[entity.Collection]{
		Model:     entity.Collection{BaseEntity: crud.BaseEntity{ID: collectionID}, ProfileID: profileID},
		ForUpdate: forUpdate,
	})
	if err != nil {
		return entity.Collection{}, err
	}
	if c.IsZero() {
		return entity.Collection{}, ungerr.NotFoundError(collectionNotFoundMsg)
	}

	return c, nil
}
