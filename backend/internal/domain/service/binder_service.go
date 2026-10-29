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
	maxLeaves = 100

	gridRangeMsg          = "rows and columns must each be from 1 to 6"
	leafCountRangeMsg     = "a binder holds from 1 to 100 Leaves"
	binderExistsMsg       = "this Collection is already a binder"
	collectionNotEmptyMsg = "a Layout can only be set on a Collection with no Inventory Entries"
	binderLayoutLockedMsg = "rows, columns and the Layout can change only while every Slot is empty"
	leafLimitMsg          = "a binder holds at most 100 Leaves"
	lastLeafMsg           = "a binder keeps at least one Leaf"
	leafNotFoundMsg       = "this binder has no Leaf with that number"
	placementMsg          = "placement must be before or after"
	leafInUseMsgFormat    = "page %d holds a card; clear it before removing its Leaf"
)

// BinderService turns a Collection into a binder and maintains its Layout and
// Leaves, and reads its Pages. Each write row-locks the Collection inside a
// transaction, so concurrent edits to one binder serialize. Writing Slots is
// BinderSlotService.
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
	layouts     repository.BinderLayoutRepository
	leaves      repository.BinderLeafRepository
	slots       repository.BinderSlotRepository
	entries     crud.Repository[entity.InventoryEntry]
	pages       BinderPageService
}

func NewBinderService(
	transactor crud.Transactor,
	collections repository.CollectionRepository,
	layouts repository.BinderLayoutRepository,
	leaves repository.BinderLeafRepository,
	slots repository.BinderSlotRepository,
	entries crud.Repository[entity.InventoryEntry],
	pages BinderPageService,
) BinderService {
	return &binderService{
		transactor:  transactor,
		collections: collections,
		layouts:     layouts,
		leaves:      leaves,
		slots:       slots,
		entries:     entries,
		pages:       pages,
	}
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

		isBinder, err := s.layouts.IsBinder(ctx, c.ID)
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

		layout, err = s.layouts.Insert(ctx, entity.CollectionLayout{CollectionID: c.ID, Rows: req.Rows, Columns: req.Columns})
		if err != nil {
			return err
		}

		leaves := make([]entity.BinderLeaf, req.LeafCount)
		for i := range leaves {
			leaves[i] = entity.BinderLeaf{CollectionID: c.ID, Position: i + 1}
		}
		if _, err := s.leaves.InsertMany(ctx, leaves); err != nil {
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

		layout, err = s.layouts.GetLayout(ctx, c.ID)
		if err != nil {
			return err
		}
		if err := s.requireNoSlots(ctx, c.ID); err != nil {
			return err
		}

		layout.Rows = req.Rows
		layout.Columns = req.Columns
		layout, err = s.layouts.Update(ctx, layout)
		if err != nil {
			return err
		}

		leafCount, err = s.leaves.CountLeaves(ctx, c.ID)
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

		layout, err := s.layouts.GetLayout(ctx, c.ID)
		if err != nil {
			return err
		}
		if err := s.requireNoSlots(ctx, c.ID); err != nil {
			return err
		}

		if err := s.leaves.DeleteLeaves(ctx, c.ID); err != nil {
			return err
		}
		return s.layouts.Delete(ctx, layout)
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

		layout, err = s.layouts.GetLayout(ctx, c.ID)
		if err != nil {
			return err
		}

		leafCount, err = s.leaves.CountLeaves(ctx, c.ID)
		if err != nil {
			return err
		}
		if req.Leaf < 1 || req.Leaf > leafCount {
			return ungerr.NotFoundError(leafNotFoundMsg)
		}
		if leafCount >= maxLeaves {
			return ungerr.BadRequestError(leafLimitMsg)
		}

		position := req.Leaf
		if req.Placement == dto.BinderPlacementAfter {
			position++
		}
		if err := s.leaves.ShiftLeaves(ctx, c.ID, position, 1); err != nil {
			return err
		}
		if _, err := s.leaves.InsertMany(ctx, []entity.BinderLeaf{{CollectionID: c.ID, Position: position}}); err != nil {
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

		layout, err = s.layouts.GetLayout(ctx, c.ID)
		if err != nil {
			return err
		}

		leaf, err := s.leaves.FindLeafAt(ctx, c.ID, req.Leaf)
		if err != nil {
			return err
		}
		if leaf.IsZero() {
			return ungerr.NotFoundError(leafNotFoundMsg)
		}

		leafCount, err = s.leaves.CountLeaves(ctx, c.ID)
		if err != nil {
			return err
		}
		if leafCount <= 1 {
			return ungerr.BadRequestError(lastLeafMsg)
		}

		for _, side := range []string{dto.BinderSideFront, dto.BinderSideBack} {
			occupied, err := s.slots.HasSlotsOnPage(ctx, leaf.ID, side)
			if err != nil {
				return err
			}
			if occupied {
				msg := fmt.Sprintf(leafInUseMsgFormat, mapper.PageOf(req.Leaf, side))
				return apperr.WithCode(ungerr.ConflictError(msg), apperr.CodeBinderLeafInUse)
			}
		}

		if err := s.leaves.Delete(ctx, leaf); err != nil {
			return err
		}
		if err := s.leaves.ShiftLeaves(ctx, c.ID, req.Leaf+1, -1); err != nil {
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

	return s.pages.Page(ctx, dto.BinderPageReadRequest{CollectionID: c.ID, Page: req.Page, IncludeUnpublished: true})
}

// WriteSlots is the one place Slot writes are checked and applied. It validates
// every change before it reads the Slots, then reads the Slots once and applies
// the changes in order, so a later change sees an earlier one on the same Slot.
// Each Card whose Slots changed gets its entry reconciled once at the end.
// pageLeaf returns the Leaf and face behind a Page number, or the not-found
// AppError when no Leaf has that position.
// leafAt returns the Leaf at a position, or the not-found AppError when there is none.
// reconcileEntry sets a Card's Inventory Entry to the number of Slots holding
// it, so the entry is created, changed or removed to match the Slots.
// resolveCard returns the Card with the id, or the not-found AppError. A nil id
// is refused before the query, since an empty Model would match any Card.
// requireNoSlots refuses a Layout change while any Slot is occupied, since the
// cards placed on the binder would no longer sit on a matching grid.
func (s *binderService) requireNoSlots(ctx context.Context, collectionID uuid.UUID) error {
	cards, unidentified, err := s.slots.CountSlots(ctx, collectionID, true)
	if err != nil {
		return err
	}
	if cards+unidentified > 0 {
		return apperr.WithCode(ungerr.ConflictError(binderLayoutLockedMsg), apperr.CodeBinderLayoutLocked)
	}

	return nil
}

func validateGrid(rows, columns int) error {
	if !inRange(rows, entity.MaxBinderGridSide) || !inRange(columns, entity.MaxBinderGridSide) {
		return ungerr.BadRequestError(gridRangeMsg)
	}

	return nil
}

// inRange reports whether v is a 1-based position or size up to limit. It is the
// one bounds check, shared by a Layout's size and a Slot's address.
func inRange(v, limit int) bool {
	return v >= 1 && v <= limit
}
