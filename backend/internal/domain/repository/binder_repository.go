package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"gorm.io/gorm"
)

const layoutNotFoundMsg = "this Collection is not a binder"

// BinderRepository reads and writes a binder: its Layout (the embedded
// repository, since the Layout is the binder's root row), its Leaves and its
// occupied Slots. Leaves need shifting updates and Slots need grouped counts,
// which go-crud has no call for, hence the raw GORM.
type BinderRepository interface {
	crud.Repository[entity.CollectionLayout]
	// GetLayout returns the Collection's Layout, or the 404 AppError when it is
	// not a binder. It never returns a zero value with a nil error.
	GetLayout(ctx context.Context, collectionID uuid.UUID) (entity.CollectionLayout, error)
	// FindLeafAt returns the Leaf at position (counted from 1), or the zero value
	// when there is none.
	FindLeafAt(ctx context.Context, collectionID uuid.UUID, position int) (entity.BinderLeaf, error)
	CountLeaves(ctx context.Context, collectionID uuid.UUID) (int, error)
	InsertLeaves(ctx context.Context, leaves []entity.BinderLeaf) error
	DeleteLeaf(ctx context.Context, leaf entity.BinderLeaf) error
	DeleteLeaves(ctx context.Context, collectionID uuid.UUID) error
	// ShiftLeaves adds delta to the position of every Leaf at or after from. The
	// position constraint is deferred, so one statement can shift a whole run.
	ShiftLeaves(ctx context.Context, collectionID uuid.UUID, from, delta int) error
	// CountSlots returns, across the whole binder, how many Slots hold a Card and
	// how many are Unidentified.
	CountSlots(ctx context.Context, collectionID uuid.UUID) (cards, unidentified int, err error)
	// CountSlotsOnPage counts the occupied Slots of one face of a Leaf. side is 0
	// for the front and 1 for the back.
	CountSlotsOnPage(ctx context.Context, leafID uuid.UUID, side int) (int, error)
	// ListSlotsOnPage returns the occupied Slots of one face of a Leaf, with their
	// Cards, Rarities and Expansion Sets loaded.
	ListSlotsOnPage(ctx context.Context, leafID uuid.UUID, side int) ([]entity.BinderSlot, error)
}

type binderRepository struct {
	crud.Repository[entity.CollectionLayout]
}

func NewBinderRepository(base crud.Repository[entity.CollectionLayout]) BinderRepository {
	return &binderRepository{Repository: base}
}

func (r *binderRepository) GetLayout(ctx context.Context, collectionID uuid.UUID) (entity.CollectionLayout, error) {
	if collectionID == uuid.Nil {
		return entity.CollectionLayout{}, ungerr.NotFoundError(layoutNotFoundMsg)
	}

	layout, err := r.FindFirst(ctx, crud.Specification[entity.CollectionLayout]{
		Model: entity.CollectionLayout{CollectionID: collectionID},
	})
	if err != nil {
		return entity.CollectionLayout{}, err
	}
	if layout.IsZero() {
		return entity.CollectionLayout{}, ungerr.NotFoundError(layoutNotFoundMsg)
	}

	return layout, nil
}

func (r *binderRepository) FindLeafAt(ctx context.Context, collectionID uuid.UUID, position int) (entity.BinderLeaf, error) {
	// A zero position would drop out of a crud.Specification's Model, so guard it.
	if collectionID == uuid.Nil || position < 1 {
		return entity.BinderLeaf{}, nil
	}

	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return entity.BinderLeaf{}, err
	}

	var leaf entity.BinderLeaf
	err = db.Where("collection_id = ? AND position = ?", collectionID, position).Limit(1).Find(&leaf).Error
	if err != nil {
		return entity.BinderLeaf{}, ungerr.Wrap(err, "finding binder leaf")
	}

	return leaf, nil
}

func (r *binderRepository) CountLeaves(ctx context.Context, collectionID uuid.UUID) (int, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return 0, err
	}

	var n int64
	err = db.Model(&entity.BinderLeaf{}).Where("collection_id = ?", collectionID).Count(&n).Error
	if err != nil {
		return 0, ungerr.Wrap(err, "counting binder leaves")
	}

	return int(n), nil
}

func (r *binderRepository) InsertLeaves(ctx context.Context, leaves []entity.BinderLeaf) error {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}

	if err := db.Create(&leaves).Error; err != nil {
		return ungerr.Wrap(err, "inserting binder leaves")
	}

	return nil
}

func (r *binderRepository) DeleteLeaf(ctx context.Context, leaf entity.BinderLeaf) error {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}

	if err := db.Delete(&leaf).Error; err != nil {
		return ungerr.Wrap(err, "deleting binder leaf")
	}

	return nil
}

func (r *binderRepository) DeleteLeaves(ctx context.Context, collectionID uuid.UUID) error {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}

	if err := db.Where("collection_id = ?", collectionID).Delete(&entity.BinderLeaf{}).Error; err != nil {
		return ungerr.Wrap(err, "deleting binder leaves")
	}

	return nil
}

func (r *binderRepository) ShiftLeaves(ctx context.Context, collectionID uuid.UUID, from, delta int) error {
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

func (r *binderRepository) CountSlots(ctx context.Context, collectionID uuid.UUID) (int, int, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return 0, 0, err
	}

	var counts struct {
		Cards        int64
		Unidentified int64
	}
	err = db.Model(&entity.BinderSlot{}).
		Select("COUNT(card_id) AS cards, COUNT(*) FILTER (WHERE unidentified) AS unidentified").
		Where("collection_id = ?", collectionID).
		Scan(&counts).
		Error
	if err != nil {
		return 0, 0, ungerr.Wrap(err, "counting binder slots")
	}

	return int(counts.Cards), int(counts.Unidentified), nil
}

func (r *binderRepository) CountSlotsOnPage(ctx context.Context, leafID uuid.UUID, side int) (int, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return 0, err
	}

	var n int64
	err = db.Model(&entity.BinderSlot{}).Where("leaf_id = ? AND side = ?", leafID, side).Count(&n).Error
	if err != nil {
		return 0, ungerr.Wrap(err, "counting page slots")
	}

	return int(n), nil
}

func (r *binderRepository) ListSlotsOnPage(ctx context.Context, leafID uuid.UUID, side int) ([]entity.BinderSlot, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var slots []entity.BinderSlot
	err = db.Preload("Card.Rarity").
		Preload("Card.ExpansionSet").
		Where("leaf_id = ? AND side = ?", leafID, side).
		Find(&slots).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "listing page slots")
	}

	return slots, nil
}
