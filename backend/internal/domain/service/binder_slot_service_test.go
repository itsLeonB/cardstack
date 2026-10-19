package service

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testBinderLeaf is leaf 1 of the fixture's binder, the Leaf behind pages 1 and 2.
func testBinderLeaf(f binderFixture) entity.BinderLeaf {
	return entity.BinderLeaf{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collection.ID, Position: 1}
}

// testBinderSlot is one occupied Slot on the front face of leaf.
func testBinderSlot(f binderFixture, leaf entity.BinderLeaf, row, column int, card *uuid.UUID, unidentified bool) entity.BinderSlot {
	return entity.BinderSlot{
		BaseEntity:   baseEntity(uuid.New()),
		CollectionID: f.collection.ID,
		LeafID:       leaf.ID,
		Side:         dto.BinderSideFront,
		Row:          row,
		Column:       column,
		CardID:       card,
		Unidentified: unidentified,
	}
}

// slotAddressed matches the Slot query for one row and column of any face.
func slotAddressed(row, column int) any {
	return mock.MatchedBy(func(spec crud.Specification[entity.BinderSlot]) bool {
		return spec.Model.Row == row && spec.Model.Column == column
	})
}

// expectSlotRead expects one Slot address to be looked up on the front of leaf 1.
func (f binderFixture) expectSlotRead(leaf entity.BinderLeaf, row, column int, slot entity.BinderSlot) {
	f.leaves.EXPECT().GetLeafAt(f.ctx, f.collection.ID, 1).Return(leaf, nil).Once()
	f.slots.EXPECT().FindFirst(f.ctx, slotAddressed(row, column)).Return(slot, nil).Once()
}

// expectWriteStart expects the owned, row-locked Collection and its Layout, the
// reads every Slot write starts with.
func (f binderFixture) expectWriteStart(rows, columns int) {
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(rows, columns), nil).Once()
}

// expectOccupied expects the write to read the Slots at its addresses, and
// returns the given ones, which are the only occupied Slots among them.
func (f binderFixture) expectOccupied(slots ...entity.BinderSlot) {
	f.slots.EXPECT().ListSlotsAt(f.ctx, f.collection.ID, mock.Anything).Return(slots, nil).Once()
}

// expectNewEntry expects a Card that just gained Slots, with no entry yet, to get
// one whose quantity is its Slot count.
func (f binderFixture) expectNewEntry(cardID uuid.UUID, count int) {
	f.slots.EXPECT().CountCardSlots(f.ctx, f.collection.ID, cardID).Return(count, nil).Once()
	f.entries.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entity.InventoryEntry{}, nil).Once()
	f.entries.EXPECT().Insert(f.ctx, mock.MatchedBy(func(e entity.InventoryEntry) bool {
		return e.CardID == cardID && e.Quantity == count
	})).Return(entity.InventoryEntry{}, nil).Once()
}

func (f binderFixture) moveRequest(from, to dto.BinderSlotTarget) dto.BinderSlotMove {
	return dto.BinderSlotMove{ProfileID: f.profileID, CollectionID: f.collection.ID, From: from, To: to}
}

func (f binderFixture) replaceRequest(page int, slots ...dto.BinderSlotState) dto.BinderPageReplace {
	return dto.BinderPageReplace{ProfileID: f.profileID, CollectionID: f.collection.ID, Page: page, Slots: slots}
}

func emptyState() dto.BinderSlotState { return dto.BinderSlotState{State: dto.BinderSlotEmpty} }

func (f binderFixture) placeRequest(items ...dto.BinderPlacementItem) dto.BinderPlacementRequest {
	return dto.BinderPlacementRequest{ProfileID: f.profileID, CollectionID: f.collection.ID, Items: items}
}

// expectPlacementStart expects the owned, row-locked Collection, its Layout and
// the Leaf count that placement reads before it searches for empty Slots.
func (f binderFixture) expectPlacementStart(rows, columns, leaves int) {
	f.expectWriteStart(rows, columns)
	f.leaves.EXPECT().CountLeaves(f.ctx, f.collection.ID).Return(leaves, nil).Once()
}

// occupiedSlot is a Slot on the front of Leaf 1 holding nothing identified, so
// the placement must treat it as taken.
func occupiedSlot(row, column int) entity.BinderSlot {
	return entity.BinderSlot{
		Side:         dto.BinderSideFront,
		Row:          row,
		Column:       column,
		Unidentified: true,
		Leaf:         entity.BinderLeaf{Position: 1},
	}
}

