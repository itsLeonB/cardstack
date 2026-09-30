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
	collections *mocks.MockRepository[entity.Collection]
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
	f.collections = mocks.NewMockRepository[entity.Collection](t)
	f.entries = mocks.NewMockInventoryRepository(t)
	f.cards = mocks.NewMockRepository[entity.Card](t)
	transactor := mocks.NewMockTransactor(t)
	transactor.EXPECT().WithinTransaction(f.ctx, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Maybe()
	f.svc = NewInventoryService(transactor, f.collections, f.entries, f.cards)
	return f
}

func (f inventoryFixture) lockedCollection() {
	spec := ownedSpec(f.profileID, f.collection.ID)
	spec.ForUpdate = true
	f.collections.EXPECT().FindFirst(f.ctx, spec).Return(f.collection, nil).Once()
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
	lockedSpec := ownedSpec(f.profileID, f.collection.ID)
	lockedSpec.ForUpdate = true
	f.collections.EXPECT().FindFirst(f.ctx, ownedSpec(f.profileID, f.collection.ID)).Return(entity.Collection{}, nil).Once()
	f.collections.EXPECT().FindFirst(f.ctx, lockedSpec).Return(entity.Collection{}, nil).Times(3)

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
	f.collections.EXPECT().FindFirst(f.ctx, ownedSpec(f.profileID, f.collection.ID)).Return(f.collection, nil).Once()
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

func TestInventoryService_NilProfileIsNotFound(t *testing.T) {
	f := newInventoryFixture(t, 0) // no repository call is expected
	req := f.entryReq(uuid.New(), 1)
	req.ProfileID = uuid.Nil

	_, err := f.svc.List(f.ctx, dto.InventoryListRequest{CollectionID: f.collection.ID})
	requireStatus(t, err, http.StatusNotFound)
	_, err = f.svc.Add(f.ctx, req)
	requireStatus(t, err, http.StatusNotFound)
	_, err = f.svc.UpdateQuantity(f.ctx, req)
	requireStatus(t, err, http.StatusNotFound)
	requireStatus(t, f.svc.Remove(f.ctx, dto.InventoryEntryLookup{CollectionID: f.collection.ID, CardID: req.CardID}), http.StatusNotFound)
}
