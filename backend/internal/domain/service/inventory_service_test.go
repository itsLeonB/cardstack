package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type inventoryFixture struct {
	ctx         context.Context
	profileID   uuid.UUID
	collection  entity.Collection
	collections *mocks.MockCollectionRepository
	entries     *mocks.MockInventoryRepository
	cards       *mocks.MockRepository[entity.Card]
	svc         InventoryService
}

// newInventoryFixture's Transactor mock runs the transaction body inline.
// The owned Collection lookup is not expected here: tests set it (locked or
// not) themselves.
func newInventoryFixture(t *testing.T, limit int) inventoryFixture {
	t.Helper()
	f := inventoryFixture{ctx: context.Background(), profileID: uuid.New()}
	f.collection = entity.Collection{BaseEntity: baseEntity(uuid.New()), ProfileID: f.profileID, Title: "Binder", MaxCardCount: limit}
	f.collections = mocks.NewMockCollectionRepository(t)
	f.entries = mocks.NewMockInventoryRepository(t)
	f.cards = mocks.NewMockRepository[entity.Card](t)
	transactor := mocks.NewMockTransactor(t)
	transactor.EXPECT().WithinTransaction(f.ctx, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Maybe()
	f.svc = NewInventoryService(transactor, f.collections, f.entries, f.cards)
	return f
}

func (f inventoryFixture) lockedCollection() {
	f.collections.EXPECT().GetOwnedCollection(f.ctx, f.profileID, f.collection.ID, true).Return(f.collection, nil).Once()
}

func (f inventoryFixture) entryReq(cardID uuid.UUID, quantity int) dto.InventoryEntryRequest {
	return dto.InventoryEntryRequest{ProfileID: f.profileID, CollectionID: f.collection.ID, CardID: cardID, Quantity: quantity}
}

func lockedEntrySpec(collectionID, cardID uuid.UUID) crud.Specification[entity.InventoryEntry] {
	return crud.Specification[entity.InventoryEntry]{
		Model:     entity.InventoryEntry{CollectionID: collectionID, CardID: cardID},
		ForUpdate: true,
	}
}

func (f inventoryFixture) expectCard(cardID uuid.UUID, found bool) {
	got := entity.Card{}
	if found {
		got.BaseEntity = baseEntity(cardID)
	}
	f.cards.EXPECT().FindFirst(f.ctx, crud.Specification[entity.Card]{Model: entity.Card{BaseEntity: crud.BaseEntity{ID: cardID}}}).Return(got, nil).Once()
}

func (f inventoryFixture) expectEntry(cardID uuid.UUID, existing entity.InventoryEntry) {
	f.entries.EXPECT().FindFirst(f.ctx, lockedEntrySpec(f.collection.ID, cardID)).Return(existing, nil).Once()
}

func (f inventoryFixture) expectSum(sum int) {
	f.entries.EXPECT().SumQuantity(f.ctx, f.collection.ID).Return(sum, nil).Once()
}

func requireStatus(t *testing.T, err error, status int) {
	t.Helper()
	var appErr ungerr.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, status, appErr.HttpStatus())
}

func TestInventoryService_NotOwnedCollectionIsNotFound(t *testing.T) {
	f := newInventoryFixture(t, 0)
	cardID := uuid.New()
	notFound := ungerr.NotFoundError("collection not found")
	f.collections.EXPECT().GetOwnedCollection(f.ctx, f.profileID, f.collection.ID, false).Return(entity.Collection{}, notFound).Once()
	f.collections.EXPECT().GetOwnedCollection(f.ctx, f.profileID, f.collection.ID, true).Return(entity.Collection{}, notFound).Times(3)

	_, err := f.svc.List(f.ctx, dto.InventoryListRequest{ProfileID: f.profileID, CollectionID: f.collection.ID})
	requireStatus(t, err, http.StatusNotFound)
	_, err = f.svc.Add(f.ctx, f.entryReq(cardID, 1))
	requireStatus(t, err, http.StatusNotFound)
	_, err = f.svc.UpdateQuantity(f.ctx, f.entryReq(cardID, 1))
	requireStatus(t, err, http.StatusNotFound)
	requireStatus(t, f.svc.Remove(f.ctx, dto.InventoryEntryLookup{ProfileID: f.profileID, CollectionID: f.collection.ID, CardID: cardID}), http.StatusNotFound)
}

