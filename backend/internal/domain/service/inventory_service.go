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
)

// InventoryService scopes every method to the request's ProfileID; another
// profile's Collection is reported as "collection not found", same as a
// missing one.
type InventoryService interface {
	List(ctx context.Context, req dto.InventoryListRequest) ([]dto.InventoryItem, error)
	Add(ctx context.Context, req dto.InventoryEntryRequest) (dto.InventoryEntry, error)
	UpdateQuantity(ctx context.Context, req dto.InventoryEntryRequest) (dto.InventoryEntry, error)
	Remove(ctx context.Context, req dto.InventoryEntryLookup) error
}

type inventoryService struct {
	transactor  crud.Transactor
	collections crud.Repository[entity.Collection]
	entries     repository.InventoryRepository
	cards       crud.Repository[entity.Card]
}

func NewInventoryService(
	transactor crud.Transactor,
	collections crud.Repository[entity.Collection],
	entries repository.InventoryRepository,
	cards crud.Repository[entity.Card],
) InventoryService {
	return &inventoryService{transactor: transactor, collections: collections, entries: entries, cards: cards}
}

func (s *inventoryService) List(ctx context.Context, req dto.InventoryListRequest) ([]dto.InventoryItem, error) {
	if _, err := findOwnedCollection(ctx, s.collections, req.ProfileID, req.CollectionID, false); err != nil {
		return nil, err
	}

	entries, err := s.entries.FindAll(ctx, crud.Specification[entity.InventoryEntry]{
		Model:            entity.InventoryEntry{CollectionID: req.CollectionID},
		PreloadRelations: []string{"Card.Rarity", "Card.ExpansionSet"},
	})
	if err != nil {
		return nil, err
	}

	// crud.Repository always orders by created_at; this is the catalog
	// search order (release date, unknown last).
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

// entrySpec addresses one Card's entry, row-locked. A nil card id would
// drop that condition (see crud.WhereBySpec), so callers reject it first.
func entrySpec(collectionID, cardID uuid.UUID) crud.Specification[entity.InventoryEntry] {
	return crud.Specification[entity.InventoryEntry]{
		Model:     entity.InventoryEntry{CollectionID: collectionID, CardID: cardID},
		ForUpdate: true,
	}
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
	if sum-current+quantity > c.MaxCardCount {
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
		c, err := findOwnedCollection(ctx, s.collections, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}

		if req.CardID == uuid.Nil {
			return ungerr.NotFoundError(cardNotFoundMsg)
		}
		card, err := s.cards.FindFirst(ctx, crud.Specification[entity.Card]{
			Model: entity.Card{BaseEntity: crud.BaseEntity{ID: req.CardID}},
		})
		if err != nil {
			return err
		}
		if card.IsZero() {
			return ungerr.NotFoundError(cardNotFoundMsg)
		}

		existing, err := s.entries.FindFirst(ctx, entrySpec(c.ID, req.CardID))
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
		c, err := findOwnedCollection(ctx, s.collections, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}

		if req.CardID == uuid.Nil {
			return ungerr.NotFoundError(entryNotFoundMsg)
		}
		entry, err := s.entries.FindFirst(ctx, entrySpec(c.ID, req.CardID))
		if err != nil {
			return err
		}
		if entry.IsZero() {
			return ungerr.NotFoundError(entryNotFoundMsg)
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
		c, err := findOwnedCollection(ctx, s.collections, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}

		if req.CardID == uuid.Nil {
			return ungerr.NotFoundError(entryNotFoundMsg)
		}
		entry, err := s.entries.FindFirst(ctx, entrySpec(c.ID, req.CardID))
		if err != nil {
			return err
		}
		if entry.IsZero() {
			return ungerr.NotFoundError(entryNotFoundMsg)
		}

		return s.entries.Delete(ctx, entry)
	})
}
