package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/apperr"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	"github.com/itsLeonB/ungerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type binderFixture struct {
	ctx         context.Context
	profileID   uuid.UUID
	collection  entity.Collection
	collections *mocks.MockCollectionRepository
	layouts     *mocks.MockBinderLayoutRepository
	leaves      *mocks.MockBinderLeafRepository
	slots       *mocks.MockBinderSlotRepository
	pages       *mocks.MockBinderPageService
	cards       *mocks.MockCardRepository
	entries     *mocks.MockRepository[entity.InventoryEntry]
	svc         BinderService
	slot        BinderSlotService
}

// newBinderFixture's Transactor mock runs the transaction body inline. Tests
// set the owned Collection lookup and any repository calls they expect.
func newBinderFixture(t *testing.T) binderFixture {
	t.Helper()
	f := binderFixture{ctx: context.Background(), profileID: uuid.New()}
	f.collection = entity.Collection{BaseEntity: baseEntity(uuid.New()), ProfileID: f.profileID, Title: "Binder"}
	f.collections = mocks.NewMockCollectionRepository(t)
	f.layouts = mocks.NewMockBinderLayoutRepository(t)
	f.leaves = mocks.NewMockBinderLeafRepository(t)
	f.slots = mocks.NewMockBinderSlotRepository(t)
	f.pages = mocks.NewMockBinderPageService(t)
	f.cards = mocks.NewMockCardRepository(t)
	f.entries = mocks.NewMockRepository[entity.InventoryEntry](t)
	transactor := mocks.NewMockTransactor(t)
	transactor.EXPECT().WithinTransaction(f.ctx, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Maybe()
	f.svc = NewBinderService(transactor, f.collections, f.layouts, f.leaves, f.slots, f.entries, f.pages)
	f.slot = NewBinderSlotService(transactor, f.collections, f.layouts, f.leaves, f.slots, f.entries, NewCardLookup(f.cards))
	return f
}

// lockedCollection expects the owned, row-locked lookup that every write starts with.
func (f binderFixture) lockedCollection() {
	f.collections.EXPECT().GetOwnedCollection(f.ctx, f.profileID, f.collection.ID, true).Return(f.collection, nil).Once()
}

func (f binderFixture) layout(rows, columns int) entity.CollectionLayout {
	return entity.CollectionLayout{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collection.ID, Rows: rows, Columns: columns}
}

func TestBinderService_CreateBinder_RejectsGridOutOfRange(t *testing.T) {
	f := newBinderFixture(t)

	for _, req := range []dto.BinderLayoutRequest{
		{Rows: 0, Columns: 3, LeafCount: 10},
		{Rows: 3, Columns: 7, LeafCount: 10},
	} {
		_, err := f.svc.CreateBinder(f.ctx, req)
		requireStatus(t, err, http.StatusBadRequest)
	}
}

func TestBinderService_CreateBinder_RejectsLeafCountOutOfRange(t *testing.T) {
	f := newBinderFixture(t)

	for _, leaves := range []int{0, 101} {
		_, err := f.svc.CreateBinder(f.ctx, dto.BinderLayoutRequest{Rows: 2, Columns: 2, LeafCount: leaves})
		requireStatus(t, err, http.StatusBadRequest)
	}
}

func TestBinderService_CreateBinder_RefusesCollectionHoldingEntries(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().IsBinder(f.ctx, f.collection.ID).Return(false, nil).Once()
	f.entries.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entity.InventoryEntry{BaseEntity: baseEntity(uuid.New())}, nil).Once()

	_, err := f.svc.CreateBinder(f.ctx, dto.BinderLayoutRequest{ProfileID: f.profileID, CollectionID: f.collection.ID, Rows: 2, Columns: 2, LeafCount: 10})
	requireStatus(t, err, http.StatusConflict)
	assert.Equal(t, apperr.CodeCollectionNotEmpty, apperr.CodeOf(err))
}

func TestBinderService_ChangeGrid_RefusesLayoutWithOccupiedSlots(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(2, 2), nil).Once()
	f.slots.EXPECT().CountSlots(f.ctx, f.collection.ID, true).Return(1, 0, nil).Once()

	_, err := f.svc.ChangeGrid(f.ctx, dto.BinderGridRequest{ProfileID: f.profileID, CollectionID: f.collection.ID, Rows: 3, Columns: 3})
	requireStatus(t, err, http.StatusConflict)
	assert.Equal(t, apperr.CodeBinderLayoutLocked, apperr.CodeOf(err))
}

func TestBinderService_AddLeaf_PlacesBeforeOrAfter(t *testing.T) {
	const leafCount = 3
	for _, tt := range []struct {
		name      string
		placement string
		position  int
	}{
		{name: "before leaf 2 takes its number", placement: dto.BinderPlacementBefore, position: 2},
		{name: "after leaf 2 takes the next", placement: dto.BinderPlacementAfter, position: 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newBinderFixture(t)
			f.lockedCollection()
			f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(2, 2), nil).Once()
			f.leaves.EXPECT().CountLeaves(f.ctx, f.collection.ID).Return(leafCount, nil).Once()
			f.leaves.EXPECT().ShiftLeaves(f.ctx, f.collection.ID, tt.position, 1).Return(nil).Once()
			f.leaves.EXPECT().InsertMany(f.ctx, []entity.BinderLeaf{{CollectionID: f.collection.ID, Position: tt.position}}).
				Return(nil, nil).Once()

			got, err := f.svc.AddLeaf(f.ctx, dto.BinderLeafRequest{ProfileID: f.profileID, CollectionID: f.collection.ID, Leaf: 2, Placement: tt.placement})
			require.NoError(t, err)
			assert.Equal(t, leafCount+1, got.LeafCount)
		})
	}
}

