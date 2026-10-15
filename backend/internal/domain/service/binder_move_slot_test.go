package service

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
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
	f.leaves.EXPECT().FindLeafAt(f.ctx, f.collection.ID, 1).Return(leaf, nil).Once()
	f.slots.EXPECT().FindFirst(f.ctx, slotAddressed(row, column)).Return(slot, nil).Once()
}

func (f binderFixture) moveRequest(from, to dto.BinderSlotTarget) dto.BinderSlotMove {
	return dto.BinderSlotMove{ProfileID: f.profileID, CollectionID: f.collection.ID, From: from, To: to}
}

func TestBinderService_MoveSlot_RejectsSourceOutsideLayout(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(2, 2), nil).Once()

	err := f.svc.MoveSlot(f.ctx, f.moveRequest(
		dto.BinderSlotTarget{Page: 1, Row: 3, Column: 1},
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 2},
	))
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderService_MoveSlot_RejectsTargetOutsideLayout(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(2, 2), nil).Once()
	leaf := testBinderLeaf(f)
	card := uuid.New()
	f.expectSlotRead(leaf, 1, 1, testBinderSlot(f, leaf, 1, 1, &card, false))

	err := f.svc.MoveSlot(f.ctx, f.moveRequest(
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 3},
	))
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderService_MoveSlot_RejectsEmptySource(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(2, 2), nil).Once()
	leaf := testBinderLeaf(f)
	f.expectSlotRead(leaf, 1, 1, entity.BinderSlot{})

	err := f.svc.MoveSlot(f.ctx, f.moveRequest(
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 2},
	))
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderService_MoveSlot_SelfMoveChangesNothing(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(2, 2), nil).Once()
	leaf := testBinderLeaf(f)
	card := uuid.New()
	// Both ends of a self-move are read before the no-op check; no Update is expected.
	slot := testBinderSlot(f, leaf, 1, 1, &card, false)
	f.expectSlotRead(leaf, 1, 1, slot)
	f.expectSlotRead(leaf, 1, 1, slot)

	err := f.svc.MoveSlot(f.ctx, f.moveRequest(
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
	))
	require.NoError(t, err)
}

func TestBinderService_MoveSlot_MovesToEmptySlotOnAnotherPage(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(2, 2), nil).Once()
	leaf := testBinderLeaf(f)
	card := uuid.New()
	source := testBinderSlot(f, leaf, 1, 1, &card, false)
	f.expectSlotRead(leaf, 1, 1, source)
	// Page 2 is the back of leaf 1, so the same leaf is read with the back side.
	f.leaves.EXPECT().FindLeafAt(f.ctx, f.collection.ID, 1).Return(leaf, nil).Once()
	f.slots.EXPECT().FindFirst(f.ctx, slotAddressed(1, 1)).Return(entity.BinderSlot{}, nil).Once()

	// The card row takes the empty Slot's position and keeps its own ID and card.
	f.slots.EXPECT().Update(f.ctx, mock.MatchedBy(func(s entity.BinderSlot) bool {
		return s.ID == source.ID && s.Side == dto.BinderSideBack && s.LeafID == leaf.ID &&
			s.Row == 1 && s.Column == 1 && s.CardID != nil && *s.CardID == card && !s.Unidentified
	})).Return(entity.BinderSlot{}, nil).Once()

	err := f.svc.MoveSlot(f.ctx, f.moveRequest(
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		dto.BinderSlotTarget{Page: 2, Row: 1, Column: 1},
	))
	require.NoError(t, err)
}

func TestBinderService_MoveSlot_SwapsWithFilledTarget(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(2, 2), nil).Once()
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

	err := f.svc.MoveSlot(f.ctx, f.moveRequest(
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		dto.BinderSlotTarget{Page: 1, Row: 2, Column: 2},
	))
	require.NoError(t, err)
}

func TestBinderService_MoveSlot_SwapsUnidentifiedTargetLikeAnySlot(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(2, 2), nil).Once()
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

	err := f.svc.MoveSlot(f.ctx, f.moveRequest(
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 1},
		dto.BinderSlotTarget{Page: 1, Row: 1, Column: 2},
	))
	require.NoError(t, err)
}