func (f binderFixture) expectCard(cardID uuid.UUID, found bool) {
	f.cards.EXPECT().Exists(f.ctx, cardID).Return(found, nil).Once()
}

func TestBinderSlotService_MoveSlot_RejectsSourceOutsideLayout(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)

	err := f.slot.MoveSlot(f.ctx, f.moveRequest(
		dto.BinderSlotTarget{Page: 1, Row: 3, Column: 1},
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 2},
	))
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderSlotService_MoveSlot_RejectsTargetOutsideLayout(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)
	leaf := testBinderLeaf(f)
	card := uuid.New()
	f.expectSlotRead(leaf, 1, 1, testBinderSlot(f, leaf, 1, 1, &card, false))

	err := f.slot.MoveSlot(f.ctx, f.moveRequest(
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 3},
	))
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderSlotService_MoveSlot_RejectsEmptySource(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)
	leaf := testBinderLeaf(f)
	f.expectSlotRead(leaf, 1, 1, entity.BinderSlot{})

	err := f.slot.MoveSlot(f.ctx, f.moveRequest(
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 2},
	))
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderSlotService_MoveSlot_SelfMoveChangesNothing(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)
	leaf := testBinderLeaf(f)
	card := uuid.New()
	// Both ends of a self-move are read before the no-op check; no Update is expected.
	slot := testBinderSlot(f, leaf, 1, 1, &card, false)
	f.expectSlotRead(leaf, 1, 1, slot)
	f.expectSlotRead(leaf, 1, 1, slot)

	err := f.slot.MoveSlot(f.ctx, f.moveRequest(
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
	))
	require.NoError(t, err)
}

func TestBinderSlotService_MoveSlot_MovesToEmptySlotOnAnotherPage(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)
	leaf := testBinderLeaf(f)
	card := uuid.New()
	source := testBinderSlot(f, leaf, 1, 1, &card, false)
	f.expectSlotRead(leaf, 1, 1, source)
	// Page 2 is the back of leaf 1, so the same leaf is read with the back side.
	f.leaves.EXPECT().GetLeafAt(f.ctx, f.collection.ID, 1).Return(leaf, nil).Once()
	f.slots.EXPECT().FindFirst(f.ctx, slotAddressed(1, 1)).Return(entity.BinderSlot{}, nil).Once()

	// The card row takes the empty Slot's position and keeps its own ID and card.
	f.slots.EXPECT().Update(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.ID == source.ID && s.Side == dto.BinderSideBack && s.LeafID == leaf.ID &&
			s.Row == 1 && s.Column == 1 && s.CardID != nil && *s.CardID == card && !s.Unidentified
	})).Return(entity.BinderSlot{}, nil).Once()

	err := f.slot.MoveSlot(f.ctx, f.moveRequest(
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		dto.BinderSlotTarget{Page: 2, Row: 1, Column: 1},
	))
	require.NoError(t, err)
}

func TestBinderSlotService_MoveSlot_SwapsWithFilledTarget(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)
	leaf := testBinderLeaf(f)
	cardA, cardB := uuid.New(), uuid.New()
	f.expectSlotRead(leaf, 1, 1, testBinderSlot(f, leaf, 1, 1, &cardA, false))
	f.expectSlotRead(leaf, 2, 2, testBinderSlot(f, leaf, 2, 2, &cardB, false))

	// Each row keeps its position and takes the other card; neither row is deleted or inserted.
	f.slots.EXPECT().Update(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.Row == 1 && s.Column == 1 && s.CardID != nil && *s.CardID == cardB && !s.Unidentified
	})).Return(entity.BinderSlot{}, nil).Once()
	f.slots.EXPECT().Update(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.Row == 2 && s.Column == 2 && s.CardID != nil && *s.CardID == cardA && !s.Unidentified
	})).Return(entity.BinderSlot{}, nil).Once()

	err := f.slot.MoveSlot(f.ctx, f.moveRequest(
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		dto.BinderSlotTarget{Page: 1, Row: 2, Column: 2},
	))
	require.NoError(t, err)
}

func TestBinderSlotService_MoveSlot_SwapsUnidentifiedTargetLikeAnySlot(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)
	leaf := testBinderLeaf(f)
	card := uuid.New()
	f.expectSlotRead(leaf, 1, 1, testBinderSlot(f, leaf, 1, 1, &card, false))
	f.expectSlotRead(leaf, 1, 2, testBinderSlot(f, leaf, 1, 2, nil, true))

	// The card moves to the Unidentified row and the Unidentified mark moves to the card's old row.
	f.slots.EXPECT().Update(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.Row == 1 && s.Column == 1 && s.CardID == nil && s.Unidentified
	})).Return(entity.BinderSlot{}, nil).Once()
	f.slots.EXPECT().Update(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.Row == 1 && s.Column == 2 && s.CardID != nil && *s.CardID == card && !s.Unidentified
	})).Return(entity.BinderSlot{}, nil).Once()

	err := f.slot.MoveSlot(f.ctx, f.moveRequest(
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 2},
	))
	require.NoError(t, err)
}

