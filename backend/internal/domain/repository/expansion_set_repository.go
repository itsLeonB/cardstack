package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// ExpansionSetRepository is crud.Repository[entity.ExpansionSet] plus the
// queries go-crud's Specification cannot express: its Model drops zero-valued
// fields from the WHERE clause, so filters on cover_key = ”, published = false
// and source_logo_url = ” need raw SQL.
type ExpansionSetRepository interface {
	crud.Repository[entity.ExpansionSet]
	// ListMissingCovers returns the Expansion Sets that have a hosted original
	// (image_key) but no pre-sized cover (cover_key), ordered by code. A
	// non-empty code limits it to the set with that code.
	ListMissingCovers(ctx context.Context, code string) ([]entity.ExpansionSet, error)
	// ListUnpublished returns the Game's unpublished Expansion Sets, ordered by
	// code.
	ListUnpublished(ctx context.Context, gameID uuid.UUID) ([]entity.ExpansionSet, error)
	// FillSourceLogoURL sets source_logo_url of the set only while it is empty.
	// It changes no other column, not even updated_by, so the row keeps its
	// writer (docs/adr/0021).
	FillSourceLogoURL(ctx context.Context, id uuid.UUID, url string) error
	// InsertUnpublished inserts the set with published = false and returns it
	// with its ID. Insert cannot write that false: a zero Published takes the
	// column default (true).
	InsertUnpublished(ctx context.Context, set entity.ExpansionSet) (entity.ExpansionSet, error)
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

func (r *expansionSetRepository) ListUnpublished(ctx context.Context, gameID uuid.UUID) ([]entity.ExpansionSet, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var sets []entity.ExpansionSet
	if err := db.Where("game_id = ? AND published = false", gameID).Order("code, id").Find(&sets).Error; err != nil {
		return nil, ungerr.Wrap(err, "listing unpublished expansion sets")
	}
	return sets, nil
}

func (r *expansionSetRepository) FillSourceLogoURL(ctx context.Context, id uuid.UUID, url string) error {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}

	// UpdateColumn leaves updated_at alone, so the row changes in its logo column only.
	err = db.Model(&entity.ExpansionSet{}).
		Where("id = ? AND source_logo_url = ''", id).
		UpdateColumn("source_logo_url", url).Error
	if err != nil {
		return ungerr.Wrap(err, "filling expansion set logo")
	}
	return nil
}

func (r *expansionSetRepository) InsertUnpublished(ctx context.Context, set entity.ExpansionSet) (entity.ExpansionSet, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return entity.ExpansionSet{}, err
	}

	set.Published = false
	row := expansionSetInsert{ExpansionSet: set}
	if err := db.Create(&row).Error; err != nil {
		return entity.ExpansionSet{}, ungerr.Wrap(err, "inserting unpublished expansion set")
	}

	return row.ExpansionSet, nil
}

// expansionSetInsert shadows Published with a field that has no column
// default. Create would otherwise replace a zero Published with the default
// (true), so the false reaches the insert itself.
type expansionSetInsert struct {
	entity.ExpansionSet
	Published bool `gorm:"not null"`
}
