package routes

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type cardsEnvelope struct {
	Data []dto.CardSummary `json:"data"`
}

type seriesEnvelope struct {
	Data dto.SeriesBrowseResult `json:"data"`
}

// TestCatalogImageURLs proves the served image address is the configured base
// plus the hosted key, and is empty without a key even when the row still
// carries its scraped source address (docs/adr/0016: never hot-link).
func TestCatalogImageURLs(t *testing.T) {
	api := newTestAPI(t)

	dsn := "host=" + envOr("DB_HOST", "localhost") + " port=" + envOr("DB_PORT", "5432") + " user=" + envOr("DB_USER", "cardstack") +
		" password=" + envOr("DB_PASSWORD", "cardstack") + " dbname=" + envOr("DB_NAME", "cardstack") + " sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)

	s := uuid.NewString()
	game := entity.Game{Slug: "img-" + s, Name: "Img " + s}
	require.NoError(t, db.Create(&game).Error)
	locale := entity.Locale{Code: "img-" + s}
	require.NoError(t, db.Create(&locale).Error)
	rarity := entity.Rarity{GameID: game.ID, Code: "R-" + s, Name: "Rarity " + s}
	require.NoError(t, db.Create(&rarity).Error)

	hostedSet := entity.ExpansionSet{GameID: game.ID, Code: "hosted-" + s, Name: "Hosted " + s, LocaleID: locale.ID, SourceImageURL: "https://source.test/set.png"}
	require.NoError(t, db.Create(&hostedSet).Error)
	require.NoError(t, db.Model(&hostedSet).Updates(map[string]any{"image_key": "expansion-sets/" + hostedSet.ID.String(), "cover_key": "expansion-sets/" + hostedSet.ID.String() + ".0123abcd.webp"}).Error)
	unhostedSet := entity.ExpansionSet{GameID: game.ID, Code: "unhosted-" + s, Name: "Unhosted " + s, LocaleID: locale.ID, SourceImageURL: "https://source.test/set.png"}
	require.NoError(t, db.Create(&unhostedSet).Error)

	newCard := func(set entity.ExpansionSet, localID, key string) entity.Card {
		card := entity.Card{ExpansionSetID: set.ID, LocalID: localID, Name: "Card " + s, Category: "Pokémon", Tags: datatypes.JSONSlice[string]{},
			RarityID: rarity.ID, Attributes: datatypes.JSONMap{}, SourceImageURL: "https://source.test/" + localID + ".png", ImageKey: key}
		require.NoError(t, db.Create(&card).Error)
		return card
	}
	hostedCard := newCard(hostedSet, "001", "cards/hosted-"+s)
	unhostedCard := newCard(unhostedSet, "002", "")

	t.Run("cards", func(t *testing.T) {
		resp := api.Get("/catalog/cards?expansionSetId=" + hostedSet.ID.String() + "&expansionSetId=" + unhostedSet.ID.String())
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		var body cardsEnvelope
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))

		byID := map[uuid.UUID]dto.CardSummary{}
		for _, c := range body.Data {
			byID[c.ID] = c
		}
		require.Len(t, byID, 2)
		assert.Equal(t, testImageBase+"/cards/hosted-"+s, byID[hostedCard.ID].ImageURL)
		assert.Equal(t, testImageBase+"/expansion-sets/"+hostedSet.ID.String()+".0123abcd.webp", byID[hostedCard.ID].ExpansionSet.ImageURL)
		assert.Empty(t, byID[unhostedCard.ID].ImageURL, "no key means no image, never the source address")
		assert.Empty(t, byID[unhostedCard.ID].ExpansionSet.ImageURL)
	})

	t.Run("series browse", func(t *testing.T) {
		resp := api.Get("/catalog/series")
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		var body seriesEnvelope
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))

		urls := map[uuid.UUID]string{}
		for _, set := range body.Data.UngroupedExpansionSets {
			urls[set.ID] = set.ImageURL
		}
		assert.Equal(t, testImageBase+"/expansion-sets/"+hostedSet.ID.String()+".0123abcd.webp", urls[hostedSet.ID])
		assert.Empty(t, urls[unhostedSet.ID])
		_, listed := urls[unhostedSet.ID]
		assert.True(t, listed, "the unhosted set is still listed")
	})
}

func TestCatalogCardIDFilter(t *testing.T) {
	api := newTestAPI(t)
	cards := newTestCards(t, 3)
	set := "cardId="

	t.Run("returns exactly the requested Cards", func(t *testing.T) {
		resp := api.Get("/catalog/cards?" + set + cards[0].String() + "&" + set + cards[2].String())
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		var body cardsEnvelope
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		got := []uuid.UUID{}
		for _, c := range body.Data {
			got = append(got, c.ID)
		}
		assert.ElementsMatch(t, []uuid.UUID{cards[0], cards[2]}, got)
	})

	t.Run("combines with other filters", func(t *testing.T) {
		resp := api.Get("/catalog/cards?" + set + cards[0].String() + "&localId=b")
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		var body cardsEnvelope
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		assert.Empty(t, body.Data)
	})

	t.Run("rejects more than 100 ids and malformed ids", func(t *testing.T) {
		q := ""
		for range 101 {
			q += "&" + set + uuid.NewString()
		}
		assert.Equal(t, http.StatusUnprocessableEntity, api.Get("/catalog/cards?"+q[1:]).Code)
		assert.Equal(t, http.StatusBadRequest, api.Get("/catalog/cards?"+set+"nope").Code)
	})
}