func TestInventoryService_List(t *testing.T) {
	f := newInventoryFixture(t, 0)
	early, late := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	mk := func(name string, release *time.Time, quantity int) entity.InventoryEntry {
		return entity.InventoryEntry{
			Quantity: quantity,
			Card:     entity.Card{BaseEntity: baseEntity(uuid.New()), Name: name, ExpansionSet: entity.ExpansionSet{ReleaseDate: release}},
		}
	}
	f.collections.EXPECT().GetOwnedCollection(f.ctx, f.profileID, f.collection.ID, false).Return(f.collection, nil).Once()
	f.entries.EXPECT().FindAll(f.ctx, crud.Specification[entity.InventoryEntry]{
		Model:            entity.InventoryEntry{CollectionID: f.collection.ID},
		PreloadRelations: []string{"Card.Rarity", "Card.ExpansionSet"},
	}).Return([]entity.InventoryEntry{mk("unknown", nil, 1), mk("late", &late, 2), mk("early", &early, 3)}, nil).Once()

	got, err := f.svc.List(f.ctx, dto.InventoryListRequest{ProfileID: f.profileID, CollectionID: f.collection.ID})
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, []string{"early", "late", "unknown"}, []string{got[0].Card.Name, got[1].Card.Name, got[2].Card.Name})
	assert.Equal(t, 3, got[0].Quantity)
}

func TestInventoryService_Add(t *testing.T) {
	f := newInventoryFixture(t, 0)
	cardID := uuid.New()
	f.lockedCollection()
	f.expectCard(cardID, true)
	f.expectEntry(cardID, entity.InventoryEntry{})
	want := entity.InventoryEntry{CollectionID: f.collection.ID, CardID: cardID, Quantity: 2}
	f.entries.EXPECT().Insert(f.ctx, want).Return(want, nil).Once()

	got, err := f.svc.Add(f.ctx, f.entryReq(cardID, 2))
	require.NoError(t, err)
	assert.Equal(t, dto.InventoryEntry{CardID: cardID, Quantity: 2}, got)
}

func TestInventoryService_Add_Rejections(t *testing.T) {
	cardID := uuid.New()

	t.Run("non-positive quantity", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		_, err := f.svc.Add(f.ctx, f.entryReq(cardID, 0))
		requireStatus(t, err, http.StatusBadRequest)
	})
	t.Run("nil card", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		f.lockedCollection()
		_, err := f.svc.Add(f.ctx, f.entryReq(uuid.Nil, 1))
		requireStatus(t, err, http.StatusNotFound)
	})
	t.Run("unknown card", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		f.lockedCollection()
		f.expectCard(cardID, false)
		_, err := f.svc.Add(f.ctx, f.entryReq(cardID, 1))
		requireStatus(t, err, http.StatusNotFound)
	})
	t.Run("card already present", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		f.lockedCollection()
		f.expectCard(cardID, true)
		f.expectEntry(cardID, entity.InventoryEntry{BaseEntity: baseEntity(uuid.New()), Quantity: 1})
		_, err := f.svc.Add(f.ctx, f.entryReq(cardID, 1))
		requireStatus(t, err, http.StatusConflict)
	})
	t.Run("over capacity", func(t *testing.T) {
		f := newInventoryFixture(t, 10)
		f.lockedCollection()
		f.expectCard(cardID, true)
		f.expectEntry(cardID, entity.InventoryEntry{})
		f.expectSum(8)
		_, err := f.svc.Add(f.ctx, f.entryReq(cardID, 3))
		requireStatus(t, err, http.StatusUnprocessableEntity)
	})
}

func TestInventoryService_Add_AllowsExactlyReachingCapacity(t *testing.T) {
	f := newInventoryFixture(t, 10)
	cardID := uuid.New()
	f.lockedCollection()
	f.expectCard(cardID, true)
	f.expectEntry(cardID, entity.InventoryEntry{})
	f.expectSum(8)
	want := entity.InventoryEntry{CollectionID: f.collection.ID, CardID: cardID, Quantity: 2}
	f.entries.EXPECT().Insert(f.ctx, want).Return(want, nil).Once()

	_, err := f.svc.Add(f.ctx, f.entryReq(cardID, 2))
	require.NoError(t, err)
}

