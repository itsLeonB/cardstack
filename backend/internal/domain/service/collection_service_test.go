package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

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

func TestCollectionService_Create(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	limit := 100

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().
		Insert(ctx, entity.Collection{ProfileID: profileID, Title: "Binder", Description: "desc", MaxCardCount: limit}).
		Return(entity.Collection{BaseEntity: baseEntity(uuid.New()), ProfileID: profileID, Title: "Binder", Description: "desc", MaxCardCount: limit}, nil).
		Once()

	got, err := NewCollectionService(repo).Create(ctx, dto.CollectionRequest{ProfileID: profileID, Title: "Binder", Description: "desc", MaxCardCount: limit})
	require.NoError(t, err)
	assert.Equal(t, "Binder", got.Title)
	assert.Equal(t, "desc", got.Description)
	assert.Equal(t, limit, got.MaxCardCount)
	assert.Zero(t, got.CardCount)
}

func TestCollectionService_Create_PropagatesRepositoryError(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	wantErr := errors.New("boom")

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().Insert(ctx, entity.Collection{ProfileID: profileID, Title: "Binder"}).Return(entity.Collection{}, wantErr).Once()

	_, err := NewCollectionService(repo).Create(ctx, dto.CollectionRequest{ProfileID: profileID, Title: "Binder"})
	assert.ErrorIs(t, err, wantErr)
}

func TestCollectionService_List(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	emptyID := uuid.New()

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().
		FindAll(ctx, crud.Specification[entity.Collection]{Model: entity.Collection{ProfileID: profileID}}).
		Return([]entity.Collection{
			{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "Binder"},
			{BaseEntity: baseEntity(emptyID), ProfileID: profileID, Title: "Empty"},
		}, nil).
		Once()
	// One aggregate call for the whole list; an empty Collection is absent from the map.
	repo.EXPECT().SumQuantities(ctx, []uuid.UUID{id, emptyID}).Return(map[uuid.UUID]int{id: 37}, nil).Once()

	got, err := NewCollectionService(repo).List(ctx, dto.CollectionListRequest{ProfileID: profileID})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, id, got[0].ID)
	assert.Equal(t, "Binder", got[0].Title)
	assert.Equal(t, 37, got[0].CardCount)
	assert.Zero(t, got[1].CardCount)
}

func TestCollectionService_List_PropagatesSumError(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	wantErr := errors.New("boom")

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().FindAll(ctx, crud.Specification[entity.Collection]{Model: entity.Collection{ProfileID: profileID}}).
		Return([]entity.Collection{{BaseEntity: baseEntity(uuid.New()), ProfileID: profileID}}, nil).Once()
	repo.EXPECT().SumQuantities(ctx, mock.Anything).Return(nil, wantErr).Once()

	_, err := NewCollectionService(repo).List(ctx, dto.CollectionListRequest{ProfileID: profileID})
	assert.ErrorIs(t, err, wantErr)
}

func TestCollectionService_Get_ReturnsOwnedCollection(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().GetOwnedCollection(ctx, profileID, id, false).Return(entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "Binder"}, nil).Once()
	repo.EXPECT().SumQuantities(ctx, []uuid.UUID{id}).Return(map[uuid.UUID]int{id: 12}, nil).Once()

	got, err := NewCollectionService(repo).Get(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id})
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, 12, got.CardCount)
}

// TestCollectionService_NotFound: the repository's not-found AppError (missing
// or another profile's Collection, nil ids) is returned unchanged, and no
// write follows it.
func TestCollectionService_NotFound(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().GetOwnedCollection(ctx, profileID, id, false).Return(entity.Collection{}, ungerr.NotFoundError("collection not found")).Times(3)
	svc := NewCollectionService(repo)

	_, err := svc.Get(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id})
	assertNotFound(t, err)

	_, err = svc.Update(ctx, dto.CollectionRequest{ProfileID: profileID, ID: id, Title: "Hijacked"})
	assertNotFound(t, err)

	assertNotFound(t, svc.Delete(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id}))
}

func TestCollectionService_Get_PropagatesRepositoryError(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	wantErr := errors.New("boom")

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().GetOwnedCollection(ctx, profileID, id, false).Return(entity.Collection{}, wantErr).Once()

	_, err := NewCollectionService(repo).Get(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id})
	assert.ErrorIs(t, err, wantErr)
}

func TestCollectionService_Update(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	limit := 50

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().GetOwnedCollection(ctx, profileID, id, false).Return(entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "Old", Description: "old"}, nil).Once()
	repo.EXPECT().
		Update(ctx, entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "New", Description: "new", MaxCardCount: limit}).
		Return(entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "New", Description: "new", MaxCardCount: limit}, nil).
		Once()
	repo.EXPECT().SumQuantities(ctx, []uuid.UUID{id}).Return(map[uuid.UUID]int{id: 7}, nil).Once()

	got, err := NewCollectionService(repo).Update(ctx, dto.CollectionRequest{ProfileID: profileID, ID: id, Title: "New", Description: "new", MaxCardCount: limit})
	require.NoError(t, err)
	assert.Equal(t, "New", got.Title)
	assert.Equal(t, "new", got.Description)
	assert.Equal(t, limit, got.MaxCardCount)
	assert.Equal(t, 7, got.CardCount)
}

func TestCollectionService_Delete(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	found := entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "Binder"}

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().GetOwnedCollection(ctx, profileID, id, false).Return(found, nil).Once()
	repo.EXPECT().Delete(ctx, found).Return(nil).Once()

	assert.NoError(t, NewCollectionService(repo).Delete(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id}))
}

func assertNotFound(t *testing.T, err error) {
	t.Helper()
	var appErr ungerr.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, http.StatusNotFound, appErr.HttpStatus())
}

// A nil ProfileID must not list everyone's collections; the lookups' nil
// guard lives in CollectionRepository.GetOwnedCollection.
func TestCollectionService_List_NilProfile(t *testing.T) {
	svc := NewCollectionService(mocks.NewMockCollectionRepository(t)) // no repository call is expected

	got, err := svc.List(context.Background(), dto.CollectionListRequest{})
	require.NoError(t, err)
	assert.Empty(t, got)
}
