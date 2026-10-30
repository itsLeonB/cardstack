package service

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/apperr"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ezutil/v2"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

const (
	entryNotFoundMsg        = "card is not in this collection"
	entryExistsMsg          = "card is already in this collection"
	capacityExceededMsg     = "collection capacity limit exceeded"
	quantityNotPositiveMsg  = "quantity must be positive"
	duplicateCardMsg        = "duplicate cardId in request"
	binderQuantityLockedMsg = "a binder's quantities change only by placing or clearing Slots"
	moveSameCollectionMsg   = "a move needs two different collections"
	moveToBinderMsg         = "a binder cannot receive a move: a move chooses no Slot"
	moveCapacityMsg         = "this move would exceed the collection's limit of %s by %s"
)

// InventoryService scopes every method to the request's ProfileID; another
// profile's Collection is reported as "collection not found", same as a
// missing one.
type InventoryService interface {
	// List pages the Collection's Cards matching req.Filter, with quantities.
	List(ctx context.Context, req dto.InventoryListRequest) ([]dto.InventoryItem, dto.PaginationMeta, error)
	// ListFacets returns filter options computed only from the Collection's Cards.
	ListFacets(ctx context.Context, req dto.InventoryListRequest) (dto.CatalogFacets, error)
	// ListCardHoldings returns the profile's own Collections holding the Card
	// (empty when none, or when the Card is unknown).
	// ListMasterInventory returns the profile's Cards with the quantity owned
	// summed across all its Collections, computed on read.
	ListMasterInventory(ctx context.Context, req dto.MasterInventoryRequest) ([]dto.InventoryItem, dto.PaginationMeta, error)
	// ListMasterFacets returns filter options computed only from the profile's owned Cards.
	ListMasterFacets(ctx context.Context, req dto.MasterInventoryRequest) (dto.CatalogFacets, error)
	ListCardHoldings(ctx context.Context, req dto.CardHoldingsRequest) ([]dto.CardHolding, error)
	Add(ctx context.Context, req dto.InventoryEntryRequest) (dto.InventoryEntry, error)
	UpdateQuantity(ctx context.Context, req dto.InventoryEntryRequest) (dto.InventoryEntry, error)
	Remove(ctx context.Context, req dto.InventoryEntryLookup) error
	// Move takes the listed Cards, every copy of each, from the source
	// Collection into the destination, in one transaction. A refusal changes
	// neither Collection.
	Move(ctx context.Context, req dto.InventoryMoveRequest) (dto.InventoryMoveResult, error)
	// BulkUpdate applies the items in order in one transaction, declining (not
	// failing) an item that is over capacity or names an unknown Card.
	BulkUpdate(ctx context.Context, req dto.InventoryBulkUpdateRequest) ([]dto.InventoryChangeResult, error)
}

type inventoryService struct {
	transactor  crud.Transactor
	collections repository.CollectionRepository
	layouts     repository.BinderLayoutRepository
	slots       repository.BinderSlotRepository
	entries     repository.InventoryRepository
	catalog     repository.CatalogRepository
	cards       CardLookup
	images      mapper.ImageHost
	binderSlots BinderSlotService
}

func NewInventoryService(
	transactor crud.Transactor,
	collections repository.CollectionRepository,
	layouts repository.BinderLayoutRepository,
	slots repository.BinderSlotRepository,
	entries repository.InventoryRepository,
	catalog repository.CatalogRepository,
	cards CardLookup,
	images mapper.ImageHost,
	binderSlots BinderSlotService,
) InventoryService {
	return &inventoryService{transactor: transactor, collections: collections, layouts: layouts, slots: slots, entries: entries, catalog: catalog, cards: cards, images: images, binderSlots: binderSlots}
}

