package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type wishlistFixture struct {
	transactor *mocks.MockTransactor
	profiles   *mocks.MockRepository[entity.UserProfile]
	wishlists  *mocks.MockRepository[entity.Wishlist]
	items      *mocks.MockWishlistRepository
	cards      *mocks.MockCardLookup
	svc        WishlistService
}

// newWishlistFixture's Transactor mock runs the transaction body inline, so a
// test asserts the same calls the locked write makes.
func newWishlistFixture(t *testing.T) wishlistFixture {
	t.Helper()
	f := wishlistFixture{
		transactor: mocks.NewMockTransactor(t),
		profiles:   mocks.NewMockRepository[entity.UserProfile](t),
		wishlists:  mocks.NewMockRepository[entity.Wishlist](t),
		items:      mocks.NewMockWishlistRepository(t),
		cards:      mocks.NewMockCardLookup(t),
	}
	f.transactor.EXPECT().WithinTransaction(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Maybe()
	f.svc = NewWishlistService(f.transactor, f.profiles, f.wishlists, f.items, f.cards, mapper.NewImageHost(""))

	return f
}

// expectLock is the owner row lock every Wishlist write takes first.
func (f wishlistFixture) expectLock(ctx context.Context, profileID uuid.UUID) {
	f.profiles.EXPECT().
		FindFirst(ctx, crud.Specification[entity.UserProfile]{
			Model:     entity.UserProfile{BaseEntity: crud.BaseEntity{ID: profileID}},
			ForUpdate: true,
		}).
		Return(entity.UserProfile{BaseEntity: baseEntity(profileID)}, nil).
		Once()
}

func TestWishlistService_List(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	setID := uuid.New()
	finished := uuid.New()
	missing := uuid.New()
	f := newWishlistFixture(t)

	f.items.EXPECT().
		List(ctx, repository.WishlistFilter{
			ProfileID: profileID,
			Status:    repository.WishlistStatusFulfilled,
			Cards:     repository.CardFilter{Name: "Pikachu", Limit: 10, Offset: 0},
		}).
		Return([]repository.CardResult{
			{ID: finished, Name: "Pikachu", ExpansionSetID: setID, Quantity: 2, WantedQuantity: 2},
			{ID: missing, Name: "Raichu", ExpansionSetID: setID, Quantity: 0, WantedQuantity: 3},
		}, 2, nil).
		Once()

	items, meta, err := f.svc.List(ctx, dto.WishlistListRequest{
		ProfileID: profileID,
		Status:    dto.WishlistStatusFulfilled,
		Filter:    dto.CardFilter{Name: "Pikachu", Page: 1, Limit: 10},
	})
	require.NoError(t, err)
	assert.Equal(t, dto.PaginationMeta{Total: 2, Page: 1, Limit: 10}, meta)
	require.Len(t, items, 2)
	assert.Equal(t, 2, items[0].WantedQuantity)
	assert.Equal(t, 2, items[0].OwnedQuantity)
	assert.True(t, items[0].Fulfilled)
	assert.Equal(t, 3, items[1].WantedQuantity)
	assert.Equal(t, 0, items[1].OwnedQuantity)
	assert.False(t, items[1].Fulfilled)
}

func TestWishlistService_List_AnonymousIsRefused(t *testing.T) {
	ctx := context.Background()
	f := newWishlistFixture(t)

	_, _, err := f.svc.List(ctx, dto.WishlistListRequest{Filter: dto.CardFilter{Page: 1, Limit: 10}})

	require.Error(t, err)
}

func TestWishlistService_List_PropagatesRepositoryError(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	wantErr := errors.New("boom")
	f := newWishlistFixture(t)
	f.items.EXPECT().List(ctx, mock.Anything).Return(nil, int64(0), wantErr).Once()

	_, _, err := f.svc.List(ctx, dto.WishlistListRequest{ProfileID: profileID, Filter: dto.CardFilter{Page: 1, Limit: 10}})

	require.ErrorIs(t, err, wantErr)
}

func TestWishlistService_SetQuantity_CreatesTheItem(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	cardID := uuid.New()
	f := newWishlistFixture(t)

	f.cards.EXPECT().RequireCard(ctx, cardID).Return(nil).Once()
	f.expectLock(ctx, profileID)
	f.items.EXPECT().
		FindFirst(ctx, crud.Specification[entity.WishlistItem]{
			Model:     entity.WishlistItem{ProfileID: profileID, CardID: cardID},
			ForUpdate: true,
		}).
		Return(entity.WishlistItem{}, nil).
		Once()
	f.items.EXPECT().
		Insert(ctx, entity.WishlistItem{ProfileID: profileID, CardID: cardID, WantedQuantity: 2}).
		Return(entity.WishlistItem{BaseEntity: baseEntity(uuid.New()), ProfileID: profileID, CardID: cardID, WantedQuantity: 2}, nil).
		Once()

	require.NoError(t, f.svc.SetQuantity(ctx, dto.WishlistItemRequest{ProfileID: profileID, CardID: cardID, Quantity: 2}))
}

func TestWishlistService_SetQuantity_ReplacesAnExistingItem(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	cardID := uuid.New()
	item := entity.WishlistItem{BaseEntity: baseEntity(uuid.New()), ProfileID: profileID, CardID: cardID, WantedQuantity: 1}
	f := newWishlistFixture(t)

	f.cards.EXPECT().RequireCard(ctx, cardID).Return(nil).Once()
	f.expectLock(ctx, profileID)
	f.items.EXPECT().FindFirst(ctx, mock.Anything).Return(item, nil).Once()
	f.items.EXPECT().
		Update(ctx, entity.WishlistItem{BaseEntity: item.BaseEntity, ProfileID: profileID, CardID: cardID, WantedQuantity: 5}).
		Return(item, nil).
		Once()

	require.NoError(t, f.svc.SetQuantity(ctx, dto.WishlistItemRequest{ProfileID: profileID, CardID: cardID, Quantity: 5}))
}

func TestWishlistService_SetQuantity_RefusesANonPositiveQuantity(t *testing.T) {
	ctx := context.Background()
	f := newWishlistFixture(t)

	require.Error(t, f.svc.SetQuantity(ctx, dto.WishlistItemRequest{ProfileID: uuid.New(), CardID: uuid.New(), Quantity: 0}))
}

func TestWishlistService_SetQuantity_RefusesAnUnknownCard(t *testing.T) {
	ctx := context.Background()
	cardID := uuid.New()
	wantErr := errors.New("not found")
	f := newWishlistFixture(t)
	f.cards.EXPECT().RequireCard(ctx, cardID).Return(wantErr).Once()

	require.ErrorIs(t, f.svc.SetQuantity(ctx, dto.WishlistItemRequest{ProfileID: uuid.New(), CardID: cardID, Quantity: 1}), wantErr)
}

func TestWishlistService_SetQuantity_AnonymousIsRefused(t *testing.T) {
	ctx := context.Background()
	cardID := uuid.New()
	f := newWishlistFixture(t)
	f.cards.EXPECT().RequireCard(ctx, cardID).Return(nil).Once()

	require.Error(t, f.svc.SetQuantity(ctx, dto.WishlistItemRequest{CardID: cardID, Quantity: 1}))
}

func TestWishlistService_Remove(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	cardID := uuid.New()
	item := entity.WishlistItem{BaseEntity: baseEntity(uuid.New()), ProfileID: profileID, CardID: cardID, WantedQuantity: 2}
	f := newWishlistFixture(t)

	f.expectLock(ctx, profileID)
	f.items.EXPECT().
		FindFirst(ctx, crud.Specification[entity.WishlistItem]{
			Model:     entity.WishlistItem{ProfileID: profileID, CardID: cardID},
			ForUpdate: true,
		}).
		Return(item, nil).
		Once()
	f.items.EXPECT().Delete(ctx, item).Return(nil).Once()

	require.NoError(t, f.svc.Remove(ctx, dto.WishlistItemLookup{ProfileID: profileID, CardID: cardID}))
}

func TestWishlistService_Remove_RefusesAMissingItem(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	f := newWishlistFixture(t)

	f.expectLock(ctx, profileID)
	f.items.EXPECT().FindFirst(ctx, mock.Anything).Return(entity.WishlistItem{}, nil).Once()

	require.Error(t, f.svc.Remove(ctx, dto.WishlistItemLookup{ProfileID: profileID, CardID: uuid.New()}))
}

// The zero Card must not match an arbitrary item: a nil CardID drops the card
// condition from the query, so the service refuses it before the lookup.
func TestWishlistService_Remove_RefusesTheZeroCard(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	f := newWishlistFixture(t)

	f.expectLock(ctx, profileID)

	require.Error(t, f.svc.Remove(ctx, dto.WishlistItemLookup{ProfileID: profileID, CardID: uuid.Nil}))
}

func TestWishlistService_ClearFulfilled(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	f := newWishlistFixture(t)

	f.expectLock(ctx, profileID)
	f.items.EXPECT().DeleteFulfilled(ctx, profileID).Return(3, nil).Once()

	got, err := f.svc.ClearFulfilled(ctx, dto.WishlistLookup{ProfileID: profileID})

	require.NoError(t, err)
	assert.Equal(t, dto.WishlistClearResult{Removed: 3}, got)
}

func TestWishlistService_ClearFulfilled_AnonymousIsRefused(t *testing.T) {
	ctx := context.Background()
	f := newWishlistFixture(t)

	_, err := f.svc.ClearFulfilled(ctx, dto.WishlistLookup{})

	require.Error(t, err)
}

func TestWishlistService_PreviewPreset(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	setID := uuid.New()
	cardID := uuid.New()
	f := newWishlistFixture(t)

	f.items.EXPECT().
		FindOwnedNoneCards(ctx, profileID, []uuid.UUID{setID}, repository.CardFilter{RarityIDs: []uuid.UUID{cardID}}).
		Return([]repository.CardResult{{ID: cardID}}, nil).
		Once()

	result, err := f.svc.PreviewPreset(ctx, dto.WishlistPresetRequest{
		ProfileID:       profileID,
		ExpansionSetIDs: []uuid.UUID{setID},
		Filter:          dto.CardFilter{RarityIDs: []uuid.UUID{cardID}},
		Quantity:        1,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Count)
	require.Len(t, result.Cards, 1)
}

func TestWishlistService_PreviewPreset_RefusesNonPositiveQuantity(t *testing.T) {
	ctx := context.Background()
	f := newWishlistFixture(t)

	_, err := f.svc.PreviewPreset(ctx, dto.WishlistPresetRequest{
		ProfileID: uuid.New(),
		Quantity:  0,
	})

	require.Error(t, err)
}

func TestWishlistService_PreviewPreset_RefusesEmptyExpansionSets(t *testing.T) {
	f := newWishlistFixture(t)

	_, err := f.svc.PreviewPreset(context.Background(), dto.WishlistPresetRequest{
		ProfileID: uuid.New(),
		Quantity:  1,
	})

	require.Error(t, err)
}

func TestWishlistService_PreviewPreset_AnonymousIsRefused(t *testing.T) {
	ctx := context.Background()
	f := newWishlistFixture(t)

	_, err := f.svc.PreviewPreset(ctx, dto.WishlistPresetRequest{
		Quantity: 1,
	})

	require.Error(t, err)
}

func TestWishlistService_ApplyPreset(t *testing.T) {
	ctx := context.Background()
	profileID := uuid.New()
	setID := uuid.New()
	cardID := uuid.New()
	f := newWishlistFixture(t)

	f.expectLock(ctx, profileID)
	f.items.EXPECT().
		FindOwnedNoneCards(ctx, profileID, []uuid.UUID{setID}, repository.CardFilter{RarityIDs: []uuid.UUID{cardID}}).
		Return([]repository.CardResult{{ID: cardID}}, nil).
		Once()
	f.items.EXPECT().
		Insert(ctx, entity.WishlistItem{ProfileID: profileID, CardID: cardID, WantedQuantity: 2}).
		Return(entity.WishlistItem{BaseEntity: baseEntity(uuid.New())}, nil).
		Once()

	result, err := f.svc.ApplyPreset(ctx, dto.WishlistPresetRequest{
		ProfileID:       profileID,
		ExpansionSetIDs: []uuid.UUID{setID},
		Filter:          dto.CardFilter{RarityIDs: []uuid.UUID{cardID}},
		Quantity:        2,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Count)
	assert.Len(t, result.Cards, 1)
}

func TestWishlistService_ApplyPreset_RefusesNonPositiveQuantity(t *testing.T) {
	ctx := context.Background()
	f := newWishlistFixture(t)

	_, err := f.svc.ApplyPreset(ctx, dto.WishlistPresetRequest{
		ProfileID: uuid.New(),
		Quantity:  0,
	})

	require.Error(t, err)
}

func TestWishlistService_ApplyPreset_RefusesEmptyExpansionSets(t *testing.T) {
	f := newWishlistFixture(t)

	_, err := f.svc.ApplyPreset(context.Background(), dto.WishlistPresetRequest{
		ProfileID: uuid.New(),
		Quantity:  1,
	})

	require.Error(t, err)
}

func TestWishlistService_ApplyPreset_AnonymousIsRefused(t *testing.T) {
	ctx := context.Background()
	f := newWishlistFixture(t)

	_, err := f.svc.ApplyPreset(ctx, dto.WishlistPresetRequest{
		Quantity: 1,
	})

	require.Error(t, err)
}
