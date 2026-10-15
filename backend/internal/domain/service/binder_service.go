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
	maxGridSide = dto.BinderMaxGridSide
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
	slotOutOfGridMsg      = "row and column must lie within the Layout's rows and columns"
)

// BinderService turns a Collection into a binder and maintains its Layout,
// Leaves and Pages. Each write row-locks the Collection inside a transaction, so
// concurrent edits to one binder serialize.
type BinderService interface {
	CreateBinder(ctx context.Context, req dto.BinderLayoutRequest) (dto.BinderLayout, error)
	ChangeGrid(ctx context.Context, req dto.BinderGridRequest) (dto.BinderLayout, error)
	RemoveBinder(ctx context.Context, req dto.CollectionLookup) error
	AddLeaf(ctx context.Context, req dto.BinderLeafRequest) (dto.BinderLayout, error)
	RemoveLeaf(ctx context.Context, req dto.BinderLeafLookup) (dto.BinderLayout, error)
	GetPage(ctx context.Context, req dto.BinderPageLookup) (dto.BinderPage, error)
	// WriteSlot sets one Slot and keeps the Inventory Entries of the Cards it
	// held or now holds in step, in one transaction. It joins a transaction
	// already open on ctx, so a later operation can call it inside its own.
	WriteSlot(ctx context.Context, req dto.BinderSlotWrite) error
	// MoveSlot moves a card to another Slot of the same binder, swapping when the
	// target is filled. Quantities do not change.
	MoveSlot(ctx context.Context, req dto.BinderSlotMove) error
	// ReplacePage sets every Slot of one Page to the state requested for it. A
	// Card the server does not know is stored Unidentified and reported as such.
	ReplacePage(ctx context.Context, req dto.BinderPageReplace) (dto.BinderPageReplaceResult, error)
}

type binderService struct {
	transactor  crud.Transactor
	collections repository.CollectionRepository
	layouts     repository.BinderLayoutRepository
	leaves      repository.BinderLeafRepository
	slots       repository.BinderSlotRepository
	cards       crud.Repository[entity.Card]
	entries     crud.Repository[entity.InventoryEntry]
	images      mapper.ImageHost
}

func NewBinderService(
	transactor crud.Transactor,
	collections repository.CollectionRepository,
	layouts repository.BinderLayoutRepository,
	leaves repository.BinderLeafRepository,
	slots repository.BinderSlotRepository,
	cards crud.Repository[entity.Card],
	entries crud.Repository[entity.InventoryEntry],
	images mapper.ImageHost,
) BinderService {
	return &binderService{
		transactor:  transactor,
		collections: collections,
		layouts:     layouts,
		leaves:      leaves,
		slots:       slots,
		cards:       cards,
		entries:     entries,
		images:      images,
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

	layout, err := s.layouts.GetLayout(ctx, c.ID)
	if err != nil {
		return dto.BinderPage{}, err
	}

	leafCount, err := s.leaves.CountLeaves(ctx, c.ID)
	if err != nil {
		return dto.BinderPage{}, err
	}
	if req.Page < 1 || req.Page > 2*leafCount {
		return dto.BinderPage{}, ungerr.NotFoundError(pageNotFoundMsg)
	}

	leaf, side, err := s.pageLeaf(ctx, c.ID, req.Page)
	if err != nil {
		return dto.BinderPage{}, err
	}

	slots, err := s.slots.ListSlotsOnPage(ctx, leaf.ID, side)
	if err != nil {
		return dto.BinderPage{}, err
	}

	cards, unidentified, err := s.slots.CountSlots(ctx, c.ID)
	if err != nil {
		return dto.BinderPage{}, err
	}

	return mapper.ToBinderPage(s.images, req.Page, layout, leafCount, slots, cards+unidentified, unidentified), nil
}

func (s *binderService) WriteSlot(ctx context.Context, req dto.BinderSlotWrite) error {
	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}

		layout, err := s.layouts.GetLayout(ctx, c.ID)
		if err != nil {
			return err
		}
		target := req.Target
		if err := checkSlotInGrid(layout, target); err != nil {
			return err
		}

		leaf, side, err := s.pageLeaf(ctx, c.ID, target.Page)
		if err != nil {
			return err
		}

		var cardID *uuid.UUID
		if req.State == dto.BinderSlotCard {
			if _, err := s.resolveCard(ctx, req.CardID); err != nil {
				return err
			}
			cardID = &req.CardID
		}

		previous, err := s.writeSlotRow(ctx, c, leaf, side, target, req.State, cardID)
		if err != nil {
			return err
		}

		return s.reconcileCards(ctx, c.ID, []*uuid.UUID{previous, cardID})
	})
}

// pageLeaf returns the Leaf and face behind a Page number, or the not-found
// AppError when no Leaf has that position.
func (s *binderService) pageLeaf(ctx context.Context, collectionID uuid.UUID, page int) (entity.BinderLeaf, string, error) {
	leafNumber, side := mapper.LeafOf(page)
	leaf, err := s.leaves.FindLeafAt(ctx, collectionID, leafNumber)
	if err != nil {
		return entity.BinderLeaf{}, "", err
	}
	if leaf.IsZero() {
		return entity.BinderLeaf{}, "", ungerr.NotFoundError(pageNotFoundMsg)
	}

	return leaf, side, nil
}

// reconcileEntry sets a Card's Inventory Entry to the number of Slots holding
// it, so the entry is created, changed or removed to match the Slots.
func (s *binderService) reconcileEntry(ctx context.Context, collectionID, cardID uuid.UUID) error {
	count, err := s.slots.CountCardSlots(ctx, collectionID, cardID)
	if err != nil {
		return err
	}

	entry, err := s.entries.FindFirst(ctx, crud.Specification[entity.InventoryEntry]{
		Model:     entity.InventoryEntry{CollectionID: collectionID, CardID: cardID},
		ForUpdate: true,
	})
	if err != nil {
		return err
	}

	switch {
	case count == 0 && entry.IsZero():
		return nil
	case count == 0:
		return s.entries.Delete(ctx, entry)
	case entry.IsZero():
		_, err = s.entries.Insert(ctx, entity.InventoryEntry{CollectionID: collectionID, CardID: cardID, Quantity: count})
	case entry.Quantity != count:
		entry.Quantity = count
		_, err = s.entries.Update(ctx, entry)
	}

	return err
}

// resolveCard returns the Card with the id, or the not-found AppError. A nil id
// is refused before the query, since an empty Model would match any Card.
func (s *binderService) resolveCard(ctx context.Context, cardID uuid.UUID) (entity.Card, error) {
	if cardID == uuid.Nil {
		return entity.Card{}, ungerr.NotFoundError(cardNotFoundMsg)
	}

	card, err := s.cards.FindFirst(ctx, crud.Specification[entity.Card]{
		Model: entity.Card{BaseEntity: crud.BaseEntity{ID: cardID}},
	})
	if err != nil {
		return entity.Card{}, err
	}
	if card.IsZero() {
		return entity.Card{}, ungerr.NotFoundError(cardNotFoundMsg)
	}

	return card, nil
}

// requireNoSlots refuses a Layout change while any Slot is occupied, since the
// cards placed on the binder would no longer sit on a matching grid.
func (s *binderService) requireNoSlots(ctx context.Context, collectionID uuid.UUID) error {
	cards, unidentified, err := s.slots.CountSlots(ctx, collectionID)
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
