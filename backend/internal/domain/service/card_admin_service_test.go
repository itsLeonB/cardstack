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

func (f cardAdminFixture) upsertReq() dto.AdminCardUpsert {
	return dto.AdminCardUpsert{ExpansionSetID: f.set.ID, LocalID: "001", Name: "Pikachu", Category: "Pokémon"}
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

func TestCardAdminService_UpsertCard_TagOnlyOnTrainer(t *testing.T) {
	f := newCardAdminFixture(t)
	req := f.upsertReq()
	req.Tag = "Item"

	_, err := f.svc.UpsertCard(f.ctx, req)

	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func TestCardAdminService_UpsertCard_UnknownSetIsRefused(t *testing.T) {
	f := newCardAdminFixture(t)
	req := f.upsertReq()
	f.sets.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entity.ExpansionSet{}, nil).Once()

	_, err := f.svc.UpsertCard(f.ctx, req)
	requireStatus(t, err, http.StatusUnprocessableEntity)

	req.ExpansionSetID = uuid.Nil
	_, err = f.svc.UpsertCard(f.ctx, req)
	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func TestCardAdminService_UpsertCard_NewCardDefaultsToTanpaTandaAndIsCurated(t *testing.T) {
	f := newCardAdminFixture(t)
	promo := entity.Rarity{BaseEntity: baseEntity(uuid.New()), Name: DefaultRarityName}
	f.expectLockedSet()
	f.expectCardAt("001", entity.Card{})
	f.expectRarityByName(promo)
	want := entity.Card{
		ExpansionSetID: f.set.ID, LocalID: "001", Name: "Pikachu", Category: "Pokémon",
		Tags: datatypes.JSONSlice[string]{}, RarityID: promo.ID, Attributes: datatypes.JSONMap{}, Curated: true,
	}
	inserted := want
	inserted.BaseEntity = baseEntity(uuid.New())
	f.cards.EXPECT().Insert(f.ctx, want).Return(inserted, nil).Once()

	got, err := f.svc.UpsertCard(f.ctx, f.upsertReq())

	require.NoError(t, err)
	assert.Equal(t, inserted.ID, got.ID)
	assert.Equal(t, promo.ID, got.RarityID)
}

func TestCardAdminService_UpsertCard_NewTrainerCardKeepsItsTagAndChosenRarity(t *testing.T) {
	f := newCardAdminFixture(t)
	chosen := entity.Rarity{BaseEntity: baseEntity(uuid.New()), Name: "Double Rare"}
	req := f.upsertReq()
	req.Category, req.Tag, req.RarityID, req.Illustrator = "Trainer", "Supporter", chosen.ID, "Someone"
	f.expectLockedSet()
	f.expectCardAt("001", entity.Card{})
	f.expectRarityByID(chosen.ID, chosen)
	f.cards.EXPECT().Insert(f.ctx, entity.Card{
		ExpansionSetID: f.set.ID, LocalID: "001", Name: "Pikachu", Category: "Trainer", Illustrator: "Someone",
		Tags: datatypes.JSONSlice[string]{"Supporter"}, RarityID: chosen.ID, Attributes: datatypes.JSONMap{}, Curated: true,
	}).Return(entity.Card{BaseEntity: baseEntity(uuid.New()), Tags: datatypes.JSONSlice[string]{"Supporter"}}, nil).Once()

	got, err := f.svc.UpsertCard(f.ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "Supporter", got.Tag)
}

func TestCardAdminService_UpsertCard_NewCardWithRarityOfAnotherGameIsRefused(t *testing.T) {
	f := newCardAdminFixture(t)
	req := f.upsertReq()
	req.RarityID = uuid.New()
	f.expectLockedSet()
	f.expectCardAt("001", entity.Card{})
	f.expectRarityByID(req.RarityID, entity.Rarity{})

	_, err := f.svc.UpsertCard(f.ctx, req)

	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func TestCardAdminService_UpsertCard_NewCardWithoutAPromoRarityIsRefused(t *testing.T) {
	f := newCardAdminFixture(t)
	f.expectLockedSet()
	f.expectCardAt("001", entity.Card{})
	f.expectRarityByName(entity.Rarity{})

	_, err := f.svc.UpsertCard(f.ctx, f.upsertReq())

	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func storedCard(f cardAdminFixture) entity.Card {
	return entity.Card{
		BaseEntity: baseEntity(uuid.New()), ExpansionSetID: f.set.ID, LocalID: "001", Name: "Typo", Category: "Trainer",
		Illustrator: "Arita", Tags: datatypes.JSONSlice[string]{"Item"}, RarityID: uuid.New(), ImageKey: "cards/keep",
	}
}

func TestCardAdminService_UpsertCard_EditKeepsOmittedRarityAndIllustrator(t *testing.T) {
	f := newCardAdminFixture(t)
	stored := storedCard(f)
	f.expectLockedSet()
	f.expectCardAt("001", stored)
	// No rarity lookup is expected: an omitted rarity is not resolved.
	want := stored
	want.Name, want.Category, want.Tags, want.Curated = "Pikachu", "Pokémon", datatypes.JSONSlice[string]{}, true
	f.cards.EXPECT().Update(f.ctx, want).Return(want, nil).Once()

	got, err := f.svc.UpsertCard(f.ctx, f.upsertReq())

	require.NoError(t, err)
	assert.Equal(t, stored.RarityID, got.RarityID)
	assert.Equal(t, "Arita", got.Illustrator)
	assert.Empty(t, got.Tag, "the tag follows the request, not the stored card")
	assert.Equal(t, testImageBase+"/cards/keep", got.ImageURL)
}

func TestCardAdminService_UpsertCard_EditReplacesAGivenRarityAndIllustrator(t *testing.T) {
	f := newCardAdminFixture(t)
	stored := storedCard(f)
	chosen := entity.Rarity{BaseEntity: baseEntity(uuid.New()), Name: "Double Rare"}
	req := f.upsertReq()
	req.RarityID, req.Illustrator = chosen.ID, "Someone"
	f.expectLockedSet()
	f.expectCardAt("001", stored)
	f.expectRarityByID(chosen.ID, chosen)
	want := stored
	want.Name, want.Category, want.Tags, want.Curated = "Pikachu", "Pokémon", datatypes.JSONSlice[string]{}, true
	want.RarityID, want.Illustrator = chosen.ID, "Someone"
	f.cards.EXPECT().Update(f.ctx, want).Return(want, nil).Once()

	got, err := f.svc.UpsertCard(f.ctx, req)

	require.NoError(t, err)
	assert.Equal(t, chosen.ID, got.RarityID)
	assert.Equal(t, "Someone", got.Illustrator)
}

func TestCardAdminService_UpsertCard_EditWithUnknownRarityWritesNothing(t *testing.T) {
	f := newCardAdminFixture(t)
	req := f.upsertReq()
	req.RarityID = uuid.New()
	f.expectLockedSet()
	f.expectCardAt("001", storedCard(f))
	f.expectRarityByID(req.RarityID, entity.Rarity{})

	_, err := f.svc.UpsertCard(f.ctx, req)

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
			// Not Curated, as a Card ingestion wrote: confirming marks it.
			card := entity.Card{BaseEntity: baseEntity(uuid.New()), ExpansionSetID: f.set.ID, Name: "X"}
			key := f.expectConfirmReads(card, head, nil)
			f.expectLockedCard(card)
			want := card
			want.ImageKey, want.Curated = key, true
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
	deleteSpec := func(f cardAdminFixture) crud.Specification[entity.Card] {
		return crud.Specification[entity.Card]{Model: entity.Card{ExpansionSetID: f.set.ID, LocalID: "001"}, ForUpdate: true}
	}
	req := func(f cardAdminFixture) dto.CardDelete {
		return dto.CardDelete{ExpansionSetID: f.set.ID, LocalID: "001"}
	}

	t.Run("a blank identity is not found without a query", func(t *testing.T) {
		f := newCardAdminFixture(t)
		requireStatus(t, f.svc.DeleteCard(f.ctx, dto.CardDelete{ExpansionSetID: uuid.Nil, LocalID: "001"}), http.StatusNotFound)
		requireStatus(t, f.svc.DeleteCard(f.ctx, dto.CardDelete{ExpansionSetID: f.set.ID}), http.StatusNotFound)
	})

	t.Run("unknown card", func(t *testing.T) {
		f := newCardAdminFixture(t)
		f.cards.EXPECT().FindFirst(f.ctx, deleteSpec(f)).Return(entity.Card{}, nil).Once()
		requireStatus(t, f.svc.DeleteCard(f.ctx, req(f)), http.StatusNotFound)
	})

	t.Run("a collected card is a conflict and keeps its object", func(t *testing.T) {
		f := newCardAdminFixture(t)
		card := entity.Card{BaseEntity: baseEntity(uuid.New())}
		f.cards.EXPECT().FindFirst(f.ctx, deleteSpec(f)).Return(card, nil).Once()
		f.entries.EXPECT().FindFirst(f.ctx, crud.Specification[entity.InventoryEntry]{Model: entity.InventoryEntry{CardID: card.ID}}).
			Return(entity.InventoryEntry{BaseEntity: baseEntity(uuid.New())}, nil).Once()
		requireStatus(t, f.svc.DeleteCard(f.ctx, req(f)), http.StatusConflict)
	})

	t.Run("deletes the row, then the object derived from its id", func(t *testing.T) {
		f := newCardAdminFixture(t)
		card := entity.Card{BaseEntity: baseEntity(uuid.New())}
		f.cards.EXPECT().FindFirst(f.ctx, deleteSpec(f)).Return(card, nil).Once()
		f.entries.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entity.InventoryEntry{}, nil).Once()
		f.cards.EXPECT().Delete(f.ctx, card).Return(nil).Once()
		f.store.EXPECT().Delete(f.ctx, "cards/"+card.ID.String()).Return(errors.New("bucket down")).Once()

		assert.NoError(t, f.svc.DeleteCard(f.ctx, req(f)), "a failed object delete is logged, not returned")
	})
}
