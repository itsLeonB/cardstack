package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

type cardAdminFixture struct {
	ctx      context.Context
	set      entity.ExpansionSet
	cards    *mocks.MockRepository[entity.Card]
	sets     *mocks.MockRepository[entity.ExpansionSet]
	rarities *mocks.MockRepository[entity.Rarity]
	entries  *mocks.MockRepository[entity.InventoryEntry]
	store    *mocks.MockObjectStore
	svc      CardAdminService
}

// newCardAdminFixture's Transactor mock runs the transaction body inline.
func newCardAdminFixture(t *testing.T) cardAdminFixture {
	t.Helper()
	f := cardAdminFixture{
		ctx: context.Background(),
		set: entity.ExpansionSet{BaseEntity: baseEntity(uuid.New()), GameID: uuid.New(), Code: "SM-P"},
	}
	f.cards = mocks.NewMockRepository[entity.Card](t)
	f.sets = mocks.NewMockRepository[entity.ExpansionSet](t)
	f.rarities = mocks.NewMockRepository[entity.Rarity](t)
	f.entries = mocks.NewMockRepository[entity.InventoryEntry](t)
	f.store = mocks.NewMockObjectStore(t)
	transactor := mocks.NewMockTransactor(t)
	transactor.EXPECT().WithinTransaction(f.ctx, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Maybe()
	f.svc = NewCardAdminService(transactor, f.cards, f.sets, f.rarities, f.entries, f.store, testImages)
	return f
}

func (f cardAdminFixture) expectLockedSet() {
	f.sets.EXPECT().FindFirst(f.ctx, crud.Specification[entity.ExpansionSet]{
		Model:     entity.ExpansionSet{BaseEntity: baseEntity(f.set.ID)},
		ForUpdate: true,
	}).Return(f.set, nil).Once()
}

func (f cardAdminFixture) createReq() dto.AdminCardCreate {
	return dto.AdminCardCreate{ExpansionSetID: f.set.ID, LocalID: "001", Name: "Pikachu", Category: "Pokémon"}
}

func (f cardAdminFixture) expectCardAt(localID string, found entity.Card) {
	f.cards.EXPECT().FindFirst(f.ctx, crud.Specification[entity.Card]{
		Model: entity.Card{ExpansionSetID: f.set.ID, LocalID: localID},
	}).Return(found, nil).Once()
}

func (f cardAdminFixture) expectRarityByName(found entity.Rarity) {
	f.rarities.EXPECT().FindFirst(f.ctx, crud.Specification[entity.Rarity]{
		Model: entity.Rarity{GameID: f.set.GameID, Name: DefaultRarityName},
	}).Return(found, nil).Once()
}

func (f cardAdminFixture) expectRarityByID(id uuid.UUID, found entity.Rarity) {
	f.rarities.EXPECT().FindFirst(f.ctx, crud.Specification[entity.Rarity]{
		Model: entity.Rarity{BaseEntity: baseEntity(id), GameID: f.set.GameID},
	}).Return(found, nil).Once()
}

func unlockedCardSpec(id uuid.UUID) crud.Specification[entity.Card] {
	return crud.Specification[entity.Card]{Model: entity.Card{BaseEntity: baseEntity(id)}}
}

func lockedCardSpec(id uuid.UUID) crud.Specification[entity.Card] {
	return crud.Specification[entity.Card]{Model: entity.Card{BaseEntity: baseEntity(id)}, ForUpdate: true}
}

func TestCardAdminService_CreateCard_TagOnlyOnTrainer(t *testing.T) {
	f := newCardAdminFixture(t)
	req := f.createReq()
	req.Tags = []string{"Item"}

	_, err := f.svc.CreateCard(f.ctx, req)

	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func TestCardAdminService_CreateCard_UnknownSetIsRefused(t *testing.T) {
	f := newCardAdminFixture(t)
	req := f.createReq()
	f.sets.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entity.ExpansionSet{}, nil).Once()

	_, err := f.svc.CreateCard(f.ctx, req)
	requireStatus(t, err, http.StatusUnprocessableEntity)

	req.ExpansionSetID = uuid.Nil
	_, err = f.svc.CreateCard(f.ctx, req)
	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func TestCardAdminService_CreateCard_DefaultsToTanpaTandaAndIsCurated(t *testing.T) {
	f := newCardAdminFixture(t)
	promo := entity.Rarity{BaseEntity: baseEntity(uuid.New()), Name: DefaultRarityName}
	f.expectLockedSet()
	f.expectCardAt("001", entity.Card{})
	f.expectRarityByName(promo)
	want := entity.Card{
		ExpansionSetID: f.set.ID, LocalID: "001", Name: "Pikachu", Category: "Pokémon",
		Tags: datatypes.JSONSlice[string]{}, RarityID: promo.ID, Attributes: datatypes.JSONMap{},
		CreatedBy: entity.SourceAdmin, UpdatedBy: entity.SourceAdmin,
	}
	inserted := want
	inserted.BaseEntity = baseEntity(uuid.New())
	f.cards.EXPECT().Insert(f.ctx, want).Return(inserted, nil).Once()

	got, err := f.svc.CreateCard(f.ctx, f.createReq())

	require.NoError(t, err)
	assert.Equal(t, inserted.ID, got.ID)
	assert.Equal(t, promo.ID, got.RarityID)
}

func TestCardAdminService_CreateCard_TrainerKeepsItsTagsAndChosenRarity(t *testing.T) {
	f := newCardAdminFixture(t)
	chosen := entity.Rarity{BaseEntity: baseEntity(uuid.New()), Name: "Double Rare"}
	req := f.createReq()
	req.Category, req.Tags, req.RarityID, req.Illustrator = "Trainer", []string{"Item", "Supporter"}, chosen.ID, "Someone"
	f.expectLockedSet()
	f.expectCardAt("001", entity.Card{})
	f.expectRarityByID(chosen.ID, chosen)
	f.cards.EXPECT().Insert(f.ctx, entity.Card{
		ExpansionSetID: f.set.ID, LocalID: "001", Name: "Pikachu", Category: "Trainer", Illustrator: "Someone",
		Tags: datatypes.JSONSlice[string]{"Item", "Supporter"}, RarityID: chosen.ID, Attributes: datatypes.JSONMap{},
		CreatedBy: entity.SourceAdmin, UpdatedBy: entity.SourceAdmin,
	}).Return(entity.Card{BaseEntity: baseEntity(uuid.New()), Tags: datatypes.JSONSlice[string]{"Item", "Supporter"}}, nil).Once()

	got, err := f.svc.CreateCard(f.ctx, req)

	require.NoError(t, err)
	assert.Equal(t, []string{"Item", "Supporter"}, got.Tags)
}

func TestCardAdminService_CreateCard_AnExistingNumberIsAConflictAndWritesNothing(t *testing.T) {
	f := newCardAdminFixture(t)
	f.expectLockedSet()
	f.expectCardAt("001", storedCard(f))

	_, err := f.svc.CreateCard(f.ctx, f.createReq())

	requireStatus(t, err, http.StatusConflict)
}

func TestCardAdminService_CreateCard_RarityOfAnotherGameIsRefused(t *testing.T) {
	f := newCardAdminFixture(t)
	req := f.createReq()
	req.RarityID = uuid.New()
	f.expectLockedSet()
	f.expectCardAt("001", entity.Card{})
	f.expectRarityByID(req.RarityID, entity.Rarity{})

	_, err := f.svc.CreateCard(f.ctx, req)

	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func TestCardAdminService_CreateCard_WithoutAPromoRarityIsRefused(t *testing.T) {
	f := newCardAdminFixture(t)
	f.expectLockedSet()
	f.expectCardAt("001", entity.Card{})
	f.expectRarityByName(entity.Rarity{})

	_, err := f.svc.CreateCard(f.ctx, f.createReq())

	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func storedCard(f cardAdminFixture) entity.Card {
	return entity.Card{
		BaseEntity: baseEntity(uuid.New()), ExpansionSetID: f.set.ID, LocalID: "001", Name: "Typo", Category: "Trainer",
		Illustrator: "Arita", Tags: datatypes.JSONSlice[string]{"Item"}, RarityID: uuid.New(), ImageKey: "cards/keep",
	}
}

func (f cardAdminFixture) updateReq(stored entity.Card, rarityID uuid.UUID) dto.AdminCardUpdate {
	return dto.AdminCardUpdate{ID: stored.ID, Name: "Pikachu", Category: "Pokémon", RarityID: rarityID}
}

// expectUpdateReads sets up the locked Card read and the unlocked read of its set.
func (f cardAdminFixture) expectUpdateReads(stored entity.Card) {
	f.cards.EXPECT().FindFirst(f.ctx, lockedCardSpec(stored.ID)).Return(stored, nil).Once()
	f.sets.EXPECT().FindFirst(f.ctx, crud.Specification[entity.ExpansionSet]{
		Model: entity.ExpansionSet{BaseEntity: baseEntity(f.set.ID)},
	}).Return(f.set, nil).Once()
}

func TestCardAdminService_UpdateCard_ReplacesEveryFieldWithTheRequests(t *testing.T) {
	f := newCardAdminFixture(t)
	stored := storedCard(f)
	chosen := entity.Rarity{BaseEntity: baseEntity(uuid.New()), Name: "Double Rare"}
	req := f.updateReq(stored, chosen.ID)
	req.Tags = []string{"Item"}
	req.Category = "Trainer"
	f.expectUpdateReads(stored)
	f.expectRarityByID(chosen.ID, chosen)
	// The illustrator is not in the request, so it is cleared, not kept.
	want := stored
	want.Name, want.Category, want.Tags, want.RarityID, want.Illustrator, want.UpdatedBy =
		"Pikachu", "Trainer", datatypes.JSONSlice[string]{"Item"}, chosen.ID, "", entity.SourceAdmin
	f.cards.EXPECT().Update(f.ctx, want).Return(want, nil).Once()

	got, err := f.svc.UpdateCard(f.ctx, req)

	require.NoError(t, err)
	assert.Equal(t, chosen.ID, got.RarityID)
	assert.Empty(t, got.Illustrator)
	assert.Equal(t, []string{"Item"}, got.Tags)
	assert.Equal(t, testImageBase+"/cards/keep", got.ImageURL, "the hosted image stays")
}

func TestCardAdminService_UpdateCard_NoTagsClearsTheStoredOnes(t *testing.T) {
	f := newCardAdminFixture(t)
	stored := storedCard(f)
	chosen := entity.Rarity{BaseEntity: baseEntity(uuid.New())}
	f.expectUpdateReads(stored)
	f.expectRarityByID(chosen.ID, chosen)
	f.cards.EXPECT().Update(f.ctx, mock.MatchedBy(func(c entity.Card) bool {
		return c.Tags != nil && len(c.Tags) == 0
	})).Return(stored, nil).Once()

	_, err := f.svc.UpdateCard(f.ctx, f.updateReq(stored, chosen.ID))

	require.NoError(t, err)
}

func TestCardAdminService_UpdateCard_RarityIsRequired(t *testing.T) {
	f := newCardAdminFixture(t)

	// Nothing is read: an omitted rarity is refused, never defaulted or kept.
	_, err := f.svc.UpdateCard(f.ctx, f.updateReq(storedCard(f), uuid.Nil))

	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func TestCardAdminService_UpdateCard_TagOnlyOnTrainer(t *testing.T) {
	f := newCardAdminFixture(t)
	req := f.updateReq(storedCard(f), uuid.New())
	req.Tags = []string{"Item"}

	_, err := f.svc.UpdateCard(f.ctx, req)

	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func TestCardAdminService_UpdateCard_UnknownCardIsNotFound(t *testing.T) {
	f := newCardAdminFixture(t)
	missing := uuid.New()
	f.cards.EXPECT().FindFirst(f.ctx, lockedCardSpec(missing)).Return(entity.Card{}, nil).Once()

	_, err := f.svc.UpdateCard(f.ctx, dto.AdminCardUpdate{ID: missing, Name: "X", Category: "Pokémon", RarityID: uuid.New()})

	requireStatus(t, err, http.StatusNotFound)
}

func TestCardAdminService_UpdateCard_UnknownRarityWritesNothing(t *testing.T) {
	f := newCardAdminFixture(t)
	stored := storedCard(f)
	rarityID := uuid.New()
	f.expectUpdateReads(stored)
	f.expectRarityByID(rarityID, entity.Rarity{})

	_, err := f.svc.UpdateCard(f.ctx, f.updateReq(stored, rarityID))

	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func TestCardAdminService_RequestImageUpload(t *testing.T) {
	f := newCardAdminFixture(t)
	card := entity.Card{BaseEntity: baseEntity(uuid.New())}
	expires := time.Now().Add(cardImageUploadTTL)
	f.cards.EXPECT().FindFirst(f.ctx, unlockedCardSpec(card.ID)).Return(card, nil).Once()
	f.store.EXPECT().PresignPut(f.ctx, "cards/"+card.ID.String(), "image/webp", cardImageUploadTTL).
		Return("https://bucket.test/signed", expires, nil).Once()

	got, err := f.svc.RequestImageUpload(f.ctx, dto.CardImageUploadRequest{CardID: card.ID, ContentType: "image/webp"})

	require.NoError(t, err)
	assert.Equal(t, dto.CardImageUpload{URL: "https://bucket.test/signed", ContentType: "image/webp", ExpiresAt: expires}, got)
}

func TestCardAdminService_UnknownCardIsNotFoundBeforeAnyBucketCall(t *testing.T) {
	f := newCardAdminFixture(t)
	missing := uuid.New()
	f.cards.EXPECT().FindFirst(f.ctx, unlockedCardSpec(missing)).Return(entity.Card{}, nil).Times(2)

	_, err := f.svc.RequestImageUpload(f.ctx, dto.CardImageUploadRequest{CardID: missing, ContentType: "image/png"})
	requireStatus(t, err, http.StatusNotFound)
	_, err = f.svc.ConfirmImage(f.ctx, dto.CardImageConfirm{CardID: missing})
	requireStatus(t, err, http.StatusNotFound)
	_, err = f.svc.ConfirmImage(f.ctx, dto.CardImageConfirm{CardID: uuid.Nil})
	requireStatus(t, err, http.StatusNotFound)
}

// expectConfirmReads sets up the unlocked existence check and the head read.
func (f cardAdminFixture) expectConfirmReads(card entity.Card, head []byte, readErr error) string {
	key := "cards/" + card.ID.String()
	f.cards.EXPECT().FindFirst(f.ctx, unlockedCardSpec(card.ID)).Return(card, nil).Once()
	f.store.EXPECT().ReadHead(f.ctx, key, signatureBytes).Return(head, readErr).Once()
	return key
}

func (f cardAdminFixture) expectLockedCard(card entity.Card) {
	f.cards.EXPECT().FindFirst(f.ctx, lockedCardSpec(card.ID)).Return(card, nil).Once()
}

func TestCardAdminService_ConfirmImage_AcceptsJPEGPNGAndWebP(t *testing.T) {
	heads := map[string][]byte{
		"jpeg": {0xff, 0xd8, 0xff, 0xe0, 0, 0, 0, 0, 0, 0, 0, 0},
		"png":  {0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0},
		"webp": {'R', 'I', 'F', 'F', 1, 2, 3, 4, 'W', 'E', 'B', 'P'},
	}
	for name, head := range heads {
		t.Run(name, func(t *testing.T) {
			f := newCardAdminFixture(t)
			// Last updated by ingestion, so confirming makes the Admin the updater.
			card := entity.Card{BaseEntity: baseEntity(uuid.New()), ExpansionSetID: f.set.ID, Name: "X", CreatedBy: entity.SourcePokemonAsia, UpdatedBy: entity.SourcePokemonAsia}
			key := f.expectConfirmReads(card, head, nil)
			f.expectLockedCard(card)
			want := card
			want.ImageKey, want.UpdatedBy = key, entity.SourceAdmin
			f.cards.EXPECT().Update(f.ctx, want).Return(want, nil).Once()

			got, err := f.svc.ConfirmImage(f.ctx, dto.CardImageConfirm{CardID: card.ID})

			require.NoError(t, err)
			assert.Equal(t, testImageBase+"/"+key, got.ImageURL)
		})
	}
}

func TestCardAdminService_ConfirmImage_RefusesAnythingElseAndDeletesTheObject(t *testing.T) {
	heads := map[string][]byte{
		"gif":                  []byte("GIF89a......"),
		"empty":                {},
		"truncated png":        {0x89, 'P', 'N', 'G'},
		"riff but not webp":    {'R', 'I', 'F', 'F', 1, 2, 3, 4, 'W', 'A', 'V', 'E'},
		"webp cut before form": {'R', 'I', 'F', 'F', 1, 2, 3, 4, 'W'},
	}
	for name, head := range heads {
		t.Run(name, func(t *testing.T) {
			f := newCardAdminFixture(t)
			card := entity.Card{BaseEntity: baseEntity(uuid.New()), ImageKey: "cards/old"}
			key := f.expectConfirmReads(card, head, nil)
			f.store.EXPECT().Delete(f.ctx, key).Return(nil).Once()
			f.expectLockedCard(card)
			// A re-upload replaced the object under the old key, so the key is cleared.
			cleared := card
			cleared.ImageKey = ""
			f.cards.EXPECT().Update(f.ctx, cleared).Return(cleared, nil).Once()

			_, err := f.svc.ConfirmImage(f.ctx, dto.CardImageConfirm{CardID: card.ID})

			requireStatus(t, err, http.StatusUnprocessableEntity)
		})
	}
}

func TestCardAdminService_ConfirmImage_RefusalStillHoldsWhenTheObjectCannotBeDeleted(t *testing.T) {
	f := newCardAdminFixture(t)
	card := entity.Card{BaseEntity: baseEntity(uuid.New())}
	key := f.expectConfirmReads(card, []byte("GIF89a......"), nil)
	f.store.EXPECT().Delete(f.ctx, key).Return(errors.New("bucket down")).Once()
	f.expectLockedCard(card)
	f.cards.EXPECT().Update(f.ctx, card).Return(card, nil).Once()

	_, err := f.svc.ConfirmImage(f.ctx, dto.CardImageConfirm{CardID: card.ID})

	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func TestCardAdminService_ConfirmImage_NothingUploadedWritesNothing(t *testing.T) {
	f := newCardAdminFixture(t)
	card := entity.Card{BaseEntity: baseEntity(uuid.New())}
	f.expectConfirmReads(card, nil, objectstore.ErrNotFound)

	_, err := f.svc.ConfirmImage(f.ctx, dto.CardImageConfirm{CardID: card.ID})

	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func TestCardAdminService_ConfirmImage_BucketFailureIsReturnedUntouched(t *testing.T) {
	f := newCardAdminFixture(t)
	card := entity.Card{BaseEntity: baseEntity(uuid.New())}
	boom := errors.New("bucket down")
	f.expectConfirmReads(card, nil, boom)

	_, err := f.svc.ConfirmImage(f.ctx, dto.CardImageConfirm{CardID: card.ID})

	assert.ErrorIs(t, err, boom)
}

func TestCardAdminService_ConfirmImage_CardDeletedDuringTheCheckIsNotFound(t *testing.T) {
	f := newCardAdminFixture(t)
	card := entity.Card{BaseEntity: baseEntity(uuid.New())}
	f.expectConfirmReads(card, []byte{0xff, 0xd8, 0xff, 0xe0, 0, 0, 0, 0, 0, 0, 0, 0}, nil)
	f.cards.EXPECT().FindFirst(f.ctx, lockedCardSpec(card.ID)).Return(entity.Card{}, nil).Once()

	_, err := f.svc.ConfirmImage(f.ctx, dto.CardImageConfirm{CardID: card.ID})

	requireStatus(t, err, http.StatusNotFound)
}

func TestCardAdminService_GetCard_BlankIdentityIsNotFoundWithoutAQuery(t *testing.T) {
	f := newCardAdminFixture(t)

	_, err := f.svc.GetCard(f.ctx, dto.CardLookup{LocalID: "001"})
	requireStatus(t, err, http.StatusNotFound)
	_, err = f.svc.GetCard(f.ctx, dto.CardLookup{ExpansionSetID: f.set.ID})
	requireStatus(t, err, http.StatusNotFound)
}

func TestCardAdminService_DeleteCard(t *testing.T) {
	t.Run("a nil id is not found without a query", func(t *testing.T) {
		f := newCardAdminFixture(t)
		requireStatus(t, f.svc.DeleteCard(f.ctx, dto.CardDelete{}), http.StatusNotFound)
	})

	t.Run("unknown card", func(t *testing.T) {
		f := newCardAdminFixture(t)
		missing := uuid.New()
		f.cards.EXPECT().FindFirst(f.ctx, lockedCardSpec(missing)).Return(entity.Card{}, nil).Once()
		requireStatus(t, f.svc.DeleteCard(f.ctx, dto.CardDelete{ID: missing}), http.StatusNotFound)
	})

	t.Run("a collected card is a conflict and keeps its object", func(t *testing.T) {
		f := newCardAdminFixture(t)
		card := entity.Card{BaseEntity: baseEntity(uuid.New())}
		f.cards.EXPECT().FindFirst(f.ctx, lockedCardSpec(card.ID)).Return(card, nil).Once()
		f.entries.EXPECT().FindFirst(f.ctx, crud.Specification[entity.InventoryEntry]{Model: entity.InventoryEntry{CardID: card.ID}}).
			Return(entity.InventoryEntry{BaseEntity: baseEntity(uuid.New())}, nil).Once()
		requireStatus(t, f.svc.DeleteCard(f.ctx, dto.CardDelete{ID: card.ID}), http.StatusConflict)
	})

	t.Run("deletes the row, then the object derived from its id", func(t *testing.T) {
		f := newCardAdminFixture(t)
		card := entity.Card{BaseEntity: baseEntity(uuid.New())}
		f.cards.EXPECT().FindFirst(f.ctx, lockedCardSpec(card.ID)).Return(card, nil).Once()
		f.entries.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entity.InventoryEntry{}, nil).Once()
		f.cards.EXPECT().Delete(f.ctx, card).Return(nil).Once()
		f.store.EXPECT().Delete(f.ctx, "cards/"+card.ID.String()).Return(errors.New("bucket down")).Once()

		assert.NoError(t, f.svc.DeleteCard(f.ctx, dto.CardDelete{ID: card.ID}), "a failed object delete is logged, not returned")
	})
}
