package service

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (f binderFixture) replaceRequest(page int, slots ...dto.BinderSlotState) dto.BinderPageReplace {
	return dto.BinderPageReplace{ProfileID: f.profileID, CollectionID: f.collection.ID, Page: page, Slots: slots}
}

func emptyState() dto.BinderSlotState { return dto.BinderSlotState{State: dto.BinderSlotEmpty} }

// expectReplaceStart expects the two locked Collection and Layout reads a Page
// replace makes: one in ReplacePage and one in the WriteSlots batch it calls.
func (f binderFixture) expectReplaceStart(rows, columns int) {
	for range 2 {
		f.lockedCollection()
		f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(rows, columns), nil).Once()
	}
}

// expectOccupied expects the batch to read the binder's occupied Slots once.
func (f binderFixture) expectOccupied(slots ...entity.BinderSlot) {
	f.slots.EXPECT().ListAllOccupiedSlots(f.ctx, f.collection.ID).Return(slots, nil).Once()
}

func TestBinderService_ReplacePage_RejectsWrongSlotCountBeforeWriting(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(2, 2), nil).Once()

	_, err := f.svc.ReplacePage(f.ctx, f.replaceRequest(1, emptyState(), emptyState(), emptyState()))
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderService_ReplacePage_RejectsPageBeyondLastLeafWithNothingWritten(t *testing.T) {
	f := newBinderFixture(t)
	f.expectReplaceStart(2, 2)
	// Page 3 is the front of leaf 2, which this binder does not have. The batch
	// refuses it before it reads or writes any Slot.
	f.leaves.EXPECT().FindLeafAt(f.ctx, f.collection.ID, 2).Return(entity.BinderLeaf{}, nil).Once()

	_, err := f.svc.ReplacePage(f.ctx, f.replaceRequest(3, emptyState(), emptyState(), emptyState(), emptyState()))
	requireStatus(t, err, http.StatusNotFound)
}

func TestBinderService_ReplacePage_UnknownCardIsStoredUnidentified(t *testing.T) {
	f := newBinderFixture(t)
	f.expectReplaceStart(2, 2)
	leaf := testBinderLeaf(f)
	f.leaves.EXPECT().FindLeafAt(f.ctx, f.collection.ID, 1).Return(leaf, nil).Once()
	f.cards.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entity.Card{}, nil).Once()
	f.expectOccupied()

	// Only the unknown card's Slot is written, as Unidentified; the three empty Slots write nothing.
	f.slots.EXPECT().Insert(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.Row == 1 && s.Column == 1 && s.CardID == nil && s.Unidentified
	})).Return(entity.BinderSlot{}, nil).Once()

	result, err := f.svc.ReplacePage(f.ctx, f.replaceRequest(1,
		dto.BinderSlotState{State: dto.BinderSlotCard, CardID: uuid.New()},
		emptyState(), emptyState(), emptyState(),
	))
	require.NoError(t, err)
	require.Len(t, result.Slots, 4)
	assert.Equal(t, dto.BinderSlotUnidentified, result.Slots[0].State)
	assert.Equal(t, dto.BinderSlotReasonUnknownCard, result.Slots[0].Reason)
	assert.Nil(t, result.Slots[0].CardID)
}

func TestBinderService_ReplacePage_NilCardIDIsUnknownWithoutLookup(t *testing.T) {
	f := newBinderFixture(t)
	f.expectReplaceStart(2, 2)
	leaf := testBinderLeaf(f)
	f.leaves.EXPECT().FindLeafAt(f.ctx, f.collection.ID, 1).Return(leaf, nil).Once()
	// No cards expectation: a nil id must never reach the Card lookup, which would match every Card.
	f.expectOccupied()
	f.slots.EXPECT().Insert(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.Row == 1 && s.Column == 1 && s.CardID == nil && s.Unidentified
	})).Return(entity.BinderSlot{}, nil).Once()

	result, err := f.svc.ReplacePage(f.ctx, f.replaceRequest(1,
		dto.BinderSlotState{State: dto.BinderSlotCard},
		emptyState(), emptyState(), emptyState(),
	))
	require.NoError(t, err)
	assert.Equal(t, dto.BinderSlotUnidentified, result.Slots[0].State)
	assert.Equal(t, dto.BinderSlotReasonUnknownCard, result.Slots[0].Reason)
}

func TestBinderService_ReplacePage_ReconcilesEachCardOnce(t *testing.T) {
	f := newBinderFixture(t)
	f.expectReplaceStart(2, 2)
	leaf := testBinderLeaf(f)
	f.leaves.EXPECT().FindLeafAt(f.ctx, f.collection.ID, 1).Return(leaf, nil).Once()
	card := entity.Card{BaseEntity: baseEntity(uuid.New())}
	// Each copy is checked once by ReplacePage, and the batch checks the card once more.
	f.cards.EXPECT().FindFirst(f.ctx, mock.Anything).Return(card, nil).Times(3)
	f.expectOccupied()
	f.slots.EXPECT().Insert(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.CardID != nil && *s.CardID == card.ID
	})).Return(entity.BinderSlot{}, nil).Twice()

	// Both copies hold the card, so its entry is reconciled once, not per Slot.
	f.slots.EXPECT().CountCardSlots(f.ctx, f.collection.ID, card.ID).Return(2, nil).Once()
	f.entries.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entity.InventoryEntry{}, nil).Once()
	f.entries.EXPECT().Insert(f.ctx, mock.MatchedBy(func(e entity.InventoryEntry) bool {
		return e.CardID == card.ID && e.Quantity == 2
	})).Return(entity.InventoryEntry{}, nil).Once()

	_, err := f.svc.ReplacePage(f.ctx, f.replaceRequest(1,
		dto.BinderSlotState{State: dto.BinderSlotCard, CardID: card.ID},
		dto.BinderSlotState{State: dto.BinderSlotCard, CardID: card.ID},
		emptyState(), emptyState(),
	))
	require.NoError(t, err)
}

func TestBinderService_ReplacePage_CardLeavingThePageIsDeletedAndReconciled(t *testing.T) {
	f := newBinderFixture(t)
	f.expectReplaceStart(2, 2)
	leaf := testBinderLeaf(f)
	f.leaves.EXPECT().FindLeafAt(f.ctx, f.collection.ID, 1).Return(leaf, nil).Once()
	cardID := uuid.New()
	held := testBinderSlot(f, leaf, 1, 1, &cardID, false)
	held.Leaf = leaf
	f.expectOccupied(held)
	// The held Slot is emptied, which the batch does by deleting its row.
	f.slots.EXPECT().Delete(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.ID == held.ID
	})).Return(nil).Once()

	// The card has no Slots left in the binder, so its entry goes.
	f.slots.EXPECT().CountCardSlots(f.ctx, f.collection.ID, cardID).Return(0, nil).Once()
	entry := entity.InventoryEntry{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collection.ID, CardID: cardID, Quantity: 1}
	f.entries.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entry, nil).Once()
	f.entries.EXPECT().Delete(f.ctx, entry).Return(nil).Once()

	_, err := f.svc.ReplacePage(f.ctx, f.replaceRequest(1, emptyState(), emptyState(), emptyState(), emptyState()))
	require.NoError(t, err)
}