func (s *inventoryService) List(ctx context.Context, req dto.InventoryListRequest) ([]dto.InventoryItem, dto.PaginationMeta, error) {
	if _, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, false); err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	repoFilter, page, limit := pagedRepoFilter(req.Filter)
	repoFilter.CollectionID = req.CollectionID

	results, total, err := s.catalog.SearchCards(ctx, repoFilter)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	items := ezutil.MapSlice(results, func(r repository.CardResult) dto.InventoryItem { return mapper.ToInventoryItem(s.images, r) })
	// Only a binder has Slots, so a plain Collection's list finds none and omits positions.
	if len(items) > 0 {
		cardIDs := ezutil.MapSlice(items, func(i dto.InventoryItem) uuid.UUID { return i.Card.ID })
		slots, err := s.slots.ListOccupiedSlots(ctx, req.CollectionID, cardIDs)
		if err != nil {
			return nil, dto.PaginationMeta{}, err
		}
		positions := mapper.ToCardPositions(slots)
		for i := range items {
			items[i].Positions = positions[items[i].Card.ID]
		}
	}

	return items, dto.PaginationMeta{Total: int(total), Page: page, Limit: limit}, nil
}

func (s *inventoryService) ListFacets(ctx context.Context, req dto.InventoryListRequest) (dto.CatalogFacets, error) {
	if _, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, false); err != nil {
		return dto.CatalogFacets{}, err
	}

	f := mapper.ToRepoCardFilter(req.Filter)
	f.CollectionID = req.CollectionID

	facets, err := s.catalog.ListCardFacets(ctx, f)
	if err != nil {
		return dto.CatalogFacets{}, err
	}

	return mapper.ToCatalogFacets(s.images, facets), nil
}

func (s *inventoryService) ListMasterInventory(ctx context.Context, req dto.MasterInventoryRequest) ([]dto.InventoryItem, dto.PaginationMeta, error) {
	// A nil ProfileID would drop the scoping and list the whole catalog.
	if req.ProfileID == uuid.Nil {
		return nil, dto.PaginationMeta{}, ungerr.UnauthorizedError("missing profile")
	}

	repoFilter, page, limit := pagedRepoFilter(req.Filter)
	repoFilter.ProfileID = req.ProfileID

	results, total, err := s.catalog.SearchCards(ctx, repoFilter)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	return ezutil.MapSlice(results, func(r repository.CardResult) dto.InventoryItem { return mapper.ToInventoryItem(s.images, r) }), dto.PaginationMeta{Total: int(total), Page: page, Limit: limit}, nil
}

func (s *inventoryService) ListMasterFacets(ctx context.Context, req dto.MasterInventoryRequest) (dto.CatalogFacets, error) {
	// A nil ProfileID would drop the scoping and facet the whole catalog.
	if req.ProfileID == uuid.Nil {
		return dto.CatalogFacets{}, ungerr.UnauthorizedError("missing profile")
	}

	f := mapper.ToRepoCardFilter(req.Filter)
	f.ProfileID = req.ProfileID

	facets, err := s.catalog.ListCardFacets(ctx, f)
	if err != nil {
		return dto.CatalogFacets{}, err
	}

	return mapper.ToCatalogFacets(s.images, facets), nil
}

func (s *inventoryService) ListCardHoldings(ctx context.Context, req dto.CardHoldingsRequest) ([]dto.CardHolding, error) {
	holdings, err := s.entries.ListHoldings(ctx, req.ProfileID, req.CardID)
	if err != nil {
		return nil, err
	}

	return ezutil.MapSlice(holdings, mapper.ToCardHolding), nil
}

// refuseBinder refuses a quantity write on a binder, whose quantities change
// only by placing or clearing Slots (docs/adr/0020).
func (s *inventoryService) refuseBinder(ctx context.Context, collectionID uuid.UUID) error {
	isBinder, err := s.layouts.IsBinder(ctx, collectionID)
	if err != nil {
		return err
	}
	if isBinder {
		return apperr.WithCode(ungerr.ConflictError(binderQuantityLockedMsg), apperr.CodeBinderQuantityLocked)
	}

	return nil
}

