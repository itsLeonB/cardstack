package routes

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// adminCatalog is one Expansion Set with two Rarities, one of them the promo
// default "Tanpa Tanda", in a Game of its own so lookups cannot meet another
// test's rows.
type adminCatalog struct {
	set          entity.ExpansionSet
	promoRarity  entity.Rarity
	otherRarity  entity.Rarity
	bearerHeader string
}

func (a testAPI) seedAdminCatalog(t *testing.T) adminCatalog {
	t.Helper()

	s := uuid.NewString()
	game := entity.Game{Slug: "adm-" + s, Name: "Adm " + s}
	require.NoError(t, a.db.Create(&game).Error)
	locale := entity.Locale{Code: "adm-" + s}
	require.NoError(t, a.db.Create(&locale).Error)
	promo := entity.Rarity{GameID: game.ID, Code: "PR-" + s, Name: "Tanpa Tanda"}
	require.NoError(t, a.db.Create(&promo).Error)
	other := entity.Rarity{GameID: game.ID, Code: "RR-" + s, Name: "Double Rare"}
	require.NoError(t, a.db.Create(&other).Error)
	set := entity.ExpansionSet{GameID: game.ID, Code: "ADM-" + s, Name: "Set " + s, LocaleID: locale.ID}
	require.NoError(t, a.db.Create(&set).Error)

	token := a.tokenFor(t, adminEmail)
	return adminCatalog{set: set, promoRarity: promo, otherRarity: other, bearerHeader: bearer(token)}
}

type adminCardBody struct {
	Data struct {
		ID        uuid.UUID `json:"id"`
		LocalID   string    `json:"localId"`
		Name      string    `json:"name"`
		Category  string    `json:"category"`
		Tag       string    `json:"tag"`
		RarityID  uuid.UUID `json:"rarityId"`
		ImageURL  string    `json:"imageUrl"`
		SetID     uuid.UUID `json:"expansionSetId"`
		URL       string    `json:"url"`
		ExpiresAt time.Time `json:"expiresAt"`
	} `json:"data"`
}

func (c adminCatalog) cardBody(localID, name, category string) map[string]any {
	return map[string]any{"expansionSetId": c.set.ID, "localId": localID, "name": name, "category": category}
}

func upsertCard(t *testing.T, api testAPI, c adminCatalog, body map[string]any) (int, adminCardBody) {
	t.Helper()
	resp := api.Put("/admin/cards", c.bearerHeader, body)
	var out adminCardBody
	if resp.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out), resp.Body.String())
	}
	return resp.Code, out
}

func (a testAPI) cardRow(t *testing.T, id uuid.UUID) entity.Card {
	t.Helper()
	var card entity.Card
	require.NoError(t, a.db.First(&card, "id = ?", id).Error)
	return card
}

