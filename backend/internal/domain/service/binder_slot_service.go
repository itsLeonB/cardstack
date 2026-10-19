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
	slotOutOfGridMsg           = "row and column must lie within the Layout's rows and columns"
	emptySourceMsg             = "the source Slot is empty; only a Slot holding a card can be moved"
	binderPageSlotCountMsg     = "the request needs one state per Slot of the Page, in grid order"
	placementNotEnoughEmptyMsg = "not enough empty Slots"
)

// BinderSlotService writes the Slots of a binder: one Slot, a move or swap, a
// whole Page, and the placement of scanned Cards. Each write row-locks the
// Collection inside a transaction and reconciles the Inventory Entries of the
// Cards it touched in that same transaction, so an entry always equals its
// Card's Slot count (ADR-0020).
type BinderSlotService interface {
	// WriteSlot sets one Slot. Giving a Card to an occupied Slot replaces its
	// content.
	WriteSlot(ctx context.Context, req dto.BinderSlotWrite) error
	// MoveSlot moves a card to another Slot of the same binder, swapping when the
	// target is filled. Quantities do not change.
	MoveSlot(ctx context.Context, req dto.BinderSlotMove) error
	// ReplacePage sets every Slot of one Page to the state requested for it. A
	// Card the server does not know is stored Unidentified and reported as such.
	ReplacePage(ctx context.Context, req dto.BinderPageReplace) (dto.BinderPageReplaceResult, error)
	// PlaceCards fills the first empty Slots in Page order, one Slot per copy. It
	// declines, rather than fails, the copies that find no empty Slot or whose
	// Card is unknown.
	PlaceCards(ctx context.Context, req dto.BinderPlacementRequest) ([]dto.BinderPlacementResult, error)
}

type binderSlotService struct {
	transactor  crud.Transactor
	collections repository.CollectionRepository
	layouts     repository.BinderLayoutRepository
	leaves      repository.BinderLeafRepository
	slots       repository.BinderSlotRepository
	entries     crud.Repository[entity.InventoryEntry]
	cards       CardLookup
}

func NewBinderSlotService(
	transactor crud.Transactor,
	collections repository.CollectionRepository,
	layouts repository.BinderLayoutRepository,
	leaves repository.BinderLeafRepository,
	slots repository.BinderSlotRepository,
	entries crud.Repository[entity.InventoryEntry],
	cards CardLookup,
) BinderSlotService {
	return &binderSlotService{
		transactor:  transactor,
		collections: collections,
		layouts:     layouts,
		leaves:      leaves,
		slots:       slots,
		entries:     entries,
		cards:       cards,
	}
}

func (s *binderSlotService) WriteSlot(ctx context.Context, req dto.BinderSlotWrite) error {
	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, layout, err := s.lockBinder(ctx, req.ProfileID, req.CollectionID)
		if err != nil {
			return err
		}

		if err := checkSlotInGrid(layout, req.Target); err != nil {
			return err
		}
		if req.State == dto.BinderSlotCard {
			if err := s.cards.RequireCard(ctx, req.CardID); err != nil {
				return err
			}
		}

		return s.writeSlots(ctx, c, layout, []dto.BinderSlotChange{{Target: req.Target, State: req.State, CardID: req.CardID}})
	})
}

func (s *binderSlotService) MoveSlot(ctx context.Context, req dto.BinderSlotMove) error {
	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, layout, err := s.lockBinder(ctx, req.ProfileID, req.CollectionID)
		if err != nil {
			return err
		}

		from, filled, err := s.slotAt(ctx, c, layout, req.From)
		if err != nil {
			return err
		}
		if !filled {
			return ungerr.BadRequestError(emptySourceMsg)
		}

		to, toFilled, err := s.slotAt(ctx, c, layout, req.To)
		if err != nil {
			return err
		}
		if req.From == req.To {
			return nil
		}

		if !toFilled {
			// The row keeps its card and takes the empty Slot's position. The
			// unique key on the position cannot clash, since the target is empty.
			from.LeafID, from.Side, from.Row, from.Column = to.LeafID, to.Side, to.Row, to.Column
			_, err = s.slots.Update(ctx, from)
			return err
		}

		// Both Slots are filled, so the two rows exchange what they hold and keep
		// their positions. Card and Unidentified swap together, so each row still
		// holds exactly one of them.
		from.CardID, to.CardID = to.CardID, from.CardID
		from.Unidentified, to.Unidentified = to.Unidentified, from.Unidentified
		if _, err := s.slots.Update(ctx, from); err != nil {
			return err
		}
		_, err = s.slots.Update(ctx, to)
		return err
	})
}