// findEntry returns the Card's row-locked entry, or the zero value when absent.
// A nil card id would drop that condition (see crud.WhereBySpec), hence the guard.
func (s *inventoryService) findEntry(ctx context.Context, collectionID, cardID uuid.UUID) (entity.InventoryEntry, error) {
	if cardID == uuid.Nil {
		return entity.InventoryEntry{}, nil
	}

	return s.entries.FindFirst(ctx, crud.Specification[entity.InventoryEntry]{
		Model:     entity.InventoryEntry{CollectionID: collectionID, CardID: cardID},
		ForUpdate: true,
	})
}

// getEntry is findEntry, or the not-found 404 when absent.
func (s *inventoryService) getEntry(ctx context.Context, collectionID, cardID uuid.UUID) (entity.InventoryEntry, error) {
	entry, err := s.findEntry(ctx, collectionID, cardID)
	if err != nil {
		return entity.InventoryEntry{}, err
	}
	if entry.IsZero() {
		return entity.InventoryEntry{}, ungerr.NotFoundError(entryNotFoundMsg)
	}

	return entry, nil
}

// exceedsCapacity reports whether raising a Card from current to quantity
// leaves the Collection's summed quantity (sum, before the change) above its
// limit (0 = no limit). A decrease never exceeds, even when already over.
func exceedsCapacity(c entity.Collection, sum, current, quantity int) bool {
	return c.MaxCardCount > 0 && quantity > current && sum-current+quantity > c.MaxCardCount
}

// checkCapacity rejects a write that raises a Card's quantity and would leave
// the Collection's summed quantity above its limit (0 = no limit). current is
// the quantity the Card already holds, which the write replaces. A decrease is
// always allowed, even when the Collection is already over a lowered limit.
// The caller holds the Collection's row lock, so the sum can't change under it.
func (s *inventoryService) checkCapacity(ctx context.Context, c entity.Collection, current, quantity int) error {
	if c.MaxCardCount == 0 || quantity <= current {
		return nil
	}

	sum, err := s.entries.SumQuantity(ctx, c.ID)
	if err != nil {
		return err
	}
	if exceedsCapacity(c, sum, current, quantity) {
		return ungerr.UnprocessableEntityError(capacityExceededMsg)
	}

	return nil
}

func (s *inventoryService) Add(ctx context.Context, req dto.InventoryEntryRequest) (dto.InventoryEntry, error) {
	if req.Quantity <= 0 {
		return dto.InventoryEntry{}, ungerr.BadRequestError(quantityNotPositiveMsg)
	}

	var written entity.InventoryEntry
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}
		if err := s.refuseBinder(ctx, c.ID); err != nil {
			return err
		}

		exists, err := s.cards.CardExists(ctx, req.CardID)
		if err != nil {
			return err
		}
		if !exists {
			return ungerr.NotFoundError(cardNotFoundMsg)
		}

		// The Collection's row lock (not a unique-violation mapping) guarantees
		// no concurrent insert slips past this check.
		existing, err := s.entries.FindFirst(ctx, crud.Specification[entity.InventoryEntry]{
			Model:     entity.InventoryEntry{CollectionID: c.ID, CardID: req.CardID},
			ForUpdate: true,
		})
		if err != nil {
			return err
		}
		if !existing.IsZero() {
			return ungerr.ConflictError(entryExistsMsg)
		}

		if err := s.checkCapacity(ctx, c, 0, req.Quantity); err != nil {
			return err
		}

		written, err = s.entries.Insert(ctx, entity.InventoryEntry{CollectionID: c.ID, CardID: req.CardID, Quantity: req.Quantity})
		return err
	})
	if err != nil {
		return dto.InventoryEntry{}, err
	}

	return mapper.ToInventoryEntry(written), nil
}

func (s *inventoryService) UpdateQuantity(ctx context.Context, req dto.InventoryEntryRequest) (dto.InventoryEntry, error) {
	if req.Quantity <= 0 {
		return dto.InventoryEntry{}, ungerr.BadRequestError(quantityNotPositiveMsg)
	}

	var written entity.InventoryEntry
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}
		if err := s.refuseBinder(ctx, c.ID); err != nil {
			return err
		}

		entry, err := s.getEntry(ctx, c.ID, req.CardID)
		if err != nil {
			return err
		}

		if err := s.checkCapacity(ctx, c, entry.Quantity, req.Quantity); err != nil {
			return err
		}

		entry.Quantity = req.Quantity
		written, err = s.entries.Update(ctx, entry)
		return err
	})
	if err != nil {
		return dto.InventoryEntry{}, err
	}

	return mapper.ToInventoryEntry(written), nil
}