func TestBinderService_AddLeaf_ValidatesLeafNumberBeforeLimit(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(2, 2), nil).Times(2)
	f.leaves.EXPECT().CountLeaves(f.ctx, f.collection.ID).Return(maxLeaves, nil).Times(2)

	// A full binder with a missing Leaf reports the Leaf, not the cap.
	_, err := f.svc.AddLeaf(f.ctx, dto.BinderLeafRequest{ProfileID: f.profileID, CollectionID: f.collection.ID, Leaf: maxLeaves + 1, Placement: dto.BinderPlacementAfter})
	requireStatus(t, err, http.StatusNotFound)

	f.lockedCollection()
	_, err = f.svc.AddLeaf(f.ctx, dto.BinderLeafRequest{ProfileID: f.profileID, CollectionID: f.collection.ID, Leaf: 1, Placement: dto.BinderPlacementAfter})
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderService_AddLeaf_RejectsInvalidPlacement(t *testing.T) {
	f := newBinderFixture(t)

	_, err := f.svc.AddLeaf(f.ctx, dto.BinderLeafRequest{ProfileID: f.profileID, CollectionID: f.collection.ID, Leaf: 1, Placement: "sideways"})
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderService_RemoveLeaf_RefusesLastLeaf(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(2, 2), nil).Once()
	f.leaves.EXPECT().FindLeafAt(f.ctx, f.collection.ID, 1).Return(entity.BinderLeaf{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collection.ID, Position: 1}, nil).Once()
	f.leaves.EXPECT().CountLeaves(f.ctx, f.collection.ID).Return(1, nil).Once()

	_, err := f.svc.RemoveLeaf(f.ctx, dto.BinderLeafLookup{ProfileID: f.profileID, CollectionID: f.collection.ID, Leaf: 1})
	requireStatus(t, err, http.StatusBadRequest)
}

func TestBinderService_RemoveLeaf_RefusesNamingTheOccupiedPage(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(2, 2), nil).Once()
	leaf := entity.BinderLeaf{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collection.ID, Position: 2}
	f.leaves.EXPECT().FindLeafAt(f.ctx, f.collection.ID, 2).Return(leaf, nil).Once()
	f.leaves.EXPECT().CountLeaves(f.ctx, f.collection.ID).Return(3, nil).Once()
	f.slots.EXPECT().HasSlotsOnPage(f.ctx, leaf.ID, dto.BinderSideFront).Return(false, nil).Once()
	f.slots.EXPECT().HasSlotsOnPage(f.ctx, leaf.ID, dto.BinderSideBack).Return(true, nil).Once()

	_, err := f.svc.RemoveLeaf(f.ctx, dto.BinderLeafLookup{ProfileID: f.profileID, CollectionID: f.collection.ID, Leaf: 2})
	requireStatus(t, err, http.StatusConflict)
	assert.Equal(t, apperr.CodeBinderLeafInUse, apperr.CodeOf(err))
	var appErr ungerr.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Contains(t, appErr.Details(), "page 4")
}

func TestBinderService_RemoveLeaf_NotFound(t *testing.T) {
	f := newBinderFixture(t)
	f.lockedCollection()
	f.layouts.EXPECT().GetLayout(f.ctx, f.collection.ID).Return(f.layout(2, 2), nil).Once()
	f.leaves.EXPECT().FindLeafAt(f.ctx, f.collection.ID, 9).Return(entity.BinderLeaf{}, nil).Once()

	_, err := f.svc.RemoveLeaf(f.ctx, dto.BinderLeafLookup{ProfileID: f.profileID, CollectionID: f.collection.ID, Leaf: 9})
	requireStatus(t, err, http.StatusNotFound)
}

func TestBinderService_GetPage_ReadsTheOwnersPageWithUnpublishedCards(t *testing.T) {
	f := newBinderFixture(t)
	f.collections.EXPECT().GetOwnedCollection(f.ctx, f.profileID, f.collection.ID, false).Return(f.collection, nil).Once()
	want := dto.BinderPage{Page: 4, Leaf: 2, Side: dto.BinderSideBack}
	f.pages.EXPECT().Page(f.ctx, dto.BinderPageReadRequest{CollectionID: f.collection.ID, Page: 4, IncludeUnpublished: true}).Return(want, nil).Once()

	got, err := f.svc.GetPage(f.ctx, dto.BinderPageLookup{ProfileID: f.profileID, CollectionID: f.collection.ID, Page: 4})
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestBinderService_GetPage_PassesThroughTheNotFoundForAPageBeyondLastLeaf(t *testing.T) {
	f := newBinderFixture(t)
	f.collections.EXPECT().GetOwnedCollection(f.ctx, f.profileID, f.collection.ID, false).Return(f.collection, nil).Once()
	f.pages.EXPECT().Page(f.ctx, dto.BinderPageReadRequest{CollectionID: f.collection.ID, Page: 3, IncludeUnpublished: true}).
		Return(dto.BinderPage{}, ungerr.NotFoundError("no page")).Once()

	_, err := f.svc.GetPage(f.ctx, dto.BinderPageLookup{ProfileID: f.profileID, CollectionID: f.collection.ID, Page: 3})
	requireStatus(t, err, http.StatusNotFound)
}