func TestBinderSlotService_WriteSlot_RejectsSlotOutsideLayout(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)

	err := f.slot.WriteSlot(f.ctx, dto.BinderSlotWrite{
		ProfileID:    f.profileID,
		CollectionID: f.collection.ID,
		Target:       dto.BinderSlotTarget{Page: 1, Row: 3, Column: 1},
		State:        dto.BinderSlotEmpty,
	})
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderSlotService_WriteSlot_RefusesUnknownCardBeforeWriting(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)
	cardID := uuid.New()
	f.cards.EXPECT().Exists(f.ctx, mock.Anything).Return(false, nil).Once()

	err := f.slot.WriteSlot(f.ctx, dto.BinderSlotWrite{
		ProfileID:    f.profileID,
		CollectionID: f.collection.ID,
		Target:       dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		State:        dto.BinderSlotCard,
		CardID:       cardID,
	})
	requireStatus(t, err, http.StatusNotFound)
}

func TestBinderSlotService_WriteSlot_NilCardIDIsNotFoundWithoutQuery(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)

	err := f.slot.WriteSlot(f.ctx, dto.BinderSlotWrite{
		ProfileID:    f.profileID,
		CollectionID: f.collection.ID,
		Target:       dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		State:        dto.BinderSlotCard,
	})
	requireStatus(t, err, http.StatusNotFound)
}

func TestBinderSlotService_ReplacePage_RejectsWrongSlotCountBeforeWriting(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)

	_, err := f.slot.ReplacePage(f.ctx, f.replaceRequest(1, emptyState(), emptyState(), emptyState()))
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderSlotService_ReplacePage_RejectsPageBeyondLastLeafWithNothingWritten(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)
	// Page 3 is the front of leaf 2, which this binder does not have. The write
	// refuses it before it reads or writes any Slot.
	f.leaves.EXPECT().GetLeafAt(f.ctx, f.collection.ID, 2).Return(entity.BinderLeaf{}, ungerr.NotFoundError("no page")).Once()

	_, err := f.slot.ReplacePage(f.ctx, f.replaceRequest(3, emptyState(), emptyState(), emptyState(), emptyState()))
	requireStatus(t, err, http.StatusNotFound)
}

func TestBinderSlotService_ReplacePage_UnknownCardIsStoredUnidentified(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)
	leaf := testBinderLeaf(f)
	f.leaves.EXPECT().GetLeafAt(f.ctx, f.collection.ID, 1).Return(leaf, nil).Once()
	f.cards.EXPECT().Exists(f.ctx, mock.Anything).Return(false, nil).Once()
	f.expectOccupied()

	// Only the unknown card's Slot is written, as Unidentified; the three empty Slots write nothing.
	f.slots.EXPECT().Insert(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.Row == 1 && s.Column == 1 && s.CardID == nil && s.Unidentified
	})).Return(entity.BinderSlot{}, nil).Once()

	result, err := f.slot.ReplacePage(f.ctx, f.replaceRequest(1,
		dto.BinderSlotState{State: dto.BinderSlotCard, CardID: uuid.New()},
		emptyState(), emptyState(), emptyState(),
	))
	require.NoError(t, err)
	require.Len(t, result.Slots, 4)
	assert.Equal(t, dto.BinderSlotUnidentified, result.Slots[0].State)
	assert.Equal(t, dto.BinderSlotReasonUnknownCard, result.Slots[0].Reason)
	assert.Nil(t, result.Slots[0].CardID)
}

func TestBinderSlotService_ReplacePage_NilCardIDIsUnknownWithoutLookup(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)
	leaf := testBinderLeaf(f)
	f.leaves.EXPECT().GetLeafAt(f.ctx, f.collection.ID, 1).Return(leaf, nil).Once()
	// No cards expectation: a nil id must never reach the Card lookup, which would match every Card.
	f.expectOccupied()
	f.slots.EXPECT().Insert(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.Row == 1 && s.Column == 1 && s.CardID == nil && s.Unidentified
	})).Return(entity.BinderSlot{}, nil).Once()

	result, err := f.slot.ReplacePage(f.ctx, f.replaceRequest(1,
		dto.BinderSlotState{State: dto.BinderSlotCard},
		emptyState(), emptyState(), emptyState(),
	))
	require.NoError(t, err)
	assert.Equal(t, dto.BinderSlotUnidentified, result.Slots[0].State)
	assert.Equal(t, dto.BinderSlotReasonUnknownCard, result.Slots[0].Reason)
}

