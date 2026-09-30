package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
)

func TestCollectionService_Create(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	limit := 100

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().
		Create(ctx, entity.Collection{UserID: userID, Title: "Binder", Description: "desc", MaxCardCount: &limit}).
		Return(entity.Collection{BaseEntity: baseEntity(uuid.New()), UserID: userID, Title: "Binder", Description: "desc", MaxCardCount: &limit}, nil).
		Once()
	svc := NewCollectionService(repo)

	got, err := svc.Create(ctx, userID, dto.CreateCollectionRequest{Title: "Binder", Description: "desc", MaxCardCount: &limit})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Title != "Binder" || got.Description != "desc" || got.MaxCardCount == nil || *got.MaxCardCount != limit {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestCollectionService_Create_PropagatesRepositoryError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	wantErr := errors.New("boom")

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().Create(ctx, entity.Collection{UserID: userID, Title: "Binder"}).Return(entity.Collection{}, wantErr).Once()
	svc := NewCollectionService(repo)

	_, err := svc.Create(ctx, userID, dto.CreateCollectionRequest{Title: "Binder"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestCollectionService_List(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	id := uuid.New()

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().ListByUser(ctx, userID).Return([]entity.Collection{
		{BaseEntity: baseEntity(id), UserID: userID, Title: "Binder"},
	}, nil).Once()
	svc := NewCollectionService(repo)

	got, err := svc.List(ctx, userID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].ID != id || got[0].Title != "Binder" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestCollectionService_Get_ReturnsOwnedCollection(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	id := uuid.New()

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().FindByID(ctx, id).Return(entity.Collection{BaseEntity: baseEntity(id), UserID: userID, Title: "Binder"}, nil).Once()
	svc := NewCollectionService(repo)

	got, err := svc.Get(ctx, userID, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != id {
		t.Fatalf("expected collection %s, got %+v", id, got)
	}
}

func TestCollectionService_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	id := uuid.New()

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().FindByID(ctx, id).Return(entity.Collection{}, repository.ErrCollectionNotFound).Once()
	svc := NewCollectionService(repo)

	_, err := svc.Get(ctx, userID, id)
	if !errors.Is(err, repository.ErrCollectionNotFound) {
		t.Fatalf("expected repository.ErrCollectionNotFound, got %v", err)
	}
}

// TestCollectionService_Get_AnotherUsersCollection is the ownership check
// (ticket 06's "a user cannot view/edit/delete another user's Collection"):
// a Collection that exists but belongs to someone else answers identically
// to one that doesn't exist at all, so its existence isn't leaked.
func TestCollectionService_Get_AnotherUsersCollection(t *testing.T) {
	ctx := context.Background()
	owner := uuid.New()
	otherUser := uuid.New()
	id := uuid.New()

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().FindByID(ctx, id).Return(entity.Collection{BaseEntity: baseEntity(id), UserID: owner, Title: "Binder"}, nil).Once()
	svc := NewCollectionService(repo)

	_, err := svc.Get(ctx, otherUser, id)
	if !errors.Is(err, repository.ErrCollectionNotFound) {
		t.Fatalf("expected repository.ErrCollectionNotFound for another user's collection, got %v", err)
	}
}

func TestCollectionService_Update(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	id := uuid.New()
	limit := 50

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().FindByID(ctx, id).Return(entity.Collection{BaseEntity: baseEntity(id), UserID: userID, Title: "Old", Description: "old"}, nil).Once()
	repo.EXPECT().
		Update(ctx, entity.Collection{BaseEntity: baseEntity(id), UserID: userID, Title: "New", Description: "new", MaxCardCount: &limit}).
		Return(entity.Collection{BaseEntity: baseEntity(id), UserID: userID, Title: "New", Description: "new", MaxCardCount: &limit}, nil).
		Once()
	svc := NewCollectionService(repo)

	got, err := svc.Update(ctx, userID, id, dto.UpdateCollectionRequest{Title: "New", Description: "new", MaxCardCount: &limit})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Title != "New" || got.Description != "new" || got.MaxCardCount == nil || *got.MaxCardCount != limit {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestCollectionService_Update_AnotherUsersCollection(t *testing.T) {
	ctx := context.Background()
	owner := uuid.New()
	otherUser := uuid.New()
	id := uuid.New()

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().FindByID(ctx, id).Return(entity.Collection{BaseEntity: baseEntity(id), UserID: owner, Title: "Binder"}, nil).Once()
	svc := NewCollectionService(repo)

	_, err := svc.Update(ctx, otherUser, id, dto.UpdateCollectionRequest{Title: "Hijacked"})
	if !errors.Is(err, repository.ErrCollectionNotFound) {
		t.Fatalf("expected repository.ErrCollectionNotFound, got %v", err)
	}
}

func TestCollectionService_Delete(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	id := uuid.New()

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().FindByID(ctx, id).Return(entity.Collection{BaseEntity: baseEntity(id), UserID: userID, Title: "Binder"}, nil).Once()
	repo.EXPECT().Delete(ctx, entity.Collection{BaseEntity: baseEntity(id), UserID: userID, Title: "Binder"}).Return(nil).Once()
	svc := NewCollectionService(repo)

	if err := svc.Delete(ctx, userID, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestCollectionService_Delete_AnotherUsersCollection(t *testing.T) {
	ctx := context.Background()
	owner := uuid.New()
	otherUser := uuid.New()
	id := uuid.New()

	repo := mocks.NewMockCollectionRepository(t)
	repo.EXPECT().FindByID(ctx, id).Return(entity.Collection{BaseEntity: baseEntity(id), UserID: owner, Title: "Binder"}, nil).Once()
	svc := NewCollectionService(repo)

	err := svc.Delete(ctx, otherUser, id)
	if !errors.Is(err, repository.ErrCollectionNotFound) {
		t.Fatalf("expected repository.ErrCollectionNotFound, got %v", err)
	}
}
