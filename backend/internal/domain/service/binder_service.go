package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/apperr"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

const (
	maxGridSide = 6
	maxLeaves   = 100

	gridRangeMsg          = "rows and columns must each be from 1 to 6"
	leafCountRangeMsg     = "a binder holds from 1 to 100 Leaves"
	binderExistsMsg       = "this Collection is already a binder"
	collectionNotEmptyMsg = "a Layout can only be set on a Collection with no Inventory Entries"
	binderLayoutLockedMsg = "rows, columns and the Layout can change only while every Slot is empty"
	leafLimitMsg          = "a binder holds at most 100 Leaves"
	lastLeafMsg           = "a binder keeps at least one Leaf"
	leafNotFoundMsg       = "this binder has no Leaf with that number"
	pageNotFoundMsg       = "this binder has no Page with that number"
	placementMsg          = "placement must be before or after"
	leafInUseMsgFormat    = "page %d holds a card; clear it before removing its Leaf"
)

// BinderService turns a Collection into a binder and maintains its Layout,
// Leaves and Pages. Every write runs in a transaction that row-locks the
// Collection first, so concurrent edits to one binder serialize.
type BinderService interface {
	CreateBinder(ctx context.Context, req dto.BinderLayoutRequest) (dto.BinderLayout, error)
	ChangeGrid(ctx context.Context, req dto.BinderGridRequest) (dto.BinderLayout, error)
	RemoveBinder(ctx context.Context, req dto.CollectionLookup) error
	AddLeaf(ctx context.Context, req dto.BinderLeafRequest) (dto.BinderLayout, error)
	RemoveLeaf(ctx context.Context, req dto.BinderLeafLookup) (dto.BinderLayout, error)
	GetPage(ctx context.Context, req dto.BinderPageLookup) (dto.BinderPage, error)
}

type binderService struct {
	transactor  crud.Transactor
	collections repository.CollectionRepository
	binders     repository.BinderRepository
	entries     crud.Repository[entity.InventoryEntry]
	images      mapper.ImageHost
}

func NewBinderService(
	transactor crud.Transactor,
	collections repository.CollectionRepository,
	binders repository.BinderRepository,
	entries crud.Repository[entity.InventoryEntry],
	images mapper.ImageHost,
) BinderService {
	return &binderService{transactor: transactor, collections: collections, binders: binders, entries: entries, images: images}
}

func (s *binderService) CreateBinder(ctx context.Context, req dto.BinderLayoutRequest) (dto.BinderLayout, error) {
	if err := validateGrid(req.Rows, req.Columns); err != nil {
		return dto.BinderLayout{}, err
	}
	if req.LeafCount < 1 || req.LeafCount > maxLeaves {
		return dto.BinderLayout{}, ungerr.BadRequestError(leafCountRangeMsg)
	}

	var layout entity.CollectionLayout
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}

		isBinder, err := s.collections.IsBinder(ctx, c.ID)
		if err != nil {
			return err
		}
		if isBinder {
			return ungerr.ConflictError(binderExistsMsg)
		}

		entry, err := s.entries.FindFirst(ctx, crud.Specification[entity.InventoryEntry]{
			Model: entity.InventoryEntry{CollectionID: c.ID},
		})
		if err != nil {
			return err
		}
		if !entry.IsZero() {
			return apperr.WithCode(ungerr.ConflictError(collectionNotEmptyMsg), apperr.CodeCollectionNotEmpty)
		}

		layout, err = s.binders.Insert(ctx, entity.CollectionLayout{CollectionID: c.ID, Rows: req.Rows, Columns: req.Columns})
		if err != nil {
			return err
		}

		leaves := make([]entity.BinderLeaf, req.LeafCount)
		for i := range leaves {
			leaves[i] = entity.BinderLeaf{CollectionID: c.ID, Position: i + 1}
		}
		if err := s.binders.InsertLeaves(ctx, leaves); err != nil {
			return err
		}

		// The Slot count is the binder's capacity, so the Collection's own cap is cleared.
		c.MaxCardCount = 0
		_, err = s.collections.Update(ctx, c)
		return err
	})
	if err != nil {
		return dto.BinderLayout{}, err
	}

	return mapper.ToBinderLayout(layout, req.LeafCount), nil
}

func (s *binderService) ChangeGrid(ctx context.Context, req dto.BinderGridRequest) (dto.BinderLayout, error) {
	if err := validateGrid(req.Rows, req.Columns); err != nil {
		return dto.BinderLayout{}, err
	}

	var layout entity.CollectionLayout
	var leafCount int
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}

		layout, err = s.binders.GetLayout(ctx, c.ID)
		if err != nil {
			return err
		}
		if err := s.requireNoSlots(ctx, c.ID); err != nil {
			return err
		}

		layout.Rows = req.Rows
		layout.Columns = req.Columns
		layout, err = s.binders.Update(ctx, layout)
		if err != nil {
			return err
		}

		leafCount, err = s.binders.CountLeaves(ctx, c.ID)
		return err
	})
	if err != nil {
		return dto.BinderLayout{}, err
	}

	return mapper.ToBinderLayout(layout, leafCount), nil
}

func (s *binderService) RemoveBinder(ctx context.Context, req dto.CollectionLookup) error {
	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.ID, true)
		if err != nil {
			return err
		}

		layout, err := s.binders.GetLayout(ctx, c.ID)
		if err != nil {
			return err
		}
		if err := s.requireNoSlots(ctx, c.ID); err != nil {
			return err
		}

		if err := s.binders.DeleteLeaves(ctx, c.ID); err != nil {
			return err
		}
		return s.binders.Delete(ctx, layout)
	})
}

