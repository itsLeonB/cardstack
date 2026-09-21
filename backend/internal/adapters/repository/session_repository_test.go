package repository

import (
	"context"
	"testing"

	authkit "github.com/itsLeonB/go-authkit"
)

func TestSessionRepository_CreateAndGetByID(t *testing.T) {
	db := testDB(t)
	userRepo := NewUserRepository(db)
	sessionRepo := NewSessionRepository(db)
	ctx := context.Background()

	user, err := userRepo.Create(ctx, uniqueEmail(t), "hash")
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}

	session, err := sessionRepo.Create(ctx, user.ID)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}
	if session.ID == "" {
		t.Fatal("expected a generated session ID")
	}
	if session.UserID != user.ID {
		t.Fatalf("expected UserID %q, got %q", user.ID, session.UserID)
	}

	found, err := sessionRepo.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if found.UserID != user.ID {
		t.Fatalf("GetByID returned UserID %q, want %q", found.UserID, user.ID)
	}
}

func TestSessionRepository_GetByID_NotFound(t *testing.T) {
	db := testDB(t)
	sessionRepo := NewSessionRepository(db)

	_, err := sessionRepo.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err != authkit.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSessionRepository_Delete(t *testing.T) {
	db := testDB(t)
	userRepo := NewUserRepository(db)
	sessionRepo := NewSessionRepository(db)
	ctx := context.Background()

	user, err := userRepo.Create(ctx, uniqueEmail(t), "hash")
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}
	session, err := sessionRepo.Create(ctx, user.ID)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	if err := sessionRepo.Delete(ctx, session.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := sessionRepo.GetByID(ctx, session.ID); err != authkit.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound after Delete, got %v", err)
	}
}

func TestSessionRepository_Touch(t *testing.T) {
	db := testDB(t)
	userRepo := NewUserRepository(db)
	sessionRepo := NewSessionRepository(db)
	ctx := context.Background()

	user, err := userRepo.Create(ctx, uniqueEmail(t), "hash")
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}
	session, err := sessionRepo.Create(ctx, user.ID)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	if err := sessionRepo.Touch(ctx, session.ID); err != nil {
		t.Fatalf("Touch: %v", err)
	}

	// Touch on a session that doesn't exist is not an error (no rows
	// matched is a no-op update, same as authgin's own Touch usage after
	// rotateRefreshToken already validated the session exists).
	if err := sessionRepo.Touch(ctx, "00000000-0000-0000-0000-000000000000"); err != nil {
		t.Fatalf("Touch on a missing session should be a no-op, got %v", err)
	}
}