func (s *inventoryService) Remove(ctx context.Context, req dto.InventoryEntryLookup) error {
	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}
		if err := s.refuseBinder(ctx, c.ID); err != nil {
			return err
		}

		entry, err := s.getEntry(ctx, c.ID, req.CardID)
		if err != nil {
			return err
		}

		return s.entries.Delete(ctx, entry)
	})
}

func (s *inventoryService) Move(ctx context.Context, req dto.InventoryMoveRequest) (dto.InventoryMoveResult, error) {
	if req.SourceID == req.DestinationID {
		return dto.InventoryMoveResult{}, ungerr.BadRequestError(moveSameCollectionMsg)
	}
	if err := rejectDuplicateCards(req.CardIDs); err != nil {
		return dto.InventoryMoveResult{}, err
	}

	var result dto.InventoryMoveResult
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		// Both rows are locked in one order, so two Moves in opposite
		// directions cannot deadlock on each other.
		lockOrder := []uuid.UUID{req.SourceID, req.DestinationID}
		if bytes.Compare(req.DestinationID[:], req.SourceID[:]) < 0 {
			lockOrder[0], lockOrder[1] = lockOrder[1], lockOrder[0]
		}
		owned := make(map[uuid.UUID]entity.Collection, len(lockOrder))
		for _, id := range lockOrder {
			c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, id, true)
			if err != nil {
				return err
			}
			owned[id] = c
		}
		src, dst := owned[req.SourceID], owned[req.DestinationID]

		srcBinder, err := s.layouts.IsBinder(ctx, src.ID)
		if err != nil {
			return err
		}
		isBinder, err := s.layouts.IsBinder(ctx, dst.ID)
		if err != nil {
			return err
		}
		if isBinder {
			return apperr.WithCode(ungerr.ConflictError(moveToBinderMsg), apperr.CodeMoveToBinder)
		}

		moving := make([]entity.InventoryEntry, 0, len(req.CardIDs))
		copies := 0
		for _, cardID := range req.CardIDs {
			entry, err := s.getEntry(ctx, src.ID, cardID)
			if err != nil {
				return err
			}
			moving = append(moving, entry)
			copies += entry.Quantity
		}

		// Every moved copy lands in the destination, merged or not, so the
		// destination's count rises by the whole total.
		if dst.MaxCardCount > 0 {
			sum, err := s.entries.SumQuantity(ctx, dst.ID)
			if err != nil {
				return err
			}
			if over := sum + copies - dst.MaxCardCount; over > 0 {
				return ungerr.UnprocessableEntityError(fmt.Sprintf(moveCapacityMsg, countOf(dst.MaxCardCount, "card", "cards"), countOf(over, "copy", "copies")))
			}
		}

		if srcBinder {
			// ADR-0020: a binder's entries follow its Slots, so ClearCards empties
			// the Slots holding these Cards and reconciles each entry away.
			if err := s.binderSlots.ClearCards(ctx, dto.BinderCardsClear{ProfileID: req.ProfileID, CollectionID: src.ID, CardIDs: req.CardIDs}); err != nil {
				return err
			}
		}

		for _, entry := range moving {
			if !srcBinder {
				if err := s.entries.Delete(ctx, entry); err != nil {
					return err
				}
			}
			existing, err := s.findEntry(ctx, dst.ID, entry.CardID)
			if err != nil {
				return err
			}
			if existing.IsZero() {
				_, err = s.entries.Insert(ctx, entity.InventoryEntry{CollectionID: dst.ID, CardID: entry.CardID, Quantity: entry.Quantity})
			} else {
				existing.Quantity += entry.Quantity
				_, err = s.entries.Update(ctx, existing)
			}
			if err != nil {
				return err
			}
		}

		result = dto.InventoryMoveResult{Cards: len(moving), Copies: copies}
		return nil
	})
	if err != nil {
		return dto.InventoryMoveResult{}, err
	}

	return result, nil
}