func (s *binderService) AddLeaf(ctx context.Context, req dto.BinderLeafRequest) (dto.BinderLayout, error) {
	if req.Placement != dto.BinderPlacementBefore && req.Placement != dto.BinderPlacementAfter {
		return dto.BinderLayout{}, ungerr.BadRequestError(placementMsg)
	}

	var layout entity.CollectionLayout
	var leafCount int
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}

		layout, err = s.binders.GetLayout(ctx, c.ID)
		if err != nil {
			return err
		}

		leafCount, err = s.binders.CountLeaves(ctx, c.ID)
		if err != nil {
			return err
		}
		if leafCount >= maxLeaves {
			return ungerr.BadRequestError(leafLimitMsg)
		}
		if req.Leaf < 1 || req.Leaf > leafCount {
			return ungerr.NotFoundError(leafNotFoundMsg)
		}

		position := req.Leaf
		if req.Placement == dto.BinderPlacementAfter {
			position++
		}
		if err := s.binders.ShiftLeaves(ctx, c.ID, position, 1); err != nil {
			return err
		}
		if err := s.binders.InsertLeaves(ctx, []entity.BinderLeaf{{CollectionID: c.ID, Position: position}}); err != nil {
			return err
		}

		leafCount++
		return nil
	})
	if err != nil {
		return dto.BinderLayout{}, err
	}

	return mapper.ToBinderLayout(layout, leafCount), nil
}

func (s *binderService) RemoveLeaf(ctx context.Context, req dto.BinderLeafLookup) (dto.BinderLayout, error) {
	var layout entity.CollectionLayout
	var leafCount int
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}

		layout, err = s.binders.GetLayout(ctx, c.ID)
		if err != nil {
			return err
		}

		leafCount, err = s.binders.CountLeaves(ctx, c.ID)
		if err != nil {
			return err
		}
		if leafCount <= 1 {
			return ungerr.BadRequestError(lastLeafMsg)
		}

		leaf, err := s.binders.FindLeafAt(ctx, c.ID, req.Leaf)
		if err != nil {
			return err
		}
		if leaf.IsZero() {
			return ungerr.NotFoundError(leafNotFoundMsg)
		}

		// Front is Page 2n-1 and back is Page 2n, for Leaf n.
		for side := range 2 {
			occupied, err := s.binders.CountSlotsOnPage(ctx, leaf.ID, side)
			if err != nil {
				return err
			}
			if occupied > 0 {
				page := 2*req.Leaf - 1 + side
				return apperr.WithCode(ungerr.ConflictError(fmt.Sprintf(leafInUseMsgFormat, page)), apperr.CodeBinderLeafInUse)
			}
		}

		if err := s.binders.DeleteLeaf(ctx, leaf); err != nil {
			return err
		}
		if err := s.binders.ShiftLeaves(ctx, c.ID, req.Leaf+1, -1); err != nil {
			return err
		}

		leafCount--
		return nil
	})
	if err != nil {
		return dto.BinderLayout{}, err
	}

	return mapper.ToBinderLayout(layout, leafCount), nil
}

func (s *binderService) GetPage(ctx context.Context, req dto.BinderPageLookup) (dto.BinderPage, error) {
	c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, false)
	if err != nil {
		return dto.BinderPage{}, err
	}

	layout, err := s.binders.GetLayout(ctx, c.ID)
	if err != nil {
		return dto.BinderPage{}, err
	}

	leafCount, err := s.binders.CountLeaves(ctx, c.ID)
	if err != nil {
		return dto.BinderPage{}, err
	}
	if req.Page < 1 || req.Page > 2*leafCount {
		return dto.BinderPage{}, ungerr.NotFoundError(pageNotFoundMsg)
	}

	leafNumber := (req.Page + 1) / 2
	side, sideName := 0, dto.BinderSideFront
	if req.Page%2 == 0 {
		side, sideName = 1, dto.BinderSideBack
	}

	leaf, err := s.binders.FindLeafAt(ctx, c.ID, leafNumber)
	if err != nil {
		return dto.BinderPage{}, err
	}
	if leaf.IsZero() {
		return dto.BinderPage{}, ungerr.NotFoundError(pageNotFoundMsg)
	}

	slots, err := s.binders.ListSlotsOnPage(ctx, leaf.ID, side)
	if err != nil {
		return dto.BinderPage{}, err
	}

	cards, unidentified, err := s.binders.CountSlots(ctx, c.ID)
	if err != nil {
		return dto.BinderPage{}, err
	}

	return dto.BinderPage{
		Page:              req.Page,
		Leaf:              leafNumber,
		Side:              sideName,
		Layout:            mapper.ToBinderLayout(layout, leafCount),
		CardTotal:         cards + unidentified,
		UnidentifiedCount: unidentified,
		Slots:             mapper.ToBinderSlots(s.images, layout, slots),
	}, nil
}

// requireNoSlots refuses a change to the Layout while any Slot is occupied,
// since the cards placed on the binder would no longer sit on a matching grid.
func (s *binderService) requireNoSlots(ctx context.Context, collectionID uuid.UUID) error {
	cards, unidentified, err := s.binders.CountSlots(ctx, collectionID)
	if err != nil {
		return err
	}
	if cards+unidentified > 0 {
		return apperr.WithCode(ungerr.ConflictError(binderLayoutLockedMsg), apperr.CodeBinderLayoutLocked)
	}

	return nil
}

func validateGrid(rows, columns int) error {
	if rows < 1 || rows > maxGridSide || columns < 1 || columns > maxGridSide {
		return ungerr.BadRequestError(gridRangeMsg)
	}

	return nil
}
