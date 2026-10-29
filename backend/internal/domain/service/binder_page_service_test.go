package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	"github.com/itsLeonB/ungerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type binderPageFixture struct {
	ctx          context.Context
	collectionID uuid.UUID
	layouts      *mocks.MockBinderLayoutRepository
	leaves       *mocks.MockBinderLeafRepository
	slots        *mocks.MockBinderSlotRepository
	svc          BinderPageService
}

func newBinderPageFixture(t *testing.T) binderPageFixture {
	t.Helper()
	f := binderPageFixture{ctx: context.Background(), collectionID: uuid.New()}
	f.layouts = mocks.NewMockBinderLayoutRepository(t)
	f.leaves = mocks.NewMockBinderLeafRepository(t)
	f.slots = mocks.NewMockBinderSlotRepository(t)
	f.svc = NewBinderPageService(f.layouts, f.leaves, f.slots, mapper.NewImageHost(""))
	return f
}

func (f binderPageFixture) layout(rows, columns int) entity.CollectionLayout {
	return entity.CollectionLayout{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collectionID, Rows: rows, Columns: columns}
}

// Page 1 is the front of Leaf 1, page 2 its back, and so on in turning order.
func TestBinderPageService_Page_MapsEachPageToItsLeafFace(t *testing.T) {
	cases := []struct {
		page, leaf int
		side       string
	}{
		{page: 1, leaf: 1, side: dto.BinderSideFront},
		{page: 2, leaf: 1, side: dto.BinderSideBack},
		{page: 3, leaf: 2, side: dto.BinderSideFront},
		{page: 4, leaf: 2, side: dto.BinderSideBack},
	}
	for _, tc := range cases {
		t.Run(tc.side, func(t *testing.T) {
			f := newBinderPageFixture(t)
			f.layouts.EXPECT().GetLayout(f.ctx, f.collectionID).Return(f.layout(2, 2), nil).Once()
			f.leaves.EXPECT().CountLeaves(f.ctx, f.collectionID).Return(2, nil).Once()
			leaf := entity.BinderLeaf{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collectionID, Position: tc.leaf}
			f.leaves.EXPECT().GetLeafAt(f.ctx, f.collectionID, tc.leaf).Return(leaf, nil).Once()
			f.slots.EXPECT().ListSlotsOnPage(f.ctx, leaf.ID, tc.side, false).Return(nil, nil).Once()
			f.slots.EXPECT().CountSlots(f.ctx, f.collectionID, false).Return(0, 0, nil).Once()

			got, err := f.svc.Page(f.ctx, dto.BinderPageReadRequest{CollectionID: f.collectionID, Page: tc.page})
			require.NoError(t, err)
			assert.Equal(t, tc.page, got.Page)
			assert.Equal(t, tc.leaf, got.Leaf)
			assert.Equal(t, tc.side, got.Side)
		})
	}
}

func TestBinderPageService_Page_OwnerViewCountsUnpublishedCardsAndBuildsSlots(t *testing.T) {
	f := newBinderPageFixture(t)
	f.layouts.EXPECT().GetLayout(f.ctx, f.collectionID).Return(f.layout(2, 2), nil).Once()
	f.leaves.EXPECT().CountLeaves(f.ctx, f.collectionID).Return(2, nil).Once()
	leaf := entity.BinderLeaf{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collectionID, Position: 2}
	f.leaves.EXPECT().GetLeafAt(f.ctx, f.collectionID, 2).Return(leaf, nil).Once()
	onPage := []entity.BinderSlot{{Row: 1, Column: 1, Unidentified: true}}
	f.slots.EXPECT().ListSlotsOnPage(f.ctx, leaf.ID, dto.BinderSideBack, true).Return(onPage, nil).Once()
	f.slots.EXPECT().CountSlots(f.ctx, f.collectionID, true).Return(2, 1, nil).Once()

	got, err := f.svc.Page(f.ctx, dto.BinderPageReadRequest{CollectionID: f.collectionID, Page: 4, IncludeUnpublished: true})
	require.NoError(t, err)
	assert.Equal(t, dto.BinderLayout{Rows: 2, Columns: 2, LeafCount: 2, Capacity: 16}, got.Layout)
	assert.Equal(t, 3, got.CardTotal, "cards plus Unidentified Slots across the whole binder")
	assert.Equal(t, 1, got.UnidentifiedCount)
	require.Len(t, got.Slots, 4)
	assert.Equal(t, dto.BinderSlotUnidentified, got.Slots[0].State)
	assert.Equal(t, dto.BinderSlotEmpty, got.Slots[1].State)
}

func TestBinderPageService_Page_PageBeyondLastLeafIsNotFound(t *testing.T) {
	f := newBinderPageFixture(t)
	f.layouts.EXPECT().GetLayout(f.ctx, f.collectionID).Return(f.layout(2, 2), nil).Once()
	f.leaves.EXPECT().CountLeaves(f.ctx, f.collectionID).Return(1, nil).Once()
	f.leaves.EXPECT().GetLeafAt(f.ctx, f.collectionID, 2).Return(entity.BinderLeaf{}, ungerr.NotFoundError("no page")).Once()

	_, err := f.svc.Page(f.ctx, dto.BinderPageReadRequest{CollectionID: f.collectionID, Page: 3})
	requireStatus(t, err, http.StatusNotFound)
}

func TestBinderPageService_Page_PlainCollectionIsNotFoundBeforeAnyLeaf(t *testing.T) {
	f := newBinderPageFixture(t)
	f.layouts.EXPECT().GetLayout(f.ctx, f.collectionID).Return(entity.CollectionLayout{}, ungerr.NotFoundError("not a binder")).Once()

	_, err := f.svc.Page(f.ctx, dto.BinderPageReadRequest{CollectionID: f.collectionID, Page: 1})
	requireStatus(t, err, http.StatusNotFound)
}

// The header's Unidentified count ignores the visibility flag, so the header
// reads it with the flag off and never counts Cards of unpublished sets.
func TestBinderPageService_Header_CountsTheLayoutAndTheWholeBinderUnidentified(t *testing.T) {
	f := newBinderPageFixture(t)
	f.layouts.EXPECT().GetLayout(f.ctx, f.collectionID).Return(f.layout(2, 3), nil).Once()
	f.leaves.EXPECT().CountLeaves(f.ctx, f.collectionID).Return(2, nil).Once()
	f.slots.EXPECT().CountSlots(f.ctx, f.collectionID, false).Return(5, 2, nil).Once()

	got, err := f.svc.Header(f.ctx, dto.BinderReadRequest{CollectionID: f.collectionID})
	require.NoError(t, err)
	assert.Equal(t, dto.BinderHeader{
		Layout:            dto.BinderLayout{Rows: 2, Columns: 3, LeafCount: 2, Capacity: 24},
		UnidentifiedCount: 2,
	}, got)
}

func TestBinderPageService_Header_PlainCollectionIsNotFound(t *testing.T) {
	f := newBinderPageFixture(t)
	f.layouts.EXPECT().GetLayout(f.ctx, f.collectionID).Return(entity.CollectionLayout{}, ungerr.NotFoundError("not a binder")).Once()

	_, err := f.svc.Header(f.ctx, dto.BinderReadRequest{CollectionID: f.collectionID})
	requireStatus(t, err, http.StatusNotFound)
}
