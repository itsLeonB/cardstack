package repository

import (
	"context"
	"testing"

	authkit "github.com/itsLeonB/go-authkit"
)

func TestUserRepository_CreateAndFindByEmail(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()
	email := uniqueEmail(t)

	created, err := repo.Create(ctx, email, "hashed-password")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected a generated ID")
	}
	if created.Verified {
		t.Fatal("expected a newly created user to be unverified")
	}

	found, err := repo.FindByEmail(ctx, email)
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}
	if found.ID != created.ID || found.PasswordHash != "hashed-password" {
		t.Fatalf("FindByEmail returned %+v, want ID=%s PasswordHash=hashed-password", found, created.ID)
	}

	byID, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if byID.Email != email {
		t.Fatalf("FindByID returned email %q, want %q", byID.Email, email)
	}
}

func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)

	_, err := repo.FindByEmail(context.Background(), "nobody@example.com")
	if err != authkit.ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)

	_, err := repo.FindByID(context.Background(), "not-a-uuid")
	if err != authkit.ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound for a malformed ID, got %v", err)
	}
}

func TestUserRepository_SetVerified(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	created, err := repo.Create(ctx, uniqueEmail(t), "hash")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := repo.SetVerified(ctx, created.ID, "Bob", "")
	if err != nil {
		t.Fatalf("SetVerified: %v", err)
	}
	if !updated.Verified {
		t.Fatal("expected Verified to be true after SetVerified")
	}
	if updated.ProfileID != "Bob" {
		t.Fatalf("expected ProfileID %q, got %q", "Bob", updated.ProfileID)
	}
}

func TestUserRepository_UpdatePassword(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	created, err := repo.Create(ctx, uniqueEmail(t), "old-hash")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.UpdatePassword(ctx, created.ID, "new-hash"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}

	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.PasswordHash != "new-hash" {
		t.Fatalf("expected PasswordHash %q, got %q", "new-hash", found.PasswordHash)
	}
}

func TestUserRepository_Exists(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	created, err := repo.Create(ctx, uniqueEmail(t), "hash")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Exists(ctx, created.ID); err != nil {
		t.Fatalf("Exists for a real user: %v", err)
	}

	if err := repo.Exists(ctx, "00000000-0000-0000-0000-000000000000"); err != authkit.ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound for a missing user, got %v", err)
	}
}
