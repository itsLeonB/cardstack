package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
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
	inventory   *mocks.MockInventoryRepository
	svc         InventoryService
}

func newInventoryFixture(t *testing.T, limit int) inventoryFixture {
	t.Helper()
	f := inventoryFixture{ctx: context.Background(), profileID: uuid.New()}
	f.collection = entity.Collection{BaseEntity: baseEntity(uuid.New()), ProfileID: f.profileID, Title: "Binder", MaxCardCount: limit}
	f.collections = mocks.NewMockRepository[entity.Collection](t)
	f.inventory = mocks.NewMockInventoryRepository(t)
	f.svc = NewInventoryService(f.collections, f.inventory)
	f.collections.EXPECT().FindFirst(f.ctx, ownedSpec(f.profileID, f.collection.ID)).Return(f.collection, nil).Maybe()
	return f
}

func requireStatus(t *testing.T, err error, status int) {
	t.Helper()
	var appErr ungerr.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, status, appErr.HttpStatus())
}

func TestInventoryService_NotOwnedCollectionIsNotFound(t *testing.T) {
	ctx := context.Background()
	profileID, id, cardID := uuid.New(), uuid.New(), uuid.New()
	collections := mocks.NewMockRepository[entity.Collection](t)
	collections.EXPECT().FindFirst(ctx, ownedSpec(profileID, id)).Return(entity.Collection{}, nil)
	svc := NewInventoryService(collections, mocks.NewMockInventoryRepository(t))

	_, err := svc.List(ctx, profileID, id)
	requireStatus(t, err, http.StatusNotFound)
	_, err = svc.Add(ctx, profileID, id, dto.InventoryEntryRequest{CardID: cardID, Quantity: 1})
	requireStatus(t, err, http.StatusNotFound)
	_, err = svc.UpdateQuantity(ctx, profileID, id, cardID, 1)
	requireStatus(t, err, http.StatusNotFound)
	requireStatus(t, svc.Remove(ctx, profileID, id, cardID), http.StatusNotFound)
}

func TestInventoryService_List(t *testing.T) {
	f := newInventoryFixture(t, 0)
	cardID := uuid.New()
	f.inventory.EXPECT().ListItems(f.ctx, f.collection.ID).
		Return([]repository.InventoryItemResult{{CardResult: repository.CardResult{ID: cardID, Name: "Pikachu"}, Quantity: 3}}, nil).Once()

	got, err := f.svc.List(f.ctx, f.profileID, f.collection.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, cardID, got[0].Card.ID)
	assert.Equal(t, "Pikachu", got[0].Card.Name)
	assert.Equal(t, 3, got[0].Quantity)
}

func TestInventoryService_Add(t *testing.T) {
	f := newInventoryFixture(t, 0)
	cardID := uuid.New()
	want := entity.InventoryEntry{CollectionID: f.collection.ID, CardID: cardID, Quantity: 2}
	f.inventory.EXPECT().CardExists(f.ctx, cardID).Return(true, nil).Once()
	f.inventory.EXPECT().FindEntry(f.ctx, f.collection.ID, cardID).Return(entity.InventoryEntry{}, nil).Once()
	f.inventory.EXPECT().SaveEntry(f.ctx, want).Return(want, nil).Once()

	got, err := f.svc.Add(f.ctx, f.profileID, f.collection.ID, dto.InventoryEntryRequest{CardID: cardID, Quantity: 2})
	require.NoError(t, err)
	assert.Equal(t, dto.InventoryEntry{CardID: cardID, Quantity: 2}, got)
}

func TestInventoryService_Add_Rejections(t *testing.T) {
	cardID := uuid.New()
	existing := entity.InventoryEntry{BaseEntity: baseEntity(uuid.New()), Quantity: 1}

	t.Run("non-positive quantity", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		_, err := f.svc.Add(f.ctx, f.profileID, f.collection.ID, dto.InventoryEntryRequest{CardID: cardID})
		requireStatus(t, err, http.StatusBadRequest)
	})
	t.Run("unknown card", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		f.inventory.EXPECT().CardExists(f.ctx, cardID).Return(false, nil).Once()
		_, err := f.svc.Add(f.ctx, f.profileID, f.collection.ID, dto.InventoryEntryRequest{CardID: cardID, Quantity: 1})
		requireStatus(t, err, http.StatusNotFound)
	})
	t.Run("card already present", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		f.inventory.EXPECT().CardExists(f.ctx, cardID).Return(true, nil).Once()
		f.inventory.EXPECT().FindEntry(f.ctx, f.collection.ID, cardID).Return(existing, nil).Once()
		_, err := f.svc.Add(f.ctx, f.profileID, f.collection.ID, dto.InventoryEntryRequest{CardID: cardID, Quantity: 1})
		requireStatus(t, err, http.StatusConflict)
	})
	t.Run("over capacity", func(t *testing.T) {
		f := newInventoryFixture(t, 10)
		f.inventory.EXPECT().CardExists(f.ctx, cardID).Return(true, nil).Once()
		f.inventory.EXPECT().FindEntry(f.ctx, f.collection.ID, cardID).Return(entity.InventoryEntry{}, nil).Once()
		f.inventory.EXPECT().SumQuantity(f.ctx, f.collection.ID).Return(8, nil).Once()
		_, err := f.svc.Add(f.ctx, f.profileID, f.collection.ID, dto.InventoryEntryRequest{CardID: cardID, Quantity: 3})
		requireStatus(t, err, http.StatusUnprocessableEntity)
	})
}