func TestInventoryService_Add_PropagatesRepositoryError(t *testing.T) {
	f := newInventoryFixture(t, 0)
	wantErr := errors.New("boom")
	f.lockedCollection()
	f.cards.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entity.Card{}, wantErr).Once()

	_, err := f.svc.Add(f.ctx, f.entryReq(uuid.New(), 1))
	assert.ErrorIs(t, err, wantErr)
}

func TestInventoryService_UpdateQuantity(t *testing.T) {
	cardID := uuid.New()
	existing := entity.InventoryEntry{BaseEntity: baseEntity(uuid.New()), CollectionID: uuid.Nil, CardID: cardID, Quantity: 4}
	withQuantity := func(q int) entity.InventoryEntry { e := existing; e.Quantity = q; return e }

	t.Run("replaces current quantity in the capacity sum", func(t *testing.T) {
		f := newInventoryFixture(t, 10)
		f.lockedCollection()
		f.expectEntry(cardID, existing)
		f.expectSum(8) // 8 - 4 + 6 == limit
		f.entries.EXPECT().Update(f.ctx, withQuantity(6)).Return(withQuantity(6), nil).Once()

		got, err := f.svc.UpdateQuantity(f.ctx, f.entryReq(cardID, 6))
		require.NoError(t, err)
		assert.Equal(t, dto.InventoryEntry{CardID: cardID, Quantity: 6}, got)
	})
	t.Run("over capacity", func(t *testing.T) {
		f := newInventoryFixture(t, 10)
		f.lockedCollection()
		f.expectEntry(cardID, existing)
		f.expectSum(10)
		_, err := f.svc.UpdateQuantity(f.ctx, f.entryReq(cardID, 7))
		requireStatus(t, err, http.StatusUnprocessableEntity)
	})
	t.Run("already over a lowered limit: decrease and no-op allowed, increase rejected", func(t *testing.T) {
		f := newInventoryFixture(t, 5) // holds 12 > 5
		for range 3 {
			f.lockedCollection()
			f.expectEntry(cardID, existing)
		}
		f.entries.EXPECT().Update(f.ctx, withQuantity(2)).Return(withQuantity(2), nil).Once()
		f.entries.EXPECT().Update(f.ctx, withQuantity(4)).Return(withQuantity(4), nil).Once()
		f.expectSum(12)

		_, err := f.svc.UpdateQuantity(f.ctx, f.entryReq(cardID, 2))
		require.NoError(t, err)
		_, err = f.svc.UpdateQuantity(f.ctx, f.entryReq(cardID, 4))
		require.NoError(t, err)
		_, err = f.svc.UpdateQuantity(f.ctx, f.entryReq(cardID, 5))
		requireStatus(t, err, http.StatusUnprocessableEntity)
	})
	t.Run("missing entry", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		f.lockedCollection()
		f.expectEntry(cardID, entity.InventoryEntry{})
		_, err := f.svc.UpdateQuantity(f.ctx, f.entryReq(cardID, 1))
		requireStatus(t, err, http.StatusNotFound)
	})
	t.Run("nil card", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		f.lockedCollection()
		_, err := f.svc.UpdateQuantity(f.ctx, f.entryReq(uuid.Nil, 1))
		requireStatus(t, err, http.StatusNotFound)
	})
	t.Run("non-positive quantity", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		_, err := f.svc.UpdateQuantity(f.ctx, f.entryReq(cardID, 0))
		requireStatus(t, err, http.StatusBadRequest)
	})
}

func TestInventoryService_Remove(t *testing.T) {
	cardID := uuid.New()

	t.Run("deletes existing entry", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		found := entity.InventoryEntry{BaseEntity: baseEntity(uuid.New()), CardID: cardID}
		f.lockedCollection()
		f.expectEntry(cardID, found)
		f.entries.EXPECT().Delete(f.ctx, found).Return(nil).Once()
		require.NoError(t, f.svc.Remove(f.ctx, dto.InventoryEntryLookup{ProfileID: f.profileID, CollectionID: f.collection.ID, CardID: cardID}))
	})
	t.Run("missing entry", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		f.lockedCollection()
		f.expectEntry(cardID, entity.InventoryEntry{})
		requireStatus(t, f.svc.Remove(f.ctx, dto.InventoryEntryLookup{ProfileID: f.profileID, CollectionID: f.collection.ID, CardID: cardID}), http.StatusNotFound)
	})
}

