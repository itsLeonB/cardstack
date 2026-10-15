package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type placementFixture struct {
	ctx         context.Context
	profileID   uuid.UUID
	collection  entity.Collection
	collections *mocks.MockCollectionRepository
	layouts     *mocks.MockBinderLayoutRepository
	leaves      *mocks.MockBinderLeafRepository
	slots       *mocks.MockBinderSlotRepository
	cards       *mocks.MockRepository[entity.Card]
	binder      *mocks.MockBinderService
	svc         BinderPlacementService
}

// newPlacementFixture's Transactor mock runs the transaction body inline. Every
// repository and binder mock is strict, so a call a test does not expect fails it.
func newPlacementFixture(t *testing.T) placementFixture {
	t.Helper()
	f := placementFixture{ctx: context.Background(), profileID: uuid.New()}
	f.collection = entity.Collection{BaseEntity: baseEntity(uuid.New()), ProfileID: f.profileID, Title: "Binder"}
	f.collections = mocks.NewMockCollectionRepository(t)
	f.layouts = mocks.NewMockBinderLayoutRepository(t)
	f.leaves = mocks.NewMockBinderLeafRepository(t)
	f.slots = mocks.NewMockBinderSlotRepository(t)
	f.cards = mocks.NewMockRepository[entity.Card](t)
	f.binder = mocks.NewMockBinderService(t)
	transactor := mocks.NewMockTransactor(t)
	transactor.EXPECT().WithinTransaction(f.ctx, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Maybe()
	f.svc = NewBinderPlacementService(transactor, f.collections, f.layouts, f.leaves, f.slots, f.cards, f.binder)
	return f
}

// lockedBinder expects the owned, row-locked lookup and a binder Layout of rows by columns with leaves Leaves.
func (f placementFixture) lockedBinder(rows, columns, leaves int) {
	f.collections.EXPECT().GetOwnedCollection(f.ctx, f.profileID, f.collection.ID, true).Return(f.collection, nil).Once()
	f.layouts.EXPECT().IsBinder(f.ctx, f.collection.ID).Return(true, nil).Once()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).
		Return(entity.CollectionLayout{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collection.ID, Rows: rows, Columns: columns}, nil).Once()
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

func (f placementFixture) expectCard(cardID uuid.UUID, found bool) {
	card := entity.Card{}
	if found {
		card = entity.Card{BaseEntity: baseEntity(cardID)}
	}
	f.cards.EXPECT().FindFirst(f.ctx, mock.MatchedBy(func(spec crud.Specification[entity.Card]) bool {
		return spec.Model.ID == cardID
	})).Return(card, nil).Once()
}

func (f placementFixture) req(items ...dto.BinderPlacementItem) dto.BinderPlacementRequest {
	return dto.BinderPlacementRequest{ProfileID: f.profileID, CollectionID: f.collection.ID, Items: items}
}

func TestBinderPlacementService_RejectsQuantityBelowOne(t *testing.T) {
	f := newPlacementFixture(t)

	_, err := f.svc.PlaceCards(f.ctx, f.req(dto.BinderPlacementItem{CardID: uuid.New(), Quantity: 0}))
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderPlacementService_RejectsDuplicateCard(t *testing.T) {
	f := newPlacementFixture(t)
	cardID := uuid.New()

	_, err := f.svc.PlaceCards(f.ctx, f.req(
		dto.BinderPlacementItem{CardID: cardID, Quantity: 1},
		dto.BinderPlacementItem{CardID: cardID, Quantity: 2},
	))
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderPlacementService_RefusesPlainCollection(t *testing.T) {
	f := newPlacementFixture(t)
	f.collections.EXPECT().GetOwnedCollection(f.ctx, f.profileID, f.collection.ID, true).Return(f.collection, nil).Once()
	f.layouts.EXPECT().IsBinder(f.ctx, f.collection.ID).Return(false, nil).Once()

	_, err := f.svc.PlaceCards(f.ctx, f.req(dto.BinderPlacementItem{CardID: uuid.New(), Quantity: 1}))
	requireStatus(t, err, http.StatusConflict)
}

// TestBinderPlacementService_PlacesWhatFitsAndDeclinesTheRest: the first item
// fits whole, the second takes the one empty Slot left and declines the rest.
// The occupied Slot is skipped, and the Changes go to WriteSlots in one call.
func TestBinderPlacementService_PlacesWhatFitsAndDeclinesTheRest(t *testing.T) {
	f := newPlacementFixture(t)
	a, b := uuid.New(), uuid.New()
	f.lockedBinder(1, 2, 1)
	f.slots.EXPECT().ListAllOccupiedSlots(f.ctx, f.collection.ID).Return([]entity.BinderSlot{occupiedSlot(1, 1)}, nil).Once()
	f.expectCard(a, true)
	f.expectCard(b, true)
	f.binder.EXPECT().WriteSlots(f.ctx, dto.BinderSlotsWrite{
		ProfileID:    f.profileID,
		CollectionID: f.collection.ID,
		Changes: []dto.BinderSlotChange{
			{Target: dto.BinderSlotTarget{Page: 1, Row: 1, Column: 2}, State: dto.BinderSlotCard, CardID: a},
			{Target: dto.BinderSlotTarget{Page: 2, Row: 1, Column: 1}, State: dto.BinderSlotCard, CardID: a},
			{Target: dto.BinderSlotTarget{Page: 2, Row: 1, Column: 2}, State: dto.BinderSlotCard, CardID: b},
		},
	}).Return(nil).Once()

	results, err := f.svc.PlaceCards(f.ctx, f.req(
		dto.BinderPlacementItem{CardID: a, Quantity: 2},
		dto.BinderPlacementItem{CardID: b, Quantity: 5},
	))
	require.NoError(t, err)
	assert.Equal(t, []dto.BinderPlacementResult{
		{CardID: a, Placed: 2},
		{CardID: b, Placed: 1, Declined: 4, Reason: dto.BinderPlacementReasonNotEnoughEmptySlots, Message: placementNotEnoughEmptyMsg},
	}, results)
}

// TestBinderPlacementService_NothingPlacedSkipsWrite: with no empty Slot left,
// a copy is declined with the reason and WriteSlots is not called.
func TestBinderPlacementService_NothingPlacedSkipsWrite(t *testing.T) {
	f := newPlacementFixture(t)
	a := uuid.New()
	f.lockedBinder(1, 1, 1)
	back := occupiedSlot(1, 1)
	back.Side = dto.BinderSideBack
	f.slots.EXPECT().ListAllOccupiedSlots(f.ctx, f.collection.ID).Return([]entity.BinderSlot{occupiedSlot(1, 1), back}, nil).Once()
	f.expectCard(a, true)

	results, err := f.svc.PlaceCards(f.ctx, f.req(dto.BinderPlacementItem{CardID: a, Quantity: 1}))
	require.NoError(t, err)
	assert.Equal(t, []dto.BinderPlacementResult{
		{CardID: a, Declined: 1, Reason: dto.BinderPlacementReasonNotEnoughEmptySlots, Message: placementNotEnoughEmptyMsg},
	}, results)
}

// TestBinderPlacementService_UnknownCardIsDeclinedNotWritten: an unknown Card is
// declined with card_not_found and takes no Slot, so the other item still lands.
func TestBinderPlacementService_UnknownCardIsDeclinedNotWritten(t *testing.T) {
	f := newPlacementFixture(t)
	unknown, a := uuid.New(), uuid.New()
	f.lockedBinder(1, 1, 1)
	f.slots.EXPECT().ListAllOccupiedSlots(f.ctx, f.collection.ID).Return(nil, nil).Once()
	f.expectCard(unknown, false)
	f.expectCard(a, true)
	f.binder.EXPECT().WriteSlots(f.ctx, dto.BinderSlotsWrite{
		ProfileID:    f.profileID,
		CollectionID: f.collection.ID,
		Changes: []dto.BinderSlotChange{
			{Target: dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1}, State: dto.BinderSlotCard, CardID: a},
		},
	}).Return(nil).Once()

	results, err := f.svc.PlaceCards(f.ctx, f.req(
		dto.BinderPlacementItem{CardID: unknown, Quantity: 2},
		dto.BinderPlacementItem{CardID: a, Quantity: 1},
	))
	require.NoError(t, err)
	assert.Equal(t, []dto.BinderPlacementResult{
		{CardID: unknown, Declined: 2, Reason: dto.InventoryReasonCardNotFound, Message: cardNotFoundMsg},
		{CardID: a, Placed: 1},
	}, results)
}
