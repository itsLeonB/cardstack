package service

import (
	"cmp"
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ezutil/v2"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

const (
	cardNotFoundMsg        = "card not found"
	entryNotFoundMsg       = "card is not in this collection"
	entryExistsMsg         = "card is already in this collection"
	capacityExceededMsg    = "collection capacity limit exceeded"
	quantityNotPositiveMsg = "quantity must be positive"
	duplicateCardMsg       = "duplicate cardId in request"
)

// InventoryService scopes every method to the request's ProfileID; another
// profile's Collection is reported as "collection not found", same as a
// missing one.
type InventoryService interface {
	List(ctx context.Context, req dto.InventoryListRequest) ([]dto.InventoryItem, error)
	Add(ctx context.Context, req dto.InventoryEntryRequest) (dto.InventoryEntry, error)
	UpdateQuantity(ctx context.Context, req dto.InventoryEntryRequest) (dto.InventoryEntry, error)
	Remove(ctx context.Context, req dto.InventoryEntryLookup) error
	// BulkUpdate applies the items in order in one transaction, declining (not
	// failing) an item that is over capacity or names an unknown Card.
	BulkUpdate(ctx context.Context, req dto.InventoryBulkUpdateRequest) ([]dto.InventoryChangeResult, error)
}

type inventoryService struct {
	transactor  crud.Transactor
	collections repository.CollectionRepository
	entries     repository.InventoryRepository
	cards       crud.Repository[entity.Card]
}

func NewInventoryService(
	transactor crud.Transactor,
	collections repository.CollectionRepository,
	entries repository.InventoryRepository,
	cards crud.Repository[entity.Card],
) InventoryService {
	return &inventoryService{transactor: transactor, collections: collections, entries: entries, cards: cards}
}

func (s *inventoryService) List(ctx context.Context, req dto.InventoryListRequest) ([]dto.InventoryItem, error) {
	if _, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, false); err != nil {
		return nil, err
	}

	entries, err := s.entries.FindAll(ctx, crud.Specification[entity.InventoryEntry]{
		Model:            entity.InventoryEntry{CollectionID: req.CollectionID},
		PreloadRelations: []string{"Card.Rarity", "Card.ExpansionSet"},
	})
	if err != nil {
		return nil, err
	}

	// crud.Repository always orders by created_at, so sort here: release date
	// (unknown last), then set, local id, name, id. Like catalog search, but
	// strings compare bytewise here, not by DB collation.
	slices.SortStableFunc(entries, compareEntries)

	return ezutil.MapSlice(entries, mapper.ToInventoryItem), nil
}

func compareEntries(a, b entity.InventoryEntry) int {
	ar, br := a.Card.ExpansionSet.ReleaseDate, b.Card.ExpansionSet.ReleaseDate
	if (ar == nil) != (br == nil) {
		if ar == nil {
			return 1
		}
		return -1
	}
	if ar != nil {
		if c := ar.Compare(*br); c != 0 {
			return c
		}
	}
	return cmp.Or(
		cmp.Compare(a.Card.ExpansionSet.ID.String(), b.Card.ExpansionSet.ID.String()),
		cmp.Compare(a.Card.LocalID, b.Card.LocalID),
		cmp.Compare(a.Card.Name, b.Card.Name),
		cmp.Compare(a.Card.ID.String(), b.Card.ID.String()),
	)
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

// findCard returns the Card, or the zero value when absent. Guards a nil id
// for the same reason as findEntry.
func (s *inventoryService) findCard(ctx context.Context, cardID uuid.UUID) (entity.Card, error) {
	if cardID == uuid.Nil {
		return entity.Card{}, nil
	}

	return s.cards.FindFirst(ctx, crud.Specification[entity.Card]{
		Model: entity.Card{BaseEntity: crud.BaseEntity{ID: cardID}},
	})
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

		card, err := s.findCard(ctx, req.CardID)
		if err != nil {
			return err
		}
		if card.IsZero() {
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

		entry, err := s.getEntry(ctx, c.ID, req.CardID)
		if err != nil {
			return err
		}

		return s.entries.Delete(ctx, entry)
	})
}

func (s *inventoryService) BulkUpdate(ctx context.Context, req dto.InventoryBulkUpdateRequest) ([]dto.InventoryChangeResult, error) {
	seen := make(map[uuid.UUID]struct{}, len(req.Items))
	for _, item := range req.Items {
		if _, dup := seen[item.CardID]; dup {
			return nil, ungerr.BadRequestError(duplicateCardMsg)
		}
		seen[item.CardID] = struct{}{}
	}

	var results []dto.InventoryChangeResult
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		results = make([]dto.InventoryChangeResult, 0, len(req.Items))
		c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, true)
		if err != nil {
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
		card, err := s.findCard(ctx, item.CardID)
		if err != nil {
			return dto.InventoryChangeResult{}, 0, err
		}
		if card.IsZero() {
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