func (f inventoryFixture) bulkReq(items ...dto.InventoryQuantityChange) dto.InventoryBulkUpdateRequest {
	return dto.InventoryBulkUpdateRequest{ProfileID: f.profileID, CollectionID: f.collection.ID, Items: items}
}

func change(cardID uuid.UUID, q int) dto.InventoryQuantityChange {
	return dto.InventoryQuantityChange{CardID: cardID, Quantity: q}
}

func TestInventoryService_BulkUpdate_MixedBatch(t *testing.T) {
	f := newInventoryFixture(t, 0)
	add, inc, dec, rm, gone := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	incE := entity.InventoryEntry{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collection.ID, CardID: inc, Quantity: 1}
	decE := entity.InventoryEntry{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collection.ID, CardID: dec, Quantity: 5}
	rmE := entity.InventoryEntry{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collection.ID, CardID: rm, Quantity: 2}
	f.lockedCollection()

	f.expectEntry(add, entity.InventoryEntry{})
	f.expectCard(add, true)
	created := entity.InventoryEntry{CollectionID: f.collection.ID, CardID: add, Quantity: 2}
	f.entries.EXPECT().Insert(f.ctx, created).Return(created, nil).Once()

	f.expectEntry(inc, incE)
	incE.Quantity = 4
	f.entries.EXPECT().Update(f.ctx, incE).Return(incE, nil).Once()

	f.expectEntry(dec, decE)
	decE.Quantity = 1
	f.entries.EXPECT().Update(f.ctx, decE).Return(decE, nil).Once()

	f.expectEntry(rm, rmE)
	f.entries.EXPECT().Delete(f.ctx, rmE).Return(nil).Once()

	f.expectEntry(gone, entity.InventoryEntry{})
	f.expectCard(gone, true)

	got, err := f.svc.BulkUpdate(f.ctx, f.bulkReq(change(add, 2), change(inc, 4), change(dec, 1), change(rm, 0), change(gone, 0)))
	require.NoError(t, err)
	assert.Equal(t, []dto.InventoryChangeResult{
		{CardID: add, Quantity: 2, Status: dto.InventoryStatusApplied},
		{CardID: inc, Quantity: 4, Status: dto.InventoryStatusApplied},
		{CardID: dec, Quantity: 1, Status: dto.InventoryStatusApplied},
		{CardID: rm, Quantity: 0, Status: dto.InventoryStatusRemoved},
		{CardID: gone, Quantity: 0, Status: dto.InventoryStatusRemoved},
	}, got)
}

func TestInventoryService_BulkUpdate_CapacityDeclinesLaterItems(t *testing.T) {
	f := newInventoryFixture(t, 10)
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	bE := entity.InventoryEntry{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collection.ID, CardID: b, Quantity: 2}
	f.lockedCollection()
	f.expectSum(6)

	// a: 6 -> 9 fits; b: 9 -> 9-2+5=12 declined, stays 2; c: 9 -> 10 fits again.
	f.expectEntry(a, entity.InventoryEntry{})
	f.expectCard(a, true)
	aNew := entity.InventoryEntry{CollectionID: f.collection.ID, CardID: a, Quantity: 3}
	f.entries.EXPECT().Insert(f.ctx, aNew).Return(aNew, nil).Once()
	f.expectEntry(b, bE)
	f.expectEntry(c, entity.InventoryEntry{})
	f.expectCard(c, true)
	cNew := entity.InventoryEntry{CollectionID: f.collection.ID, CardID: c, Quantity: 1}
	f.entries.EXPECT().Insert(f.ctx, cNew).Return(cNew, nil).Once()

	got, err := f.svc.BulkUpdate(f.ctx, f.bulkReq(change(a, 3), change(b, 5), change(c, 1)))
	require.NoError(t, err)
	assert.Equal(t, dto.InventoryChangeResult{CardID: a, Quantity: 3, Status: dto.InventoryStatusApplied}, got[0])
	assert.Equal(t, dto.InventoryChangeResult{CardID: b, Quantity: 2, Status: dto.InventoryStatusDeclined, Reason: dto.InventoryReasonCapacityExceeded, Message: capacityExceededMsg}, got[1])
	assert.Equal(t, dto.InventoryChangeResult{CardID: c, Quantity: 1, Status: dto.InventoryStatusApplied}, got[2])
}