func TestAdminCards_UpsertCreatesCuratedCardAndReentryUpdatesIt(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	c := api.seedAdminCatalog(t)

	code, created := upsertCard(t, api, c, c.cardBody("001", "Pikachu", "Pokémon"))
	require.Equal(t, http.StatusOK, code)
	row := api.cardRow(t, created.Data.ID)
	assert.True(t, row.Curated, "an admin write marks the card Curated")
	assert.Equal(t, c.promoRarity.ID, row.RarityID, `rarity defaults to "Tanpa Tanda"`)
	assert.Empty(t, row.Tags, "a Pokémon card has no tag")
	assert.Empty(t, row.Illustrator)
	assert.Empty(t, row.ImageKey, "the image key is set only by a confirmed upload")

	again := c.cardBody("001", "Raichu", "Trainer")
	again["tag"] = "Supporter"
	again["rarityId"] = c.otherRarity.ID
	again["illustrator"] = "Someone"
	code, updated := upsertCard(t, api, c, again)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, created.Data.ID, updated.Data.ID, "the same set and number is the same card")
	row = api.cardRow(t, created.Data.ID)
	assert.Equal(t, "Raichu", row.Name)
	assert.Equal(t, []string{"Supporter"}, []string(row.Tags))
	assert.Equal(t, c.otherRarity.ID, row.RarityID)
	assert.Equal(t, "Someone", row.Illustrator)
	assert.True(t, row.Curated)

	var count int64
	require.NoError(t, api.db.Model(&entity.Card{}).Where("expansion_set_id = ?", c.set.ID).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

func TestAdminCards_UpsertWritesAnyCardEvenOneIngestionOwns(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	c := api.seedAdminCatalog(t)
	ingested := entity.Card{ExpansionSetID: c.set.ID, LocalID: "007", Name: "Typo", Category: "Pokémon", RarityID: c.otherRarity.ID, ImageKey: "cards/keep"}
	require.NoError(t, api.db.Create(&ingested).Error)

	code, _ := upsertCard(t, api, c, c.cardBody("007", "Fixed", "Pokémon"))

	require.Equal(t, http.StatusOK, code)
	row := api.cardRow(t, ingested.ID)
	assert.Equal(t, "Fixed", row.Name)
	assert.True(t, row.Curated)
	assert.Equal(t, "cards/keep", row.ImageKey, "a correction keeps the hosted image")
}

func TestAdminCards_UpsertRejectsBadReferences(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	c := api.seedAdminCatalog(t)

	unknownSet := c.cardBody("001", "X", "Pokémon")
	unknownSet["expansionSetId"] = uuid.New()
	unknownRarity := c.cardBody("001", "X", "Pokémon")
	unknownRarity["rarityId"] = uuid.New()
	badCategory := c.cardBody("001", "X", "Weapon")
	tagOnPokemon := c.cardBody("001", "X", "Pokémon")
	tagOnPokemon["tag"] = "Item"
	badTag := c.cardBody("001", "X", "Trainer")
	badTag["tag"] = "Gadget"
	blankNumber := c.cardBody("", "X", "Pokémon")

	for name, body := range map[string]map[string]any{
		"unknown set": unknownSet, "unknown rarity": unknownRarity, "unknown category": badCategory,
		"tag on a Pokémon": tagOnPokemon, "unknown tag": badTag, "empty number": blankNumber,
	} {
		code, _ := upsertCard(t, api, c, body)
		assert.Equal(t, http.StatusUnprocessableEntity, code, name)
	}

	var count int64
	require.NoError(t, api.db.Model(&entity.Card{}).Where("expansion_set_id = ?", c.set.ID).Count(&count).Error)
	assert.Zero(t, count, "a rejected write leaves nothing behind")
}

func TestAdminCards_ImageUploadSignsAServerChosenKey(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	c := api.seedAdminCatalog(t)
	_, card := upsertCard(t, api, c, c.cardBody("001", "Pikachu", "Pokémon"))
	expires := time.Now().Add(5 * time.Minute)
	api.store.EXPECT().
		PresignPut(mock.Anything, "cards/"+card.Data.ID.String(), "image/webp", mock.MatchedBy(func(ttl time.Duration) bool { return ttl > 0 && ttl <= 15*time.Minute })).
		Return("https://bucket.test/signed", expires, nil).Once()

	resp := api.Post("/admin/cards/"+card.Data.ID.String()+"/image-upload", c.bearerHeader, map[string]any{"contentType": "image/webp"})

	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var out adminCardBody
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
	assert.Equal(t, "https://bucket.test/signed", out.Data.URL)

	bad := api.Post("/admin/cards/"+card.Data.ID.String()+"/image-upload", c.bearerHeader, map[string]any{"contentType": "image/gif"})
	assert.Equal(t, http.StatusUnprocessableEntity, bad.Code, "only JPEG, PNG and WebP can be signed")
	missing := api.Post("/admin/cards/"+uuid.NewString()+"/image-upload", c.bearerHeader, map[string]any{"contentType": "image/png"})
	assert.Equal(t, http.StatusNotFound, missing.Code, "no upload for a card that does not exist")
}

func TestAdminCards_ConfirmAcceptsOnlyAnImageSignature(t *testing.T) {
	signatures := map[string][]byte{
		"png":  {0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0},
		"jpeg": {0xff, 0xd8, 0xff, 0xe0, 0, 0, 0, 0, 0, 0, 0, 0},
		"webp": {'R', 'I', 'F', 'F', 1, 2, 3, 4, 'W', 'E', 'B', 'P'},
	}
	for name, head := range signatures {
		t.Run("accepts "+name, func(t *testing.T) {
			api := newTestAPIWithAdmins(t, []string{adminEmail})
			c := api.seedAdminCatalog(t)
			_, card := upsertCard(t, api, c, c.cardBody("001", "Pikachu", "Pokémon"))
			key := "cards/" + card.Data.ID.String()
			api.store.EXPECT().ReadHead(mock.Anything, key, mock.Anything).Return(head, nil).Once()

			resp := api.Post("/admin/cards/"+card.Data.ID.String()+"/image-confirm", c.bearerHeader)

			require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
			row := api.cardRow(t, card.Data.ID)
			assert.Equal(t, key, row.ImageKey)
			assert.True(t, row.Curated)
			var out adminCardBody
			require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
			assert.Equal(t, testImageBase+"/"+key, out.Data.ImageURL)
		})
	}

	t.Run("a wrong signature deletes the object and leaves the key unset", func(t *testing.T) {
		api := newTestAPIWithAdmins(t, []string{adminEmail})
		c := api.seedAdminCatalog(t)
		_, card := upsertCard(t, api, c, c.cardBody("001", "Pikachu", "Pokémon"))
		key := "cards/" + card.Data.ID.String()
		// A re-upload: the card already points at the key the bad file replaced.
		require.NoError(t, api.db.Model(&entity.Card{}).Where("id = ?", card.Data.ID).Update("image_key", key).Error)
		api.store.EXPECT().ReadHead(mock.Anything, key, mock.Anything).Return([]byte("GIF89a......"), nil).Once()
		api.store.EXPECT().Delete(mock.Anything, key).Return(nil).Once()

		resp := api.Post("/admin/cards/"+card.Data.ID.String()+"/image-confirm", c.bearerHeader)

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Empty(t, api.cardRow(t, card.Data.ID).ImageKey)
	})

	t.Run("nothing uploaded is refused without a delete", func(t *testing.T) {
		api := newTestAPIWithAdmins(t, []string{adminEmail})
		c := api.seedAdminCatalog(t)
		_, card := upsertCard(t, api, c, c.cardBody("001", "Pikachu", "Pokémon"))
		api.store.EXPECT().ReadHead(mock.Anything, mock.Anything, mock.Anything).Return(nil, objectstore.ErrNotFound).Once()

		resp := api.Post("/admin/cards/"+card.Data.ID.String()+"/image-confirm", c.bearerHeader)

		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Empty(t, api.cardRow(t, card.Data.ID).ImageKey)
	})
}

func TestAdminCards_ACardWithoutAConfirmedImageIsNeverInTheCatalog(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	c := api.seedAdminCatalog(t)
	_, card := upsertCard(t, api, c, c.cardBody("001", "Pikachu", "Pokémon"))
	listed := func() bool {
		resp := api.Get("/catalog/cards?expansionSetId="+c.set.ID.String(), bearer(api.newUserToken(t)))
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		var body cardsEnvelope
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		return len(body.Data) > 0
	}

	assert.False(t, listed(), "saved but not yet uploaded")

	api.store.EXPECT().ReadHead(mock.Anything, mock.Anything, mock.Anything).Return([]byte{0xff, 0xd8, 0xff, 0xe0, 0, 0, 0, 0, 0, 0, 0, 0}, nil).Once()
	require.Equal(t, http.StatusOK, api.Post("/admin/cards/"+card.Data.ID.String()+"/image-confirm", c.bearerHeader).Code)

	assert.True(t, listed(), "visible once the image is confirmed")
}

func TestAdminCards_RoutesAnswerNotFoundToANonAdmin(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	c := api.seedAdminCatalog(t)
	nonAdmin := bearer(api.newUserToken(t))

	assert.Equal(t, http.StatusNotFound, api.Put("/admin/cards", nonAdmin, c.cardBody("001", "X", "Pokémon")).Code)
	assert.Equal(t, http.StatusNotFound, api.Post("/admin/cards/"+uuid.NewString()+"/image-upload", nonAdmin, map[string]any{"contentType": "image/png"}).Code)
	assert.Equal(t, http.StatusNotFound, api.Post("/admin/cards/"+uuid.NewString()+"/image-confirm", nonAdmin).Code)
}

// cardPath is the delete address of the Card at localID in the catalog's set.
func (c adminCatalog) cardPath(localID string) string {
	return "/admin/cards?expansionSetId=" + c.set.ID.String() + "&localId=" + localID
}

func TestAdminCards_DeleteRemovesRowObjectEmbeddingsAndBatchMembership(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	c := api.seedAdminCatalog(t)
	_, card := upsertCard(t, api, c, c.cardBody("001", "Mistake", "Pokémon"))
	_, kept := upsertCard(t, api, c, c.cardBody("002", "Keeper", "Pokémon"))
	api.storeEmbedding(t, card.Data.ID, api.model, axisVector(0))
	api.storeEmbedding(t, kept.Data.ID, api.model, axisVector(0))
	batch := entity.EmbeddingBatch{JobName: "job-" + uuid.NewString(), Model: api.model, Source: "catalog", State: entity.EmbeddingBatchSubmitted}
	require.NoError(t, api.db.Create(&batch).Error)
	require.NoError(t, api.db.Create(&[]entity.EmbeddingBatchCard{{BatchID: batch.ID, CardID: card.Data.ID}, {BatchID: batch.ID, CardID: kept.Data.ID}}).Error)
	api.store.EXPECT().Delete(mock.Anything, "cards/"+card.Data.ID.String()).Return(nil).Once()

	resp := api.Delete(c.cardPath("001"), c.bearerHeader)

	require.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
	count := func(model any, where string, args ...any) int64 {
		var n int64
		require.NoError(t, api.db.Model(model).Where(where, args...).Count(&n).Error)
		return n
	}
	assert.Zero(t, count(&entity.Card{}, "id = ?", card.Data.ID), "the row is gone")
	assert.Zero(t, count(&entity.CardEmbedding{}, "card_id = ?", card.Data.ID), "its embeddings are gone")
	// An outstanding batch loses the card, so collecting it later finds a vector
	// for a card that is not in the batch and leaves it out, as it already does.
	assert.Zero(t, count(&entity.EmbeddingBatchCard{}, "card_id = ?", card.Data.ID))
	assert.EqualValues(t, 1, count(&entity.EmbeddingBatchCard{}, "card_id = ?", kept.Data.ID), "other cards are untouched")
	assert.EqualValues(t, 1, count(&entity.CardEmbedding{}, "card_id = ?", kept.Data.ID))
	assert.EqualValues(t, 1, count(&entity.Card{}, "id = ?", kept.Data.ID))
}

func TestAdminCards_DeleteIsRefusedWhileACollectionHoldsTheCard(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	c := api.seedAdminCatalog(t)
	_, card := upsertCard(t, api, c, c.cardBody("001", "Owned", "Pokémon"))
	owner := bearer(api.newUserToken(t))
	created := api.Post("/collections", owner, map[string]any{"title": "Binder"})
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	var col collectionEnvelope
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &col))
	entries := "/collections/" + col.Data.ID + "/entries"
	require.Equal(t, http.StatusCreated, api.Post(entries, owner, map[string]any{"cardId": card.Data.ID, "quantity": 2}).Code)

	resp := api.Delete(c.cardPath("001"), c.bearerHeader)

	assert.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
	assert.Equal(t, card.Data.ID, api.cardRow(t, card.Data.ID).ID, "the card survives")
	var entry entity.InventoryEntry
	require.NoError(t, api.db.First(&entry, "card_id = ?", card.Data.ID).Error)
	assert.Equal(t, 2, entry.Quantity, "the collector's entry is untouched")

	// Once the collector removes it, the Admin can delete.
	require.Equal(t, http.StatusNoContent, api.Delete(entries+"/"+card.Data.ID.String(), owner).Code)
	api.store.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil).Once()
	assert.Equal(t, http.StatusNoContent, api.Delete(c.cardPath("001"), c.bearerHeader).Code)
}

func TestAdminCards_DeleteOfAnUnknownCardIsNotFound(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	c := api.seedAdminCatalog(t)

	assert.Equal(t, http.StatusNotFound, api.Delete(c.cardPath("404"), c.bearerHeader).Code)
}

func TestAdminCards_DeleteAnswersNotFoundToANonAdmin(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	c := api.seedAdminCatalog(t)
	_, card := upsertCard(t, api, c, c.cardBody("001", "Safe", "Pokémon"))

	assert.Equal(t, http.StatusNotFound, api.Delete(c.cardPath("001"), bearer(api.newUserToken(t))).Code, "signed-in non-Admin")
	assert.Equal(t, http.StatusNotFound, api.Delete(c.cardPath("001")).Code, "guest")
	assert.Equal(t, card.Data.ID, api.cardRow(t, card.Data.ID).ID)
}
