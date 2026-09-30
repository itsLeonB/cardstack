package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"gorm.io/gorm"
)

// newUserFixture inserts a users row unique to this test run, since
// collections.user_id references users(id) and this database is shared
// with other packages' tests (see testDB's doc comment).
func newUserFixture(t *testing.T, db *gorm.DB) entity.User {
	t.Helper()

	user := entity.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hashed"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("creating user fixture: %v", err)
	}

	return user
}

func TestCollectionRepository_CreateAndFindByID(t *testing.T) {
	db := testDB(t)
	user := newUserFixture(t, db)
	repo := NewCollectionRepository(db)
	ctx := context.Background()
	limit := 100

	created, err := repo.Create(ctx, entity.Collection{
		UserID:       user.ID,
		Title:        "Base Set Binder",
		Description:  "My original cards",
		MaxCardCount: &limit,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatal("expected a generated ID")
	}

	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Title != "Base Set Binder" || found.Description != "My original cards" || found.MaxCardCount == nil || *found.MaxCardCount != limit {
		t.Fatalf("FindByID returned %+v", found)
	}
}

func TestCollectionRepository_FindByID_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewCollectionRepository(db)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("expected ErrCollectionNotFound, got %v", err)
	}
}

func TestCollectionRepository_FindByID_Nil(t *testing.T) {
	db := testDB(t)
	repo := NewCollectionRepository(db)

	_, err := repo.FindByID(context.Background(), uuid.Nil)
	if !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("expected ErrCollectionNotFound for uuid.Nil, got %v", err)
	}
}

func TestCollectionRepository_ListByUser(t *testing.T) {
	db := testDB(t)
	user := newUserFixture(t, db)
	otherUser := newUserFixture(t, db)
	repo := NewCollectionRepository(db)
	ctx := context.Background()

	mine, err := repo.Create(ctx, entity.Collection{UserID: user.ID, Title: "Mine"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := repo.Create(ctx, entity.Collection{UserID: otherUser.ID, Title: "Not Mine"}); err != nil {
		t.Fatalf("Create (other user): %v", err)
	}

	found, err := repo.ListByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(found) != 1 || found[0].ID != mine.ID {
		t.Fatalf("expected only %s's collection, got %+v", user.ID, found)
	}
}

func TestCollectionRepository_Update(t *testing.T) {
	db := testDB(t)
	user := newUserFixture(t, db)
	repo := NewCollectionRepository(db)
	ctx := context.Background()

	created, err := repo.Create(ctx, entity.Collection{UserID: user.ID, Title: "Original"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	created.Title = "Renamed"
	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "Renamed" {
		t.Fatalf("expected Title=Renamed, got %+v", updated)
	}

	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Title != "Renamed" {
		t.Fatalf("expected persisted Title=Renamed, got %+v", found)
	}
}

func TestCollectionRepository_Update_ClearsMaxCardCount(t *testing.T) {
	db := testDB(t)
	user := newUserFixture(t, db)
	repo := NewCollectionRepository(db)
	ctx := context.Background()

	limit := 50
	created, err := repo.Create(ctx, entity.Collection{UserID: user.ID, Title: "Limited", MaxCardCount: &limit})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	created.MaxCardCount = nil
	if _, err := repo.Update(ctx, created); err != nil {
		t.Fatalf("Update: %v", err)
	}

	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.MaxCardCount != nil {
		t.Fatalf("expected MaxCardCount cleared, got %d", *found.MaxCardCount)
	}
}

func TestCollectionRepository_Delete(t *testing.T) {
	db := testDB(t)
	user := newUserFixture(t, db)
	repo := NewCollectionRepository(db)
	ctx := context.Background()

	created, err := repo.Create(ctx, entity.Collection{UserID: user.ID, Title: "Temporary"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, created); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = repo.FindByID(ctx, created.ID)
	if !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("expected ErrCollectionNotFound after delete, got %v", err)
	}
}