func TestBinderSlotService_ReplacePage_ReconcilesEachCardOnce(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)
	leaf := testBinderLeaf(f)
	f.leaves.EXPECT().GetLeafAt(f.ctx, f.collection.ID, 1).Return(leaf, nil).Once()
	card := entity.Card{BaseEntity: baseEntity(uuid.New())}
	// Both copies name the card, which is looked up once for the whole Page.
	f.cards.EXPECT().Exists(f.ctx, card.ID).Return(true, nil).Once()
	f.expectOccupied()
	f.slots.EXPECT().Insert(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.CardID != nil && *s.CardID == card.ID
	})).Return(entity.BinderSlot{}, nil).Twice()

	// Both copies hold the card, so its entry is reconciled once, not per Slot.
	f.expectNewEntry(card.ID, 2)

	_, err := f.slot.ReplacePage(f.ctx, f.replaceRequest(1,
		dto.BinderSlotState{State: dto.BinderSlotCard, CardID: card.ID},
		dto.BinderSlotState{State: dto.BinderSlotCard, CardID: card.ID},
		emptyState(), emptyState(),
	))
	require.NoError(t, err)
}

func TestBinderSlotService_ReplacePage_CardLeavingThePageIsDeletedAndReconciled(t *testing.T) {
	f := newBinderFixture(t)
	f.expectWriteStart(2, 2)
	leaf := testBinderLeaf(f)
	f.leaves.EXPECT().GetLeafAt(f.ctx, f.collection.ID, 1).Return(leaf, nil).Once()
	cardID := uuid.New()
	held := testBinderSlot(f, leaf, 1, 1, &cardID, false)
	f.expectOccupied(held)
	// The held Slot is emptied, which the write does by deleting its row.
	f.slots.EXPECT().Delete(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.ID == held.ID
	})).Return(nil).Once()

	// The card has no Slots left in the binder, so its entry goes.
	f.slots.EXPECT().CountCardSlots(f.ctx, f.collection.ID, cardID).Return(0, nil).Once()
	entry := entity.InventoryEntry{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collection.ID, CardID: cardID, Quantity: 1}
	f.entries.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entry, nil).Once()
	f.entries.EXPECT().Delete(f.ctx, entry).Return(nil).Once()

	_, err := f.slot.ReplacePage(f.ctx, f.replaceRequest(1, emptyState(), emptyState(), emptyState(), emptyState()))
	require.NoError(t, err)
}

