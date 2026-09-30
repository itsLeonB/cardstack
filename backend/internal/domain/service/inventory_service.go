package service

import (
	"context"

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

// checkCapacity rejects a write that would leave the Collection's summed
// quantity above its limit (0 = no limit). current is the quantity the Card
// already holds, which the write replaces. The sum-then-write isn't atomic,
// so two concurrent writes can overshoot the limit by one write's worth.
// ponytail: row-lock the collection in a transaction if that ever matters.
func (s *inventoryService) checkCapacity(ctx context.Context, c entity.Collection, current, quantity int) error {
	if c.MaxCardCount == 0 {
		return nil
	}

	sum, err := s.inventory.SumQuantity(ctx, c.ID)
	if err != nil {
		return err
	}
	if sum-current+quantity > c.MaxCardCount {
		return ungerr.UnprocessableEntityError(capacityExceededMsg)
	}

	return nil
}

func (s *inventoryService) Add(ctx context.Context, profileID, collectionID uuid.UUID, req dto.InventoryEntryRequest) (dto.InventoryEntry, error) {
	c, err := s.findOwned(ctx, profileID, collectionID)
	if err != nil {
		return dto.InventoryEntry{}, err
	}
	if req.Quantity <= 0 {
		return dto.InventoryEntry{}, ungerr.BadRequestError(quantityNotPositiveMsg)
	}

	exists, err := s.inventory.CardExists(ctx, req.CardID)
	if err != nil {
		return dto.InventoryEntry{}, err
	}
	if !exists {
		return dto.InventoryEntry{}, ungerr.NotFoundError(cardNotFoundMsg)
	}

	existing, err := s.inventory.FindEntry(ctx, collectionID, req.CardID)
	if err != nil {
		return dto.InventoryEntry{}, err
	}
	if !existing.IsZero() {
		return dto.InventoryEntry{}, ungerr.ConflictError(entryExistsMsg)
	}

	if err := s.checkCapacity(ctx, c, 0, req.Quantity); err != nil {
		return dto.InventoryEntry{}, err
	}

	saved, err := s.inventory.SaveEntry(ctx, entity.InventoryEntry{CollectionID: collectionID, CardID: req.CardID, Quantity: req.Quantity})
	if err != nil {
		return dto.InventoryEntry{}, err
	}

	return mapper.ToInventoryEntry(saved), nil
}

func (s *inventoryService) UpdateQuantity(ctx context.Context, profileID, collectionID, cardID uuid.UUID, quantity int) (dto.InventoryEntry, error) {
	c, err := s.findOwned(ctx, profileID, collectionID)
	if err != nil {
		return dto.InventoryEntry{}, err
	}
	if quantity <= 0 {
		return dto.InventoryEntry{}, ungerr.BadRequestError(quantityNotPositiveMsg)
	}

	existing, err := s.inventory.FindEntry(ctx, collectionID, cardID)
	if err != nil {
		return dto.InventoryEntry{}, err
	}
	if existing.IsZero() {
		return dto.InventoryEntry{}, ungerr.NotFoundError(entryNotFoundMsg)
	}

	if err := s.checkCapacity(ctx, c, existing.Quantity, quantity); err != nil {
		return dto.InventoryEntry{}, err
	}

	existing.Quantity = quantity
	saved, err := s.inventory.SaveEntry(ctx, existing)
	if err != nil {
		return dto.InventoryEntry{}, err
	}

	return mapper.ToInventoryEntry(saved), nil
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