func TestInventoryService_Add_AllowsExactlyReachingCapacity(t *testing.T) {
	f := newInventoryFixture(t, 10)
	cardID := uuid.New()
	want := entity.InventoryEntry{CollectionID: f.collection.ID, CardID: cardID, Quantity: 2}
	f.inventory.EXPECT().CardExists(f.ctx, cardID).Return(true, nil).Once()
	f.inventory.EXPECT().FindEntry(f.ctx, f.collection.ID, cardID).Return(entity.InventoryEntry{}, nil).Once()
	f.inventory.EXPECT().SumQuantity(f.ctx, f.collection.ID).Return(8, nil).Once()
	f.inventory.EXPECT().SaveEntry(f.ctx, want).Return(want, nil).Once()

	_, err := f.svc.Add(f.ctx, f.profileID, f.collection.ID, dto.InventoryEntryRequest{CardID: cardID, Quantity: 2})
	require.NoError(t, err)
}

func TestInventoryService_Add_PropagatesRepositoryError(t *testing.T) {
	f := newInventoryFixture(t, 0)
	wantErr := errors.New("boom")
	f.inventory.EXPECT().CardExists(f.ctx, mock.Anything).Return(false, wantErr).Once()

	_, err := f.svc.Add(f.ctx, f.profileID, f.collection.ID, dto.InventoryEntryRequest{CardID: uuid.New(), Quantity: 1})
	assert.ErrorIs(t, err, wantErr)
}

func TestInventoryService_UpdateQuantity(t *testing.T) {
	cardID := uuid.New()
	existing := entity.InventoryEntry{BaseEntity: baseEntity(uuid.New()), CardID: cardID, Quantity: 4}

	t.Run("replaces current quantity in the capacity sum", func(t *testing.T) {
		f := newInventoryFixture(t, 10)
		existing := existing
		existing.CollectionID = f.collection.ID
		updated := existing
		updated.Quantity = 6
		f.inventory.EXPECT().FindEntry(f.ctx, f.collection.ID, cardID).Return(existing, nil).Once()
		f.inventory.EXPECT().SumQuantity(f.ctx, f.collection.ID).Return(8, nil).Once() // 8 - 4 + 6 == limit
		f.inventory.EXPECT().SaveEntry(f.ctx, updated).Return(updated, nil).Once()

		got, err := f.svc.UpdateQuantity(f.ctx, f.profileID, f.collection.ID, cardID, 6)
		require.NoError(t, err)
		assert.Equal(t, dto.InventoryEntry{CardID: cardID, Quantity: 6}, got)
	})
	t.Run("over capacity", func(t *testing.T) {
		f := newInventoryFixture(t, 10)
		f.inventory.EXPECT().FindEntry(f.ctx, f.collection.ID, cardID).Return(existing, nil).Once()
		f.inventory.EXPECT().SumQuantity(f.ctx, f.collection.ID).Return(10, nil).Once()
		_, err := f.svc.UpdateQuantity(f.ctx, f.profileID, f.collection.ID, cardID, 7)
		requireStatus(t, err, http.StatusUnprocessableEntity)
	})
	t.Run("missing entry", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		f.inventory.EXPECT().FindEntry(f.ctx, f.collection.ID, cardID).Return(entity.InventoryEntry{}, nil).Once()
		_, err := f.svc.UpdateQuantity(f.ctx, f.profileID, f.collection.ID, cardID, 1)
		requireStatus(t, err, http.StatusNotFound)
	})
	t.Run("non-positive quantity", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		_, err := f.svc.UpdateQuantity(f.ctx, f.profileID, f.collection.ID, cardID, 0)
		requireStatus(t, err, http.StatusBadRequest)
	})
}

func TestInventoryService_Remove(t *testing.T) {
	cardID := uuid.New()

	t.Run("deletes existing entry", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		f.inventory.EXPECT().FindEntry(f.ctx, f.collection.ID, cardID).Return(entity.InventoryEntry{BaseEntity: baseEntity(uuid.New())}, nil).Once()
		f.inventory.EXPECT().DeleteEntry(f.ctx, f.collection.ID, cardID).Return(nil).Once()
		require.NoError(t, f.svc.Remove(f.ctx, f.profileID, f.collection.ID, cardID))
	})
	t.Run("missing entry", func(t *testing.T) {
		f := newInventoryFixture(t, 0)
		f.inventory.EXPECT().FindEntry(f.ctx, f.collection.ID, cardID).Return(entity.InventoryEntry{}, nil).Once()
		requireStatus(t, f.svc.Remove(f.ctx, f.profileID, f.collection.ID, cardID), http.StatusNotFound)
	})
}
