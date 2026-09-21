package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
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

	// ProfileID is now an opaque FK into user_profiles (see
	// entity/user_profile.go), not the name itself — check it via the
	// row it actually points to.
	profileID, err := uuid.Parse(updated.ProfileID)
	if err != nil {
		t.Fatalf("expected ProfileID to be a valid UUID, got %q: %v", updated.ProfileID, err)
	}

	var profile entity.UserProfile
	if err := db.First(&profile, "id = ?", profileID).Error; err != nil {
		t.Fatalf("expected a user_profiles row for id %s: %v", profileID, err)
	}
	if profile.Name != "Bob" {
		t.Fatalf("expected user_profiles.name %q, got %q", "Bob", profile.Name)
	}
	if profile.UserID.String() != created.ID {
		t.Fatalf("expected user_profiles.user_id %q, got %q", created.ID, profile.UserID)
	}

	// A second SetVerified call updates the same profile row rather than
	// creating a duplicate.
	updatedAgain, err := repo.SetVerified(ctx, created.ID, "Bobby", "")
	if err != nil {
		t.Fatalf("second SetVerified: %v", err)
	}
	if updatedAgain.ProfileID != updated.ProfileID {
		t.Fatalf("expected ProfileID to stay %q on a second SetVerified, got %q", updated.ProfileID, updatedAgain.ProfileID)
	}

	var reloaded entity.UserProfile
	if err := db.First(&reloaded, "id = ?", profileID).Error; err != nil {
		t.Fatalf("expected the same user_profiles row to still exist: %v", err)
	}
	if reloaded.Name != "Bobby" {
		t.Fatalf("expected user_profiles.name to be updated to %q, got %q", "Bobby", reloaded.Name)
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
