package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"gorm.io/gorm"
)

// BinderLeafRepository reads and writes a binder's Leaves.
type BinderLeafRepository interface {
	crud.Repository[entity.BinderLeaf]
	// FindLeafAt returns the Leaf at position (counted from 1), or the zero value
	// when there is none.
	FindLeafAt(ctx context.Context, collectionID uuid.UUID, position int) (entity.BinderLeaf, error)
	// LeafCounts returns the Leaf count of each binder in one grouped query. A
	// Collection with no Leaves is absent from the map.
	LeafCounts(ctx context.Context, collectionIDs []uuid.UUID) (map[uuid.UUID]int, error)
	CountLeaves(ctx context.Context, collectionID uuid.UUID) (int, error)
	DeleteLeaves(ctx context.Context, collectionID uuid.UUID) error
	// ShiftLeaves adds delta to the position of each Leaf at or after from. go-crud
	// has no update by predicate, so this is raw GORM; the position constraint is
	// deferred, which lets one statement move a whole run.
	ShiftLeaves(ctx context.Context, collectionID uuid.UUID, from, delta int) error
}

type binderLeafRepository struct {
	crud.Repository[entity.BinderLeaf]
}

func NewBinderLeafRepository(base crud.Repository[entity.BinderLeaf]) BinderLeafRepository {
	return &binderLeafRepository{Repository: base}
}

func (r *binderLeafRepository) FindLeafAt(ctx context.Context, collectionID uuid.UUID, position int) (entity.BinderLeaf, error) {
	// A zero collection ID or position would drop out of the Model.
	if collectionID == uuid.Nil || position < 1 {
		return entity.BinderLeaf{}, nil
	}

	return r.FindFirst(ctx, crud.Specification[entity.BinderLeaf]{
		Model: entity.BinderLeaf{CollectionID: collectionID, Position: position},
	})
}

func (r *binderLeafRepository) LeafCounts(ctx context.Context, collectionIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int, len(collectionIDs))
	if len(collectionIDs) == 0 {
		return counts, nil
	}

	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var rows []struct {
		CollectionID uuid.UUID
		Total        int
	}
	err = db.Model(&entity.BinderLeaf{}).
		Select("collection_id, COUNT(*) AS total").
		Where("collection_id IN ?", collectionIDs).
		Group("collection_id").
		Scan(&rows).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "counting binder leaves")
	}

	for _, row := range rows {
		counts[row.CollectionID] = row.Total
	}

	return counts, nil
}

func (r *binderLeafRepository) CountLeaves(ctx context.Context, collectionID uuid.UUID) (int, error) {
	counts, err := r.LeafCounts(ctx, []uuid.UUID{collectionID})
	if err != nil {
		return 0, err
	}

	return counts[collectionID], nil
}

func (r *binderLeafRepository) DeleteLeaves(ctx context.Context, collectionID uuid.UUID) error {
	// A nil collection ID would drop out of the Model and match every binder's Leaves.
	if collectionID == uuid.Nil {
		return nil
	}

	leaves, err := r.FindAll(ctx, crud.Specification[entity.BinderLeaf]{
		Model: entity.BinderLeaf{CollectionID: collectionID},
	})
	if err != nil {
		return err
	}
	if len(leaves) == 0 {
		return nil
	}

	return r.DeleteMany(ctx, leaves)
}

func (r *binderLeafRepository) ShiftLeaves(ctx context.Context, collectionID uuid.UUID, from, delta int) error {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}

	err = db.Model(&entity.BinderLeaf{}).
		Where("collection_id = ? AND position >= ?", collectionID, from).
		UpdateColumn("position", gorm.Expr("position + ?", delta)).
		Error
	if err != nil {
		return ungerr.Wrap(err, "shifting binder leaves")
	}

	return nil
}
