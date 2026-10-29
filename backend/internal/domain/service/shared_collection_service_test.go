package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sharedFixture struct {
	profiles    *mocks.MockRepository[entity.UserProfile]
	collections *mocks.MockCollectionRepository
	svc         SharedCollectionService
}

func newSharedFixture(t *testing.T) sharedFixture {
	t.Helper()
	f := sharedFixture{
		profiles:    mocks.NewMockRepository[entity.UserProfile](t),
		collections: mocks.NewMockCollectionRepository(t),
	}
	f.svc = NewSharedCollectionService(f.profiles, f.collections, nil, mapper.NewImageHost(""))
	return f
}

// profileSpec is the exact lookup the service makes for a profile Slug.
func sharedProfileSpec(slug string) crud.Specification[entity.UserProfile] {
	return crud.Specification[entity.UserProfile]{Model: entity.UserProfile{Slug: slug}}
}

// TestSharedCollectionService_Get_ResolvesBothSlugs: the owner's display name
// comes from the profile, and the title and description from the Collection.
func TestSharedCollectionService_Get_ResolvesBothSlugs(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	f := newSharedFixture(t)

	f.profiles.EXPECT().FindFirst(ctx, sharedProfileSpec("ana-a1b2c")).
		Return(entity.UserProfile{BaseEntity: baseEntity(profileID), Name: "Ana", Slug: "ana-a1b2c"}, nil).Once()
	f.collections.EXPECT().FindSharedCollection(ctx, profileID, "binder-d3e4f").
		Return(entity.Collection{BaseEntity: baseEntity(uuid.New()), ProfileID: profileID, Title: "Binder", Description: "my cards"}, nil).Once()

	got, err := f.svc.Get(ctx, dto.SharedCollectionLookup{ProfileSlug: "ana-a1b2c", CollectionSlug: "binder-d3e4f"})
	require.NoError(t, err)
	assert.Equal(t, dto.SharedCollectionSummary{OwnerName: "Ana", Title: "Binder", Description: "my cards"}, got)
}

// TestSharedCollectionService_Get_EveryMissIsTheSameNotFound: an empty profile
// Slug, an unknown profile, an unknown Collection and an empty Collection Slug
// all answer one identical not-found. The mocks allow no call beyond the ones
// listed, so a miss that reaches a later lookup fails the test.
func TestSharedCollectionService_Get_EveryMissIsTheSameNotFound(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	owner := entity.UserProfile{BaseEntity: baseEntity(profileID), Name: "Ana", Slug: "ana-a1b2c"}

	t.Run("empty profile slug", func(t *testing.T) {
		f := newSharedFixture(t)

		_, err := f.svc.Get(ctx, dto.SharedCollectionLookup{ProfileSlug: "", CollectionSlug: "binder-d3e4f"})
		assertSharedNotFound(t, err)
	})

	t.Run("unknown profile", func(t *testing.T) {
		f := newSharedFixture(t)
		f.profiles.EXPECT().FindFirst(ctx, sharedProfileSpec("nobody-00000")).Return(entity.UserProfile{}, nil).Once()

		_, err := f.svc.Get(ctx, dto.SharedCollectionLookup{ProfileSlug: "nobody-00000", CollectionSlug: "binder-d3e4f"})
		assertSharedNotFound(t, err)
	})

	t.Run("unknown collection", func(t *testing.T) {
		f := newSharedFixture(t)
		f.profiles.EXPECT().FindFirst(ctx, sharedProfileSpec("ana-a1b2c")).Return(owner, nil).Once()
		f.collections.EXPECT().FindSharedCollection(ctx, profileID, "no-such-binder").Return(entity.Collection{}, nil).Once()

		_, err := f.svc.Get(ctx, dto.SharedCollectionLookup{ProfileSlug: "ana-a1b2c", CollectionSlug: "no-such-binder"})
		assertSharedNotFound(t, err)
	})

	t.Run("empty collection slug", func(t *testing.T) {
		f := newSharedFixture(t)
		f.profiles.EXPECT().FindFirst(ctx, sharedProfileSpec("ana-a1b2c")).Return(owner, nil).Once()
		f.collections.EXPECT().FindSharedCollection(ctx, profileID, "").Return(entity.Collection{}, nil).Once()

		_, err := f.svc.Get(ctx, dto.SharedCollectionLookup{ProfileSlug: "ana-a1b2c", CollectionSlug: ""})
		assertSharedNotFound(t, err)
	})
}

// assertSharedNotFound: the 404 whose detail is the shared-link message, so
// every miss reads the same to a caller.
func assertSharedNotFound(t *testing.T, err error) {
	t.Helper()
	var appErr ungerr.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, http.StatusNotFound, appErr.HttpStatus())
	assert.Equal(t, sharedNotFoundMsg, appErr.Details())
}
