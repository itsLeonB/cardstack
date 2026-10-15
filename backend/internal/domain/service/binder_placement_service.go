package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

const (
	placementNotBinderMsg      = "this Collection is not a binder"
	placementNotEnoughEmptyMsg = "not enough empty Slots"
)

// BinderPlacementService places Cards into a binder's empty Slots, the way the
// Phase 1 "Scan cards" flow adds copies to a binder. It locks the Collection
// for the whole batch and hands every copy to BinderService.WriteSlots, so
// every Inventory Entry is reconciled with its Slots in the same transaction.
type BinderPlacementService interface {
	// PlaceCards fills the first empty Slots in Page order, one Slot per copy. It
	// declines, rather than fails, the copies that find no empty Slot or whose
	// Card is unknown.
	PlaceCards(ctx context.Context, req dto.BinderPlacementRequest) ([]dto.BinderPlacementResult, error)
}

type binderPlacementService struct {
	transactor  crud.Transactor
	collections repository.CollectionRepository
	layouts     repository.BinderLayoutRepository
	leaves      repository.BinderLeafRepository
	slots       repository.BinderSlotRepository
	cards       crud.Repository[entity.Card]
	binder      BinderService
}

func NewBinderPlacementService(
	transactor crud.Transactor,
	collections repository.CollectionRepository,
	layouts repository.BinderLayoutRepository,
	leaves repository.BinderLeafRepository,
	slots repository.BinderSlotRepository,
	cards crud.Repository[entity.Card],
	binder BinderService,
) BinderPlacementService {
	return &binderPlacementService{
		transactor:  transactor,
		collections: collections,
		layouts:     layouts,
		leaves:      leaves,
		slots:       slots,
		cards:       cards,
		binder:      binder,
	}
}

func (s *binderPlacementService) PlaceCards(ctx context.Context, req dto.BinderPlacementRequest) ([]dto.BinderPlacementResult, error) {
	seen := make(map[uuid.UUID]struct{}, len(req.Items))
	for _, item := range req.Items {
		if item.Quantity < 1 {
			return nil, ungerr.BadRequestError(quantityNotPositiveMsg)
		}
		if _, dup := seen[item.CardID]; dup {
			return nil, ungerr.BadRequestError(duplicateCardMsg)
		}
		seen[item.CardID] = struct{}{}
	}

	var results []dto.BinderPlacementResult
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		results = make([]dto.BinderPlacementResult, 0, len(req.Items))
		c, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.CollectionID, true)
		if err != nil {
			return err
		}

		isBinder, err := s.layouts.IsBinder(ctx, c.ID)
		if err != nil {
			return err
		}
		if !isBinder {
			return ungerr.ConflictError(placementNotBinderMsg)
		}

		empty, err := s.emptySlots(ctx, c.ID)
		if err != nil {
			return err
		}

		var changes []dto.BinderSlotChange
		for _, item := range req.Items {
			card, err := s.findCard(ctx, item.CardID)
			if err != nil {
				return err
			}
			if card.IsZero() {
				results = append(results, dto.BinderPlacementResult{
					CardID:   item.CardID,
					Declined: item.Quantity,
					Reason:   dto.InventoryReasonCardNotFound,
					Message:  cardNotFoundMsg,
				})
				continue
			}

			placed := min(item.Quantity, len(empty))
			for _, target := range empty[:placed] {
				changes = append(changes, dto.BinderSlotChange{Target: target, State: dto.BinderSlotCard, CardID: item.CardID})
			}
			empty = empty[placed:]

			result := dto.BinderPlacementResult{CardID: item.CardID, Placed: placed, Declined: item.Quantity - placed}
			if result.Declined > 0 {
				result.Reason = dto.BinderPlacementReasonNotEnoughEmptySlots
				result.Message = placementNotEnoughEmptyMsg
			}
			results = append(results, result)
		}

		if len(changes) == 0 {
			return nil
		}

		return s.binder.WriteSlots(ctx, dto.BinderSlotsWrite{
			ProfileID:    req.ProfileID,
			CollectionID: c.ID,
			Changes:      changes,
		})
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

// emptySlots lists the Slots of a binder holding no card, in Page order and
// then row by row. It is read once under the Collection's lock, and the caller
// consumes it as it places copies.
func (s *binderPlacementService) emptySlots(ctx context.Context, collectionID uuid.UUID) ([]dto.BinderSlotTarget, error) {
	layout, err := s.layouts.GetLayout(ctx, collectionID)
	if err != nil {
		return nil, err
	}

	leafCount, err := s.leaves.CountLeaves(ctx, collectionID)
	if err != nil {
		return nil, err
	}

	occupied, err := s.slots.ListAllOccupiedSlots(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	taken := make(map[dto.BinderSlotTarget]bool, len(occupied))
	for _, slot := range occupied {
		taken[dto.BinderSlotTarget{Page: mapper.PageOf(slot.Leaf.Position, slot.Side), Row: slot.Row, Column: slot.Column}] = true
	}

	var empty []dto.BinderSlotTarget
	for page := 1; page <= 2*leafCount; page++ {
		for row := 1; row <= layout.Rows; row++ {
			for column := 1; column <= layout.Columns; column++ {
				target := dto.BinderSlotTarget{Page: page, Row: row, Column: column}
				if !taken[target] {
					empty = append(empty, target)
				}
			}
		}
	}

	return empty, nil
}

// findCard returns the Card, or the zero value when absent. A nil id is refused
// before the query, since an empty Model would match any Card.
func (s *binderPlacementService) findCard(ctx context.Context, cardID uuid.UUID) (entity.Card, error) {
	if cardID == uuid.Nil {
		return entity.Card{}, nil
	}

	return s.cards.FindFirst(ctx, crud.Specification[entity.Card]{
		Model: entity.Card{BaseEntity: crud.BaseEntity{ID: cardID}},
	})
}
