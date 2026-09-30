package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	"github.com/itsLeonB/ungerr"
)

// The handler methods are called directly: Routes() puts SessionGuard in
// front of every route, which needs a real AuthKit session.
func newTestCollectionHandler(t *testing.T) (*CollectionHandler, *mocks.MockCollectionService, context.Context, uuid.UUID) {
	t.Helper()

	svc := mocks.NewMockCollectionService(t)
	userID := uuid.New()
	ctx := authpkg.WithUserID(context.Background(), userID.String())

	return &CollectionHandler{collectionSvc: svc}, svc, ctx, userID
}

func requireStatus(t *testing.T, err error, want int) {
	t.Helper()

	var appErr ungerr.AppError
	if !errors.As(err, &appErr) || appErr.HttpStatus() != want {
		t.Fatalf("expected AppError with status %d, got %v", want, err)
	}
}

func TestCollectionHandler_MissingSession(t *testing.T) {
	h, _, _, _ := newTestCollectionHandler(t)

	_, err := h.list(context.Background(), listCollectionsInput{})
	requireStatus(t, err, http.StatusUnauthorized)

	_, err = h.list(authpkg.WithUserID(context.Background(), "not-a-uuid"), listCollectionsInput{})
	requireStatus(t, err, http.StatusUnauthorized)
}

func TestCollectionHandler_BlankTitle(t *testing.T) {
	h, _, ctx, _ := newTestCollectionHandler(t)

	_, err := h.create(ctx, createCollectionInput{Body: collectionBody{Title: " \t "}})
	requireStatus(t, err, http.StatusBadRequest)

	_, err = h.update(ctx, updateCollectionInput{ID: uuid.NewString(), Body: collectionBody{Title: " "}})
	requireStatus(t, err, http.StatusBadRequest)
}

func TestCollectionHandler_InvalidCollectionID(t *testing.T) {
	h, _, ctx, _ := newTestCollectionHandler(t)

	_, err := h.get(ctx, collectionIDInput{ID: "nope"})
	requireStatus(t, err, http.StatusBadRequest)

	_, err = h.update(ctx, updateCollectionInput{ID: "nope"})
	requireStatus(t, err, http.StatusBadRequest)

	requireStatus(t, h.delete(ctx, collectionIDInput{ID: "nope"}), http.StatusBadRequest)
}

func TestCollectionHandler_Create(t *testing.T) {
	h, svc, ctx, userID := newTestCollectionHandler(t)
	limit := 10
	want := dto.CollectionSummary{Title: "Binder"}
	svc.EXPECT().
		Create(ctx, userID, dto.CreateCollectionRequest{Title: "Binder", Description: "d", MaxCardCount: &limit}).
		Return(want, nil)

	got, err := h.create(ctx, createCollectionInput{Body: collectionBody{Title: "Binder", Description: "d", MaxCardCount: &limit}})
	if err != nil || got != want {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestCollectionHandler_List(t *testing.T) {
	h, svc, ctx, userID := newTestCollectionHandler(t)
	svc.EXPECT().List(ctx, userID).Return([]dto.CollectionSummary{{Title: "A"}}, nil)

	got, err := h.list(ctx, listCollectionsInput{})
	if err != nil || len(got) != 1 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestCollectionHandler_NotFoundPassesThrough(t *testing.T) {
	h, svc, ctx, userID := newTestCollectionHandler(t)
	id := uuid.New()
	svc.EXPECT().Get(ctx, userID, id).Return(dto.CollectionSummary{}, repository.ErrCollectionNotFound)
	svc.EXPECT().Update(ctx, userID, id, dto.UpdateCollectionRequest{Title: "T"}).Return(dto.CollectionSummary{}, repository.ErrCollectionNotFound)
	svc.EXPECT().Delete(ctx, userID, id).Return(repository.ErrCollectionNotFound)

	_, err := h.get(ctx, collectionIDInput{ID: id.String()})
	requireStatus(t, err, http.StatusNotFound)

	_, err = h.update(ctx, updateCollectionInput{ID: id.String(), Body: collectionBody{Title: "T"}})
	requireStatus(t, err, http.StatusNotFound)

	requireStatus(t, h.delete(ctx, collectionIDInput{ID: id.String()}), http.StatusNotFound)
}

func TestCollectionHandler_DeleteSuccess(t *testing.T) {
	h, svc, ctx, userID := newTestCollectionHandler(t)
	id := uuid.New()
	svc.EXPECT().Delete(ctx, userID, id).Return(nil)

	if err := h.delete(ctx, collectionIDInput{ID: id.String()}); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