func TestBinderSlotService_PlaceCards_RejectsQuantityBelowOne(t *testing.T) {
	f := newBinderFixture(t)

	_, err := f.slot.PlaceCards(f.ctx, f.placeRequest(dto.BinderPlacementItem{CardID: uuid.New(), Quantity: 0}))
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderSlotService_PlaceCards_RejectsDuplicateCard(t *testing.T) {
	f := newBinderFixture(t)
	cardID := uuid.New()

	_, err := f.slot.PlaceCards(f.ctx, f.placeRequest(
		dto.BinderPlacementItem{CardID: cardID, Quantity: 1},
		dto.BinderPlacementItem{CardID: cardID, Quantity: 2},
	))
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderSlotService_PlaceCards_RefusesPlainCollectionAsNotFound(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(entity.CollectionLayout{}, ungerr.NotFoundError("this Collection is not a binder")).Once()

	_, err := f.slot.PlaceCards(f.ctx, f.placeRequest(dto.BinderPlacementItem{CardID: uuid.New(), Quantity: 1}))
	requireStatus(t, err, http.StatusNotFound)
}

// TestBinderSlotService_PlaceCards_PlacesWhatFitsAndDeclinesTheRest: the first
// item fits whole, the second takes the one empty Slot left and declines the
// rest. The occupied Slot is skipped, and the copies are written in one batch.
func TestBinderSlotService_PlaceCards_PlacesWhatFitsAndDeclinesTheRest(t *testing.T) {
	f := newBinderFixture(t)
	a, b := uuid.New(), uuid.New()
	f.expectPlacementStart(1, 2, 1)
	f.slots.EXPECT().ListAllOccupiedSlots(f.ctx, f.collection.ID).Return([]entity.BinderSlot{occupiedSlot(1, 1)}, nil).Once()
	f.expectCard(a, true)
	f.expectCard(b, true)

	// Both pages sit on leaf 1, so the Leaf is read once for the whole batch.
	leaf := testBinderLeaf(f)
	f.leaves.EXPECT().GetLeafAt(f.ctx, f.collection.ID, 1).Return(leaf, nil).Once()
	f.expectOccupied()
	f.slots.EXPECT().Insert(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.Side == dto.BinderSideFront && s.Row == 1 && s.Column == 2 && s.CardID != nil && *s.CardID == a
	})).Return(entity.BinderSlot{}, nil).Once()
	f.slots.EXPECT().Insert(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.Side == dto.BinderSideBack && s.Row == 1 && s.Column == 1 && s.CardID != nil && *s.CardID == a
	})).Return(entity.BinderSlot{}, nil).Once()
	f.slots.EXPECT().Insert(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.Side == dto.BinderSideBack && s.Row == 1 && s.Column == 2 && s.CardID != nil && *s.CardID == b
	})).Return(entity.BinderSlot{}, nil).Once()
	f.expectNewEntry(a, 2)
	f.expectNewEntry(b, 1)

	results, err := f.slot.PlaceCards(f.ctx, f.placeRequest(
		dto.BinderPlacementItem{CardID: a, Quantity: 2},
		dto.BinderPlacementItem{CardID: b, Quantity: 5},
	))
	require.NoError(t, err)
	assert.Equal(t, []dto.BinderPlacementResult{
		{CardID: a, Placed: 2},
		{CardID: b, Placed: 1, Declined: 4, Reason: dto.BinderPlacementReasonNotEnoughEmptySlots, Message: placementNotEnoughEmptyMsg},
	}, results)
}

// TestBinderSlotService_PlaceCards_NothingPlacedSkipsWrite: with no empty Slot
// left, a copy is declined with the reason and no Slot is written.
func TestBinderSlotService_PlaceCards_NothingPlacedSkipsWrite(t *testing.T) {
	f := newBinderFixture(t)
	a := uuid.New()
	f.expectPlacementStart(1, 1, 1)
	back := occupiedSlot(1, 1)
	back.Side = dto.BinderSideBack
	f.slots.EXPECT().ListAllOccupiedSlots(f.ctx, f.collection.ID).Return([]entity.BinderSlot{occupiedSlot(1, 1), back}, nil).Once()
	f.expectCard(a, true)

	results, err := f.slot.PlaceCards(f.ctx, f.placeRequest(dto.BinderPlacementItem{CardID: a, Quantity: 1}))
	require.NoError(t, err)
	assert.Equal(t, []dto.BinderPlacementResult{
		{CardID: a, Declined: 1, Reason: dto.BinderPlacementReasonNotEnoughEmptySlots, Message: placementNotEnoughEmptyMsg},
	}, results)
}

// TestBinderSlotService_PlaceCards_UnknownCardIsDeclinedNotWritten: an unknown
// Card is declined with card_not_found and takes no Slot, so the other item
// still lands.
func TestBinderSlotService_PlaceCards_UnknownCardIsDeclinedNotWritten(t *testing.T) {
	f := newBinderFixture(t)
	unknown, a := uuid.New(), uuid.New()
	f.expectPlacementStart(1, 1, 1)
	f.slots.EXPECT().ListAllOccupiedSlots(f.ctx, f.collection.ID).Return(nil, nil).Once()
	f.expectCard(unknown, false)
	f.expectCard(a, true)

	leaf := testBinderLeaf(f)
	f.leaves.EXPECT().GetLeafAt(f.ctx, f.collection.ID, 1).Return(leaf, nil).Once()
	f.expectOccupied()
	f.slots.EXPECT().Insert(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.Row == 1 && s.Column == 1 && s.CardID != nil && *s.CardID == a
	})).Return(entity.BinderSlot{}, nil).Once()
	f.expectNewEntry(a, 1)

	results, err := f.slot.PlaceCards(f.ctx, f.placeRequest(
		dto.BinderPlacementItem{CardID: unknown, Quantity: 2},
		dto.BinderPlacementItem{CardID: a, Quantity: 1},
	))
	require.NoError(t, err)
	assert.Equal(t, []dto.BinderPlacementResult{
		{CardID: unknown, Declined: 2, Reason: dto.InventoryReasonCardNotFound, Message: cardNotFoundMsg},
		{CardID: a, Placed: 1},
	}, results)
}