func (s *binderSlotService) ReplacePage(ctx context.Context, req dto.BinderPageReplace) (dto.BinderPageReplaceResult, error) {
	var result dto.BinderPageReplaceResult
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, layout, err := s.lockBinder(ctx, req.ProfileID, req.CollectionID)
		if err != nil {
			return err
		}
		if len(req.Slots) != layout.SlotsPerPage() {
			return ungerr.BadRequestError(binderPageSlotCountMsg)
		}

		// The Page's Slots are written as one batch. A missing Page is refused by
		// the leaf lookup in writeSlots, inside this transaction, with nothing written.
		known := make(map[uuid.UUID]bool)
		changes := make([]dto.BinderSlotChange, 0, len(req.Slots))
		slots := make([]dto.BinderSlotResult, 0, len(req.Slots))
		for i, want := range req.Slots {
			target := dto.BinderSlotTarget{Page: req.Page, Row: i/layout.Columns + 1, Column: i%layout.Columns + 1}

			state, cardID, reason, err := s.storedSlotState(ctx, want, known)
			if err != nil {
				return err
			}

			change := dto.BinderSlotChange{Target: target, State: state}
			if cardID != nil {
				change.CardID = *cardID
			}
			changes = append(changes, change)

			slots = append(slots, dto.BinderSlotResult{
				Row:    target.Row,
				Column: target.Column,
				State:  state,
				CardID: cardID,
				Reason: reason,
			})
		}

		if err := s.writeSlots(ctx, c, layout, changes); err != nil {
			return err
		}

		result = dto.BinderPageReplaceResult{Slots: slots}
		return nil
	})
	if err != nil {
		return dto.BinderPageReplaceResult{}, err
	}

	return result, nil
}

