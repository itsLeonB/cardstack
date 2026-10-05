package repository

import (
	"context"

	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// ExpansionSetRepository is crud.Repository[entity.ExpansionSet] plus the one
// query go-crud's Specification cannot express: cover_key = ”, because
// Specification.Model drops zero-valued fields from the WHERE clause.
type ExpansionSetRepository interface {
	crud.Repository[entity.ExpansionSet]
	// ListMissingCovers returns the Expansion Sets that have a hosted original
	// (image_key) but no pre-sized cover (cover_key), ordered by code. A
	// non-empty code limits it to the set with that code.
	ListMissingCovers(ctx context.Context, code string) ([]entity.ExpansionSet, error)
}

type expansionSetRepository struct {
	crud.Repository[entity.ExpansionSet]
}

func NewExpansionSetRepository(base crud.Repository[entity.ExpansionSet]) ExpansionSetRepository {
	return &expansionSetRepository{Repository: base}
}

func (r *expansionSetRepository) ListMissingCovers(ctx context.Context, code string) ([]entity.ExpansionSet, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	query := db.Where("image_key <> '' AND cover_key = ''")
	if code != "" {
		query = query.Where("code = ?", code)
	}
	var sets []entity.ExpansionSet
	if err := query.Order("code, id").Find(&sets).Error; err != nil {
		return nil, ungerr.Wrap(err, "listing expansion sets missing a cover")
	}
	return sets, nil
}
