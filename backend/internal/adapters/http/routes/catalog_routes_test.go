package routes

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
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
	services := authTestServices(t)
	_, api := humatest.New(t, httpapi.NewConfig())
	RegisterRoutes(api, services)

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
	require.NoError(t, db.Model(&hostedSet).Update("image_key", "expansion-sets/"+hostedSet.ID.String()).Error)
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
		assert.Equal(t, testImageBase+"/expansion-sets/"+hostedSet.ID.String(), byID[hostedCard.ID].ExpansionSet.ImageURL)
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
		assert.Equal(t, testImageBase+"/expansion-sets/"+hostedSet.ID.String(), urls[hostedSet.ID])
		assert.Empty(t, urls[unhostedSet.ID])
		_, listed := urls[unhostedSet.ID]
		assert.True(t, listed, "the unhosted set is still listed")
	})
}
