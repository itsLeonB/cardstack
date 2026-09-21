package repository

import (
	"context"
	"testing"
	"time"

	authkit "github.com/itsLeonB/go-authkit"
)

func TestRefreshTokenRepository_CreateAndFindByHash(t *testing.T) {
	db := testDB(t)
	userRepo := NewUserRepository(db)
	sessionRepo := NewSessionRepository(db)
	refreshRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	user, err := userRepo.Create(ctx, uniqueEmail(t), "hash")
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}
	session, err := sessionRepo.Create(ctx, user.ID)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	hash := uniqueHash(t)
	expiresAt := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	if err := refreshRepo.Create(ctx, session.ID, hash, expiresAt); err != nil {
		t.Fatalf("Create: %v", err)
	}

	found, err := refreshRepo.FindByHash(ctx, hash)
	if err != nil {
		t.Fatalf("FindByHash: %v", err)
	}
	if found.SessionID != session.ID {
		t.Fatalf("expected SessionID %q, got %q", session.ID, found.SessionID)
	}
	if !found.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected ExpiresAt %v, got %v", expiresAt, found.ExpiresAt)
	}
}

func TestRefreshTokenRepository_FindByHash_NotFound(t *testing.T) {
	db := testDB(t)
	refreshRepo := NewRefreshTokenRepository(db)

	_, err := refreshRepo.FindByHash(context.Background(), "does-not-exist")
	if err != authkit.ErrTokenNotFound {
		t.Fatalf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestRefreshTokenRepository_Delete(t *testing.T) {
	db := testDB(t)
	userRepo := NewUserRepository(db)
	sessionRepo := NewSessionRepository(db)
	refreshRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	user, err := userRepo.Create(ctx, uniqueEmail(t), "hash")
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}
	session, err := sessionRepo.Create(ctx, user.ID)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}
	hash := uniqueHash(t)
	if err := refreshRepo.Create(ctx, session.ID, hash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := refreshRepo.Delete(ctx, session.ID, hash); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := refreshRepo.FindByHash(ctx, hash); err != authkit.ErrTokenNotFound {
		t.Fatalf("expected ErrTokenNotFound after Delete, got %v", err)
	}
}

func TestRefreshTokenRepository_DeleteBySession(t *testing.T) {
	db := testDB(t)
	userRepo := NewUserRepository(db)
	sessionRepo := NewSessionRepository(db)
	refreshRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	user, err := userRepo.Create(ctx, uniqueEmail(t), "hash")
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}
	session, err := sessionRepo.Create(ctx, user.ID)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}
	hashOne, hashTwo := uniqueHash(t), uniqueHash(t)
	if err := refreshRepo.Create(ctx, session.ID, hashOne, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Create hashOne: %v", err)
	}
	if err := refreshRepo.Create(ctx, session.ID, hashTwo, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Create hashTwo: %v", err)
	}

	if err := refreshRepo.DeleteBySession(ctx, session.ID); err != nil {
		t.Fatalf("DeleteBySession: %v", err)
	}

	for _, hash := range []string{hashOne, hashTwo} {
		if _, err := refreshRepo.FindByHash(ctx, hash); err != authkit.ErrTokenNotFound {
			t.Fatalf("expected ErrTokenNotFound for %q after DeleteBySession, got %v", hash, err)
		}
	}
}
