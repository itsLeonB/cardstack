package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/apperr"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type collectionFixture struct {
	repo    *mocks.MockCollectionRepository
	layouts *mocks.MockBinderLayoutRepository
	leaves  *mocks.MockBinderLeafRepository
	svc     CollectionService
}

// newCollectionFixture's Transactor mock runs the transaction body inline. The
// binder repositories are plain unless a test expects otherwise.
func newCollectionFixture(t *testing.T) collectionFixture {
	t.Helper()
	f := collectionFixture{
		repo:    mocks.NewMockCollectionRepository(t),
		layouts: mocks.NewMockBinderLayoutRepository(t),
		leaves:  mocks.NewMockBinderLeafRepository(t),
	}
	transactor := mocks.NewMockTransactor(t)
	transactor.EXPECT().WithinTransaction(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Maybe()
	f.svc = NewCollectionService(transactor, f.repo, f.layouts, f.leaves)
	return f
}

func TestCollectionService_Create(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	limit := 100
	f := newCollectionFixture(t)

	f.repo.EXPECT().
		Insert(ctx, entity.Collection{ProfileID: profileID, Title: "Binder", Description: "desc", MaxCardCount: limit}).
		Return(entity.Collection{BaseEntity: baseEntity(uuid.New()), ProfileID: profileID, Title: "Binder", Description: "desc", MaxCardCount: limit}, nil).
		Once()

	got, err := f.svc.Create(ctx, dto.CollectionRequest{ProfileID: profileID, Title: "Binder", Description: "desc", MaxCardCount: limit})
	require.NoError(t, err)
	assert.Equal(t, "Binder", got.Title)
	assert.Equal(t, "desc", got.Description)
	assert.Equal(t, limit, got.MaxCardCount)
	assert.Zero(t, got.CardCount)
	assert.Nil(t, got.Binder)
}

func TestCollectionService_Create_PropagatesRepositoryError(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	wantErr := errors.New("boom")
	f := newCollectionFixture(t)

	f.repo.EXPECT().Insert(ctx, entity.Collection{ProfileID: profileID, Title: "Binder"}).Return(entity.Collection{}, wantErr).Once()

	_, err := f.svc.Create(ctx, dto.CollectionRequest{ProfileID: profileID, Title: "Binder"})
	assert.ErrorIs(t, err, wantErr)
}

func TestCollectionService_List(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	emptyID := uuid.New()
	f := newCollectionFixture(t)

	layout := entity.CollectionLayout{BaseEntity: baseEntity(uuid.New()), CollectionID: id, Rows: 2, Columns: 2}
	f.repo.EXPECT().
		FindAll(ctx, crud.Specification[entity.Collection]{
			Model:            entity.Collection{ProfileID: profileID},
			PreloadRelations: []string{"Layout"},
		}).
		Return([]entity.Collection{
			{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "Binder", Layout: &layout},
			{BaseEntity: baseEntity(emptyID), ProfileID: profileID, Title: "Empty"},
		}, nil).
		Once()
	// One aggregate call for the whole list; an empty Collection is absent from the map.
	f.repo.EXPECT().SumQuantities(ctx, []uuid.UUID{id, emptyID}).Return(map[uuid.UUID]int{id: 37}, nil).Once()
	f.leaves.EXPECT().LeafCounts(ctx, []uuid.UUID{id, emptyID}).Return(map[uuid.UUID]int{id: 3}, nil).Once()

	got, err := f.svc.List(ctx, dto.CollectionListRequest{ProfileID: profileID})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, id, got[0].ID)
	assert.Equal(t, "Binder", got[0].Title)
	assert.Equal(t, 37, got[0].CardCount)
	assert.Equal(t, &dto.BinderLayout{Rows: 2, Columns: 2, LeafCount: 3, Capacity: 24}, got[0].Binder)
	assert.Zero(t, got[1].CardCount)
	assert.Nil(t, got[1].Binder)
}

func TestCollectionService_List_PropagatesSumError(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	wantErr := errors.New("boom")
	f := newCollectionFixture(t)

	f.repo.EXPECT().FindAll(ctx, mock.Anything).
		Return([]entity.Collection{{BaseEntity: baseEntity(uuid.New()), ProfileID: profileID}}, nil).Once()
	f.repo.EXPECT().SumQuantities(ctx, mock.Anything).Return(nil, wantErr).Once()

	_, err := f.svc.List(ctx, dto.CollectionListRequest{ProfileID: profileID})
	assert.ErrorIs(t, err, wantErr)
}

func TestCollectionService_Get_ReturnsOwnedCollection(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	f := newCollectionFixture(t)

	f.repo.EXPECT().GetOwnedCollection(ctx, profileID, id, false).Return(entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "Binder"}, nil).Once()
	f.repo.EXPECT().SumQuantities(ctx, []uuid.UUID{id}).Return(map[uuid.UUID]int{id: 12}, nil).Once()
	f.layouts.EXPECT().FindLayout(ctx, id).Return(entity.CollectionLayout{}, nil).Once()

	got, err := f.svc.Get(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id})
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, 12, got.CardCount)
	assert.Nil(t, got.Binder)
}

func TestCollectionService_Get_ReturnsBinderLayout(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	f := newCollectionFixture(t)

	f.repo.EXPECT().GetOwnedCollection(ctx, profileID, id, false).Return(entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID}, nil).Once()
	f.repo.EXPECT().SumQuantities(ctx, []uuid.UUID{id}).Return(map[uuid.UUID]int{}, nil).Once()
	f.layouts.EXPECT().FindLayout(ctx, id).Return(entity.CollectionLayout{BaseEntity: baseEntity(uuid.New()), CollectionID: id, Rows: 3, Columns: 3}, nil).Once()
	f.leaves.EXPECT().CountLeaves(ctx, id).Return(10, nil).Once()

	got, err := f.svc.Get(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id})
	require.NoError(t, err)
	assert.Equal(t, &dto.BinderLayout{Rows: 3, Columns: 3, LeafCount: 10, Capacity: 180}, got.Binder)
}

