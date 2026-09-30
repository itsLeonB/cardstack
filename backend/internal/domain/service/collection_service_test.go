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
	"github.com/stretchr/testify/require"
)

func ownedSpec(profileID, id uuid.UUID) crud.Specification[entity.Collection] {
	return crud.Specification[entity.Collection]{
		Model: entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID},
	}
}

func TestCollectionService_Create(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	limit := 100

	repo := mocks.NewMockRepository[entity.Collection](t)
	repo.EXPECT().
		Insert(ctx, entity.Collection{ProfileID: profileID, Title: "Binder", Description: "desc", MaxCardCount: limit}).
		Return(entity.Collection{BaseEntity: baseEntity(uuid.New()), ProfileID: profileID, Title: "Binder", Description: "desc", MaxCardCount: limit}, nil).
		Once()

	got, err := NewCollectionService(repo).Create(ctx, dto.CollectionRequest{ProfileID: profileID, Title: "Binder", Description: "desc", MaxCardCount: limit})
	require.NoError(t, err)
	assert.Equal(t, "Binder", got.Title)
	assert.Equal(t, "desc", got.Description)
	assert.Equal(t, limit, got.MaxCardCount)
}

func TestCollectionService_Create_PropagatesRepositoryError(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	wantErr := errors.New("boom")

	repo := mocks.NewMockRepository[entity.Collection](t)
	repo.EXPECT().Insert(ctx, entity.Collection{ProfileID: profileID, Title: "Binder"}).Return(entity.Collection{}, wantErr).Once()

	_, err := NewCollectionService(repo).Create(ctx, dto.CollectionRequest{ProfileID: profileID, Title: "Binder"})
	assert.ErrorIs(t, err, wantErr)
}

func TestCollectionService_List(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()

	repo := mocks.NewMockRepository[entity.Collection](t)
	repo.EXPECT().
		FindAll(ctx, crud.Specification[entity.Collection]{Model: entity.Collection{ProfileID: profileID}}).
		Return([]entity.Collection{{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "Binder"}}, nil).
		Once()

	got, err := NewCollectionService(repo).List(ctx, dto.CollectionListRequest{ProfileID: profileID})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, id, got[0].ID)
	assert.Equal(t, "Binder", got[0].Title)
}

func TestCollectionService_Get_ReturnsOwnedCollection(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()

	repo := mocks.NewMockRepository[entity.Collection](t)
	repo.EXPECT().FindFirst(ctx, ownedSpec(profileID, id)).Return(entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "Binder"}, nil).Once()

	got, err := NewCollectionService(repo).Get(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id})
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

// TestCollectionService_NotFound covers both a missing Collection and
// another profile's: the owner-scoped query matches nothing either way, so
// FindFirst returns a zero value and existence isn't leaked.
func TestCollectionService_NotFound(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()

	repo := mocks.NewMockRepository[entity.Collection](t)
	repo.EXPECT().FindFirst(ctx, ownedSpec(profileID, id)).Return(entity.Collection{}, nil).Times(3)
	svc := NewCollectionService(repo)

	_, err := svc.Get(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id})
	assertNotFound(t, err)

	_, err = svc.Update(ctx, dto.CollectionRequest{ProfileID: profileID, ID: id, Title: "Hijacked"})
	assertNotFound(t, err)

	assertNotFound(t, svc.Delete(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id}))
}

func TestCollectionService_NilIDIsNotFound(t *testing.T) {
	// No repository call is expected: a zero id would drop the ID condition.
	svc := NewCollectionService(mocks.NewMockRepository[entity.Collection](t))

	_, err := svc.Get(context.Background(), dto.CollectionLookup{ProfileID: uuid.New()})
	assertNotFound(t, err)
}

func TestCollectionService_Get_PropagatesRepositoryError(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	wantErr := errors.New("boom")

	repo := mocks.NewMockRepository[entity.Collection](t)
	repo.EXPECT().FindFirst(ctx, ownedSpec(profileID, id)).Return(entity.Collection{}, wantErr).Once()

	_, err := NewCollectionService(repo).Get(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id})
	assert.ErrorIs(t, err, wantErr)
}

func TestCollectionService_Update(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	limit := 50

	repo := mocks.NewMockRepository[entity.Collection](t)
	repo.EXPECT().FindFirst(ctx, ownedSpec(profileID, id)).Return(entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "Old", Description: "old"}, nil).Once()
	repo.EXPECT().
		Update(ctx, entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "New", Description: "new", MaxCardCount: limit}).
		Return(entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "New", Description: "new", MaxCardCount: limit}, nil).
		Once()

	got, err := NewCollectionService(repo).Update(ctx, dto.CollectionRequest{ProfileID: profileID, ID: id, Title: "New", Description: "new", MaxCardCount: limit})
	require.NoError(t, err)
	assert.Equal(t, "New", got.Title)
	assert.Equal(t, "new", got.Description)
	assert.Equal(t, limit, got.MaxCardCount)
}

func TestCollectionService_Delete(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	found := entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "Binder"}

	repo := mocks.NewMockRepository[entity.Collection](t)
	repo.EXPECT().FindFirst(ctx, ownedSpec(profileID, id)).Return(found, nil).Once()
	repo.EXPECT().Delete(ctx, found).Return(nil).Once()

	assert.NoError(t, NewCollectionService(repo).Delete(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id}))
}

func assertNotFound(t *testing.T, err error) {
	t.Helper()
	var appErr ungerr.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, http.StatusNotFound, appErr.HttpStatus())
}

// A nil ProfileID would make crud.WhereBySpec drop the owner condition.
func TestCollectionService_NilProfile(t *testing.T) {
	ctx := context.Background()
	svc := NewCollectionService(mocks.NewMockRepository[entity.Collection](t)) // no repository call is expected

	got, err := svc.List(ctx, dto.CollectionListRequest{})
	require.NoError(t, err)
	assert.Empty(t, got)

	_, err = svc.Get(ctx, dto.CollectionLookup{ID: uuid.New()})
	assertNotFound(t, err)
	_, err = svc.Update(ctx, dto.CollectionRequest{ID: uuid.New(), Title: "T"})
	assertNotFound(t, err)
	assertNotFound(t, svc.Delete(ctx, dto.CollectionLookup{ID: uuid.New()}))
}
