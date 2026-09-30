package service

import (
	"context"
	"errors"

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

// InventoryService scopes every method to the calling profileID; another
// profile's Collection is reported as "collection not found", same as a
// missing one.
type InventoryService interface {
	List(ctx context.Context, profileID, collectionID uuid.UUID) ([]dto.InventoryItem, error)
	Add(ctx context.Context, profileID, collectionID uuid.UUID, req dto.InventoryEntryRequest) (dto.InventoryEntry, error)
	UpdateQuantity(ctx context.Context, profileID, collectionID, cardID uuid.UUID, quantity int) (dto.InventoryEntry, error)
	Remove(ctx context.Context, profileID, collectionID, cardID uuid.UUID) error
}

type inventoryService struct {
	collections crud.Repository[entity.Collection]
	inventory   repository.InventoryRepository
}

func NewInventoryService(collections crud.Repository[entity.Collection], inventory repository.InventoryRepository) InventoryService {
	return &inventoryService{collections: collections, inventory: inventory}
}

// findOwned mirrors collectionService.findOwned (owner filter in the query;
// a nil id would drop the ID condition).
func (s *inventoryService) findOwned(ctx context.Context, profileID, id uuid.UUID) (entity.Collection, error) {
	if id == uuid.Nil {
		return entity.Collection{}, ungerr.NotFoundError(collectionNotFoundMsg)
	}

	c, err := s.collections.FindFirst(ctx, crud.Specification[entity.Collection]{
		Model: entity.Collection{BaseEntity: crud.BaseEntity{ID: id}, ProfileID: profileID},
	})
	if err != nil {
		return entity.Collection{}, err
	}
	if c.IsZero() {
		return entity.Collection{}, ungerr.NotFoundError(collectionNotFoundMsg)
	}

	return c, nil
}

func (s *inventoryService) List(ctx context.Context, profileID, collectionID uuid.UUID) ([]dto.InventoryItem, error) {
	if _, err := s.findOwned(ctx, profileID, collectionID); err != nil {
		return nil, err
	}

	items, err := s.inventory.ListItems(ctx, collectionID)
	if err != nil {
		return nil, err
	}

	return ezutil.MapSlice(items, mapper.ToInventoryItem), nil
}

// checkCapacity rejects a write that raises a Card's quantity and would leave
// the Collection's summed quantity above its limit (0 = no limit). current is
// the quantity the Card already holds, which the write replaces. A decrease is
// always allowed, even when the Collection is already over a lowered limit.
// The caller holds the Collection's row lock, so the sum can't change under it.
func checkCapacity(ctx context.Context, tx repository.InventoryRepository, c entity.Collection, current, quantity int) error {
	if c.MaxCardCount == 0 || quantity <= current {
		return nil
	}

	sum, err := tx.SumQuantity(ctx, c.ID)
	if err != nil {
		return err
	}
	if sum-current+quantity > c.MaxCardCount {
		return ungerr.UnprocessableEntityError(capacityExceededMsg)
	}

	return nil
}

// withLockedCollection maps the repository's not-found sentinel to the same
// 404 findOwned returns.
func (s *inventoryService) withLockedCollection(ctx context.Context, profileID, collectionID uuid.UUID, fn func(tx repository.InventoryRepository, c entity.Collection) error) error {
	if collectionID == uuid.Nil {
		return ungerr.NotFoundError(collectionNotFoundMsg)
	}

	err := s.inventory.WithLockedCollection(ctx, profileID, collectionID, fn)
	if errors.Is(err, repository.ErrCollectionNotFound) {
		return ungerr.NotFoundError(collectionNotFoundMsg)
	}

	return err
}

func (s *inventoryService) Add(ctx context.Context, profileID, collectionID uuid.UUID, req dto.InventoryEntryRequest) (dto.InventoryEntry, error) {
	if req.Quantity <= 0 {
		return dto.InventoryEntry{}, ungerr.BadRequestError(quantityNotPositiveMsg)
	}

	err := s.withLockedCollection(ctx, profileID, collectionID, func(tx repository.InventoryRepository, c entity.Collection) error {
		exists, err := tx.CardExists(ctx, req.CardID)
		if err != nil {
			return err
		}
		if !exists {
			return ungerr.NotFoundError(cardNotFoundMsg)
		}

		if err := checkCapacity(ctx, tx, c, 0, req.Quantity); err != nil {
			return err
		}

		err = tx.InsertEntry(ctx, entity.InventoryEntry{CollectionID: collectionID, CardID: req.CardID, Quantity: req.Quantity})
		if errors.Is(err, repository.ErrEntryExists) {
			return ungerr.ConflictError(entryExistsMsg)
		}

		return err
	})
	if err != nil {
		return dto.InventoryEntry{}, err
	}

	return dto.InventoryEntry{CardID: req.CardID, Quantity: req.Quantity}, nil
}

func (s *inventoryService) UpdateQuantity(ctx context.Context, profileID, collectionID, cardID uuid.UUID, quantity int) (dto.InventoryEntry, error) {
	if quantity <= 0 {
		return dto.InventoryEntry{}, ungerr.BadRequestError(quantityNotPositiveMsg)
	}

	err := s.withLockedCollection(ctx, profileID, collectionID, func(tx repository.InventoryRepository, c entity.Collection) error {
		existing, err := tx.FindEntry(ctx, collectionID, cardID)
		if err != nil {
			return err
		}
		if existing.IsZero() {
			return ungerr.NotFoundError(entryNotFoundMsg)
		}

		if err := checkCapacity(ctx, tx, c, existing.Quantity, quantity); err != nil {
			return err
		}

		err = tx.UpdateQuantity(ctx, collectionID, cardID, quantity)
		if errors.Is(err, repository.ErrEntryNotFound) {
			return ungerr.NotFoundError(entryNotFoundMsg)
		}

		return err
	})
	if err != nil {
		return dto.InventoryEntry{}, err
	}

	return dto.InventoryEntry{CardID: cardID, Quantity: quantity}, nil
}

func (s *inventoryService) Remove(ctx context.Context, profileID, collectionID, cardID uuid.UUID) error {
	if _, err := s.findOwned(ctx, profileID, collectionID); err != nil {
		return err
	}

	existing, err := s.inventory.FindEntry(ctx, collectionID, cardID)
	if err != nil {
		return err
	}
	if existing.IsZero() {
		return ungerr.NotFoundError(entryNotFoundMsg)
	}

	return s.inventory.DeleteEntry(ctx, collectionID, cardID)
}