func TestInventoryService_BulkUpdate_DecreaseOverLimitIsApplied(t *testing.T) {
	f := newInventoryFixture(t, 5)
	a, b := uuid.New(), uuid.New()
	aE := entity.InventoryEntry{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collection.ID, CardID: a, Quantity: 6}
	bE := entity.InventoryEntry{BaseEntity: baseEntity(uuid.New()), CollectionID: f.collection.ID, CardID: b, Quantity: 4}
	f.lockedCollection()
	f.expectSum(10) // over the limit of 5
	f.expectEntry(a, aE)
	aE.Quantity = 3
	f.entries.EXPECT().Update(f.ctx, aE).Return(aE, nil).Once()
	f.expectEntry(b, bE) // keeping the quantity is never declined, and writes nothing

	got, err := f.svc.BulkUpdate(f.ctx, f.bulkReq(change(a, 3), change(b, 4)))
	require.NoError(t, err)
	assert.Equal(t, []dto.InventoryChangeResult{
		{CardID: a, Quantity: 3, Status: dto.InventoryStatusApplied},
		{CardID: b, Quantity: 4, Status: dto.InventoryStatusApplied},
	}, got)
}

func TestInventoryService_BulkUpdate_UnknownCardDeclinedRestApplied(t *testing.T) {
	f := newInventoryFixture(t, 0)
	unknown, ok := uuid.New(), uuid.New()
	f.lockedCollection()
	f.expectEntry(unknown, entity.InventoryEntry{})
	f.expectCard(unknown, false)
	f.expectEntry(ok, entity.InventoryEntry{})
	f.expectCard(ok, true)
	created := entity.InventoryEntry{CollectionID: f.collection.ID, CardID: ok, Quantity: 1}
	f.entries.EXPECT().Insert(f.ctx, created).Return(created, nil).Once()

	got, err := f.svc.BulkUpdate(f.ctx, f.bulkReq(change(unknown, 2), change(ok, 1)))
	require.NoError(t, err)
	assert.Equal(t, dto.InventoryChangeResult{CardID: unknown, Status: dto.InventoryStatusDeclined, Reason: dto.InventoryReasonCardNotFound, Message: cardNotFoundMsg}, got[0])
	assert.Equal(t, dto.InventoryChangeResult{CardID: ok, Quantity: 1, Status: dto.InventoryStatusApplied}, got[1])
}

func TestInventoryService_BulkUpdate_NilCardIsNotFound(t *testing.T) {
	f := newInventoryFixture(t, 0)
	f.lockedCollection()

	got, err := f.svc.BulkUpdate(f.ctx, f.bulkReq(change(uuid.Nil, 1)))
	require.NoError(t, err)
	assert.Equal(t, dto.InventoryReasonCardNotFound, got[0].Reason)
}

func TestInventoryService_BulkUpdate_Rejections(t *testing.T) {
	cardID := uuid.New()

	t.Run("duplicate card", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		_, err := f.svc.BulkUpdate(f.ctx, f.bulkReq(change(cardID, 1), change(cardID, 2)))
		requireStatus(t, err, http.StatusBadRequest)
	})
	t.Run("not owned collection", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		f.collections.EXPECT().GetOwnedCollection(f.ctx, f.profileID, f.collection.ID, true).
			Return(entity.Collection{}, ungerr.NotFoundError("collection not found")).Once()
		_, err := f.svc.BulkUpdate(f.ctx, f.bulkReq(change(cardID, 1)))
		requireStatus(t, err, http.StatusNotFound)
	})
	t.Run("repository error aborts the batch", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		wantErr := errors.New("boom")
		f.lockedCollection()
		f.entries.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entity.InventoryEntry{}, wantErr).Once()
		_, err := f.svc.BulkUpdate(f.ctx, f.bulkReq(change(cardID, 1)))
		assert.ErrorIs(t, err, wantErr)
	})
}