// countOf renders n with the singular or plural noun, so "1 copy" and "2 copies" read right.
func countOf(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}

	return fmt.Sprintf("%d %s", n, many)
}

// rejectDuplicateCards refuses a request that names a Card twice, since the
// second item would silently override the first.
func rejectDuplicateCards(cardIDs []uuid.UUID) error {
	seen := make(map[uuid.UUID]struct{}, len(cardIDs))
	for _, id := range cardIDs {
		if _, dup := seen[id]; dup {
			return ungerr.BadRequestError(duplicateCardMsg)
		}
		seen[id] = struct{}{}
	}

	return nil
}

func (s *inventoryService) BulkUpdate(ctx context.Context, req dto.InventoryBulkUpdateRequest) ([]dto.InventoryChangeResult, error) {
	cardIDs := ezutil.MapSlice(req.Items, func(item dto.InventoryQuantityChange) uuid.UUID { return item.CardID })
	if err := rejectDuplicateCards(cardIDs); err != nil {
		return nil, err
	}

	var results []dto.InventoryChangeResult
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		results = make([]dto.InventoryChangeResult, 0, len(req.Items))
		c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}
		if err := s.refuseBinder(ctx, c.ID); err != nil {
			return err
		}

		// The Collection's row lock keeps this sum valid for the whole batch;
		// track it locally instead of re-summing per item.
		sum := 0
		if c.MaxCardCount > 0 {
			if sum, err = s.entries.SumQuantity(ctx, c.ID); err != nil {
				return err
			}
		}

		for _, item := range req.Items {
			res, delta, err := s.applyChange(ctx, c, sum, item)
			if err != nil {
				return err
			}
			sum += delta
			results = append(results, res)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

// applyChange applies one bulk item and returns its result and the change in
// the Collection's summed quantity. sum is only meaningful when the
// Collection has a limit.
func (s *inventoryService) applyChange(ctx context.Context, c entity.Collection, sum int, item dto.InventoryQuantityChange) (dto.InventoryChangeResult, int, error) {
	declined := func(reason, msg string) dto.InventoryChangeResult {
		return dto.InventoryChangeResult{CardID: item.CardID, Status: dto.InventoryStatusDeclined, Reason: reason, Message: msg}
	}

	existing, err := s.findEntry(ctx, c.ID, item.CardID)
	if err != nil {
		return dto.InventoryChangeResult{}, 0, err
	}

	if existing.IsZero() {
		exists, err := s.cards.CardExists(ctx, item.CardID)
		if err != nil {
			return dto.InventoryChangeResult{}, 0, err
		}
		if !exists {
			return declined(dto.InventoryReasonCardNotFound, cardNotFoundMsg), 0, nil
		}
		if item.Quantity == 0 {
			return dto.InventoryChangeResult{CardID: item.CardID, Status: dto.InventoryStatusRemoved}, 0, nil
		}
	}

	current := existing.Quantity
	if exceedsCapacity(c, sum, current, item.Quantity) {
		res := declined(dto.InventoryReasonCapacityExceeded, capacityExceededMsg)
		res.Quantity = current
		return res, 0, nil
	}

	switch {
	case item.Quantity == 0:
		err = s.entries.Delete(ctx, existing)
	case existing.IsZero():
		_, err = s.entries.Insert(ctx, entity.InventoryEntry{CollectionID: c.ID, CardID: item.CardID, Quantity: item.Quantity})
	case item.Quantity != current:
		existing.Quantity = item.Quantity
		_, err = s.entries.Update(ctx, existing)
	}
	if err != nil {
		return dto.InventoryChangeResult{}, 0, err
	}

	status := dto.InventoryStatusApplied
	if item.Quantity == 0 {
		status = dto.InventoryStatusRemoved
	}
	return dto.InventoryChangeResult{CardID: item.CardID, Quantity: item.Quantity, Status: status}, item.Quantity - current, nil
}