func (s *binderSlotService) PlaceCards(ctx context.Context, req dto.BinderPlacementRequest) ([]dto.BinderPlacementResult, error) {
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
		c, layout, err := s.lockBinder(ctx, req.ProfileID, req.CollectionID)
		if err != nil {
			return err
		}

		leafCount, err := s.leaves.CountLeaves(ctx, c.ID)
		if err != nil {
			return err
		}
		occupied, err := s.slots.ListAllOccupiedSlots(ctx, c.ID)
		if err != nil {
			return err
		}
		empty := emptySlots(layout, leafCount, occupied)

		var changes []dto.BinderSlotChange
		for _, item := range req.Items {
			exists, err := s.cards.CardExists(ctx, item.CardID)
			if err != nil {
				return err
			}
			if !exists {
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

		return s.writeSlots(ctx, c, layout, changes)
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

// lockBinder row-locks the Collection the caller owns and returns it with its
// Layout. A plain Collection has no Layout, so it is refused as not found.
func (s *binderSlotService) lockBinder(ctx context.Context, profileID, collectionID uuid.UUID) (entity.Collection, entity.CollectionLayout, error) {
	c, err := s.collections.GetOwnedCollection(ctx, profileID, collectionID, true)
	if err != nil {
		return entity.Collection{}, entity.CollectionLayout{}, err
	}

	layout, err := s.layouts.GetLayout(ctx, c.ID)
	if err != nil {
		return entity.Collection{}, entity.CollectionLayout{}, err
	}

	return c, layout, nil
}

// writeSlots sets the given Slots of c, which the caller has locked inside a
// transaction. The caller has checked each target against layout and each Card
// against the database. writeSlots loads only the Slots it touches, applies the
// changes in order, and reconciles each affected Card's entry once.
func (s *binderSlotService) writeSlots(ctx context.Context, c entity.Collection, layout entity.CollectionLayout, changes []dto.BinderSlotChange) error {
	type resolvedChange struct {
		change  dto.BinderSlotChange
		address repository.BinderSlotAddress
		cardID  *uuid.UUID
	}

	leaves := make(map[int]entity.BinderLeaf)
	resolved := make([]resolvedChange, 0, len(changes))
	addresses := make([]repository.BinderSlotAddress, 0, len(changes))
	for _, change := range changes {
		leafNumber, side := mapper.LeafOf(change.Target.Page)
		leaf, ok := leaves[leafNumber]
		if !ok {
			var err error
			leaf, err = s.leaves.GetLeafAt(ctx, c.ID, leafNumber)
			if err != nil {
				return err
			}
			leaves[leafNumber] = leaf
		}

		var cardID *uuid.UUID
		if change.State == dto.BinderSlotCard {
			id := change.CardID
			cardID = &id
		}

		address := repository.BinderSlotAddress{LeafID: leaf.ID, Side: side, Row: change.Target.Row, Column: change.Target.Column}
		resolved = append(resolved, resolvedChange{change: change, address: address, cardID: cardID})
		addresses = append(addresses, address)
	}

	occupied, err := s.slots.ListSlotsAt(ctx, c.ID, addresses)
	if err != nil {
		return err
	}
	held := make(map[repository.BinderSlotAddress]entity.BinderSlot, len(occupied))
	for _, slot := range occupied {
		held[repository.BinderSlotAddress{LeafID: slot.LeafID, Side: slot.Side, Row: slot.Row, Column: slot.Column}] = slot
	}

	var touched []uuid.UUID
	seen := make(map[uuid.UUID]bool)
	touch := func(id uuid.UUID) {
		if !seen[id] {
			seen[id] = true
			touched = append(touched, id)
		}
	}

	for _, r := range resolved {
		prev, existed := held[r.address]
		if existed && prev.CardID != nil {
			touch(*prev.CardID)
		}

		switch {
		case r.change.State == dto.BinderSlotEmpty:
			if existed {
				if err := s.slots.Delete(ctx, prev); err != nil {
					return err
				}
				delete(held, r.address)
			}
		case !existed:
			slot, err := s.slots.Insert(ctx, entity.BinderSlot{
				CollectionID: c.ID,
				LeafID:       r.address.LeafID,
				Side:         r.address.Side,
				Row:          r.address.Row,
				Column:       r.address.Column,
				CardID:       r.cardID,
				Unidentified: r.change.State == dto.BinderSlotUnidentified,
			})
			if err != nil {
				return err
			}
			held[r.address] = slot
		default:
			prev.CardID = r.cardID
			prev.Unidentified = r.change.State == dto.BinderSlotUnidentified
			slot, err := s.slots.Update(ctx, prev)
			if err != nil {
				return err
			}
			held[r.address] = slot
		}

		if r.cardID != nil {
			touch(*r.cardID)
		}
	}

	for _, id := range touched {
		if err := s.reconcileEntry(ctx, c.ID, id); err != nil {
			return err
		}
	}

	return nil
}

// storedSlotState is the state a requested Slot is stored in. A Card the server
// does not know is stored Unidentified, with the reason. known caches each
// Card's lookup for the request, so a Card named by many Slots is read once.
func (s *binderSlotService) storedSlotState(ctx context.Context, want dto.BinderSlotState, known map[uuid.UUID]bool) (string, *uuid.UUID, string, error) {
	if want.State != dto.BinderSlotCard {
		return want.State, nil, "", nil
	}

	exists, looked := known[want.CardID]
	if !looked {
		var err error
		exists, err = s.cards.CardExists(ctx, want.CardID)
		if err != nil {
			return "", nil, "", err
		}
		known[want.CardID] = exists
	}
	if !exists {
		return dto.BinderSlotUnidentified, nil, dto.BinderSlotReasonUnknownCard, nil
	}

	cardID := want.CardID
	return dto.BinderSlotCard, &cardID, "", nil
}

// slotAt returns the Slot at target and whether it holds anything. An empty
// Slot comes back as an unsaved row at its address, so a move can fill it.
func (s *binderSlotService) slotAt(ctx context.Context, c entity.Collection, layout entity.CollectionLayout, target dto.BinderSlotTarget) (entity.BinderSlot, bool, error) {
	if err := checkSlotInGrid(layout, target); err != nil {
		return entity.BinderSlot{}, false, err
	}

	leafNumber, side := mapper.LeafOf(target.Page)
	leaf, err := s.leaves.GetLeafAt(ctx, c.ID, leafNumber)
	if err != nil {
		return entity.BinderSlot{}, false, err
	}

	slot, err := s.slots.FindFirst(ctx, slotSpec(c, leaf, side, target.Row, target.Column))
	if err != nil {
		return entity.BinderSlot{}, false, err
	}
	if !slot.IsZero() {
		return slot, true, nil
	}

	return entity.BinderSlot{CollectionID: c.ID, LeafID: leaf.ID, Side: side, Row: target.Row, Column: target.Column}, false, nil
}

// reconcileEntry sets the Card's Inventory Entry to its Slot count: created when
// the Card gains its first Slot, removed when it loses its last.
func (s *binderSlotService) reconcileEntry(ctx context.Context, collectionID, cardID uuid.UUID) error {
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

// emptySlots lists the Slots of the binder that hold nothing, in Page order.
// occupied must carry each Slot's Leaf, so its Page can be computed.
func emptySlots(layout entity.CollectionLayout, leafCount int, occupied []entity.BinderSlot) []dto.BinderSlotTarget {
	taken := make(map[dto.BinderSlotTarget]bool, len(occupied))
	for _, slot := range occupied {
		taken[dto.BinderSlotTarget{Page: mapper.PageOf(slot.Leaf.Position, slot.Side), Row: slot.Row, Column: slot.Column}] = true
	}

	var empty []dto.BinderSlotTarget
	for page := 1; page <= mapper.PageCount(leafCount); page++ {
		for row := 1; row <= layout.Rows; row++ {
			for column := 1; column <= layout.Columns; column++ {
				target := dto.BinderSlotTarget{Page: page, Row: row, Column: column}
				if !taken[target] {
					empty = append(empty, target)
				}
			}
		}
	}

	return empty
}

// checkSlotInGrid refuses a Slot address outside the Layout's rows and columns.
func checkSlotInGrid(layout entity.CollectionLayout, target dto.BinderSlotTarget) error {
	if !inRange(target.Row, layout.Rows) || !inRange(target.Column, layout.Columns) {
		return ungerr.BadRequestError(slotOutOfGridMsg)
	}

	return nil
}

func slotSpec(c entity.Collection, leaf entity.BinderLeaf, side string, row, column int) crud.Specification[entity.BinderSlot] {
	return crud.Specification[entity.BinderSlot]{
		Model: entity.BinderSlot{CollectionID: c.ID, LeafID: leaf.ID, Side: side, Row: row, Column: column},
	}
}