// TestCollectionService_NotFound: the repository's not-found AppError (missing
// or another profile's Collection, nil ids) is returned unchanged, and no
// write follows it.
func TestCollectionService_NotFound(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	f := newCollectionFixture(t)

	f.repo.EXPECT().GetOwnedCollection(ctx, profileID, id, false).Return(entity.Collection{}, ungerr.NotFoundError("collection not found")).Twice()
	f.repo.EXPECT().GetOwnedCollection(ctx, profileID, id, true).Return(entity.Collection{}, ungerr.NotFoundError("collection not found")).Once()

	_, err := f.svc.Get(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id})
	assertNotFound(t, err)

	_, err = f.svc.Update(ctx, dto.CollectionRequest{ProfileID: profileID, ID: id, Title: "Hijacked"})
	assertNotFound(t, err)

	assertNotFound(t, f.svc.Delete(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id}))
}

func TestCollectionService_Get_PropagatesRepositoryError(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	wantErr := errors.New("boom")
	f := newCollectionFixture(t)

	f.repo.EXPECT().GetOwnedCollection(ctx, profileID, id, false).Return(entity.Collection{}, wantErr).Once()

	_, err := f.svc.Get(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id})
	assert.ErrorIs(t, err, wantErr)
}

func TestCollectionService_Update(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	limit := 50
	f := newCollectionFixture(t)

	f.repo.EXPECT().GetOwnedCollection(ctx, profileID, id, true).Return(entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "Old", Description: "old"}, nil).Once()
	f.layouts.EXPECT().IsBinder(ctx, id).Return(false, nil).Once()
	f.repo.EXPECT().
		Update(ctx, entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "New", Description: "new", MaxCardCount: limit}).
		Return(entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "New", Description: "new", MaxCardCount: limit}, nil).
		Once()
	f.repo.EXPECT().SumQuantities(ctx, []uuid.UUID{id}).Return(map[uuid.UUID]int{id: 7}, nil).Once()
	f.layouts.EXPECT().FindLayout(ctx, id).Return(entity.CollectionLayout{}, nil).Once()

	got, err := f.svc.Update(ctx, dto.CollectionRequest{ProfileID: profileID, ID: id, Title: "New", Description: "new", MaxCardCount: limit})
	require.NoError(t, err)
	assert.Equal(t, "New", got.Title)
	assert.Equal(t, "new", got.Description)
	assert.Equal(t, limit, got.MaxCardCount)
	assert.Equal(t, 7, got.CardCount)
}

func TestCollectionService_Update_RefusesMaxCardCountOnBinder(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	f := newCollectionFixture(t)

	f.repo.EXPECT().GetOwnedCollection(ctx, profileID, id, true).Return(entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID}, nil).Once()
	f.layouts.EXPECT().IsBinder(ctx, id).Return(true, nil).Once()

	_, err := f.svc.Update(ctx, dto.CollectionRequest{ProfileID: profileID, ID: id, Title: "Binder", MaxCardCount: 50})
	requireStatus(t, err, http.StatusConflict)
	assert.Equal(t, apperr.CodeBinderMaxCardCount, apperr.CodeOf(err))
}

func TestCollectionService_Update_AllowsZeroMaxCardCountOnBinder(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	f := newCollectionFixture(t)

	f.repo.EXPECT().GetOwnedCollection(ctx, profileID, id, true).Return(entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID}, nil).Once()
	f.repo.EXPECT().Update(ctx, mock.Anything).Return(entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "Binder"}, nil).Once()
	f.repo.EXPECT().SumQuantities(ctx, []uuid.UUID{id}).Return(map[uuid.UUID]int{}, nil).Once()
	f.layouts.EXPECT().FindLayout(ctx, id).Return(entity.CollectionLayout{BaseEntity: baseEntity(uuid.New()), CollectionID: id, Rows: 2, Columns: 2}, nil).Once()
	f.leaves.EXPECT().CountLeaves(ctx, id).Return(1, nil).Once()

	got, err := f.svc.Update(ctx, dto.CollectionRequest{ProfileID: profileID, ID: id, Title: "Binder"})
	require.NoError(t, err)
	assert.NotNil(t, got.Binder)
}

func TestCollectionService_Delete(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	id := uuid.New()
	found := entity.Collection{BaseEntity: baseEntity(id), ProfileID: profileID, Title: "Binder"}
	f := newCollectionFixture(t)

	f.repo.EXPECT().GetOwnedCollection(ctx, profileID, id, false).Return(found, nil).Once()
	f.repo.EXPECT().Delete(ctx, found).Return(nil).Once()

	assert.NoError(t, f.svc.Delete(ctx, dto.CollectionLookup{ProfileID: profileID, ID: id}))
}

func assertNotFound(t *testing.T, err error) {
	t.Helper()
	requireStatus(t, err, http.StatusNotFound)
}

// A nil ProfileID must not list everyone's collections; the lookups' nil
// guard lives in CollectionRepository.GetOwnedCollection.
func TestCollectionService_List_NilProfile(t *testing.T) {
	f := newCollectionFixture(t) // no repository call is expected

	got, err := f.svc.List(context.Background(), dto.CollectionListRequest{})
	require.NoError(t, err)
	assert.Empty(t, got)
}
