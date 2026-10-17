package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

type searchCardsEnvelope struct {
	Data []dto.CardSummary `json:"data"`
	Meta struct {
		Total int `json:"total"`
	} `json:"meta"`
}

type facetsEnvelope struct {
	Data dto.CatalogFacets `json:"data"`
}

type categoriesEnvelope struct {
	Data []string `json:"data"`
}

// TestCatalogUnpublishedSets proves an unpublished Expansion Set and its Cards
// are absent from every catalog view a Guest or a signed-in non-Admin reaches,
// and present to an Admin (GLOSSARY.md Published). The hidden set has its own
// Series, so the Series list shows whether that Series is hidden too.
func TestCatalogUnpublishedSets(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	s := uuid.NewString()
	game := entity.Game{Slug: "pub-" + s, Name: "Pub " + s}
	require.NoError(t, api.db.Create(&game).Error)
	locale := entity.Locale{Code: "pub-" + s}
	require.NoError(t, api.db.Create(&locale).Error)
	rarity := entity.Rarity{GameID: game.ID, Code: "R-" + s, Name: "Rarity " + s}
	require.NoError(t, api.db.Create(&rarity).Error)

	visibleSeries := entity.Series{GameID: game.ID, Code: "vis-" + s, Name: "Visible Series " + s}
	require.NoError(t, api.db.Create(&visibleSeries).Error)
	hiddenSeries := entity.Series{GameID: game.ID, Code: "hid-" + s, Name: "Hidden Series " + s}
	require.NoError(t, api.db.Create(&hiddenSeries).Error)

	visibleSet := entity.ExpansionSet{GameID: game.ID, Code: "vis-" + s, Name: "Visible Set " + s, LocaleID: locale.ID, SeriesID: &visibleSeries.ID}
	require.NoError(t, api.db.Create(&visibleSet).Error)
	hiddenSet := entity.ExpansionSet{GameID: game.ID, Code: "hid-" + s, Name: "Hidden Set " + s, LocaleID: locale.ID, SeriesID: &hiddenSeries.ID}
	require.NoError(t, api.db.Create(&hiddenSet).Error)
	// Create would write the column default (true) for a false Published, so the hide is an update.
	require.NoError(t, api.db.Model(&hiddenSet).Update("published", false).Error)

	newCard := func(set entity.ExpansionSet, name, category string, tags ...string) entity.Card {
		card := entity.Card{ExpansionSetID: set.ID, LocalID: "001", Name: name, Category: category,
			Tags: append(datatypes.JSONSlice[string]{}, tags...), RarityID: rarity.ID, Attributes: datatypes.JSONMap{}}
		require.NoError(t, api.db.Create(&card).Error)
		return card
	}
	visibleCard := newCard(visibleSet, "Shown-"+s, "Pokémon")
	hiddenCard := newCard(hiddenSet, "Secret-"+s, "Cat-"+s, "Tag-"+s)

	get := func(token, path string) *httptest.ResponseRecorder {
		var headers []any
		if token != "" {
			headers = append(headers, bearer(token))
		}
		return api.Get(path, headers...)
	}

	for _, tc := range []struct {
		name  string
		token string
		// seen is whether the caller sees the unpublished set and its Cards.
		seen bool
	}{
		{name: "guest", token: "", seen: false},
		{name: "signed-in non-admin", token: api.newUserToken(t), seen: false},
		{name: "admin", token: api.tokenFor(t, adminEmail), seen: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := 0
			if tc.seen {
				want = 1
			}

			resp := get(tc.token, "/catalog/series")
			require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
			var series seriesEnvelope
			require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &series))
			seriesIDs := []uuid.UUID{}
			setIDs := []uuid.UUID{}
			for _, sr := range series.Data.Series {
				seriesIDs = append(seriesIDs, sr.ID)
				for _, set := range sr.ExpansionSets {
					setIDs = append(setIDs, set.ID)
				}
			}
			for _, set := range series.Data.UngroupedExpansionSets {
				setIDs = append(setIDs, set.ID)
			}
			assert.Contains(t, setIDs, visibleSet.ID)
			assert.Equal(t, tc.seen, slices.Contains(seriesIDs, hiddenSeries.ID), "the Series of only hidden sets")
			assert.Equal(t, tc.seen, slices.Contains(setIDs, hiddenSet.ID), "the hidden set in the series list")

			searchPaths := map[string]uuid.UUID{
				"/catalog/cards?expansionSetId=" + hiddenSet.ID.String(): hiddenCard.ID,
				"/catalog/cards?cardId=" + hiddenCard.ID.String():        hiddenCard.ID,
				"/catalog/cards?name=Secret-" + s:                        hiddenCard.ID,
			}
			for path, cardID := range searchPaths {
				resp := get(tc.token, path)
				require.Equal(t, http.StatusOK, resp.Code, path+": "+resp.Body.String())
				var body searchCardsEnvelope
				require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
				ids := []uuid.UUID{}
				for _, c := range body.Data {
					ids = append(ids, c.ID)
				}
				assert.Equal(t, tc.seen, slices.Contains(ids, cardID), path)
				assert.Equal(t, want, body.Meta.Total, path+" total")
			}

			resp = get(tc.token, "/catalog/cards?name=Shown-"+s)
			require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
			var shown searchCardsEnvelope
			require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &shown))
			assert.Equal(t, 1, shown.Meta.Total, "the published set's Card stays visible")
			assert.Equal(t, visibleCard.ID, shown.Data[0].ID)

			resp = get(tc.token, "/catalog/categories")
			require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
			var categories categoriesEnvelope
			require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &categories))
			assert.Equal(t, tc.seen, slices.Contains(categories.Data, "Cat-"+s), "a category used only by hidden Cards")

			resp = get(tc.token, "/catalog/tags")
			require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
			var tags categoriesEnvelope
			require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &tags))
			assert.Equal(t, tc.seen, slices.Contains(tags.Data, "Tag-"+s), "a tag used only by hidden Cards")

			resp = get(tc.token, "/catalog/facets?expansionSetId="+hiddenSet.ID.String())
			if tc.token == "" {
				assert.Equal(t, http.StatusUnauthorized, resp.Code, "a Guest gets no facets")
				return
			}
			require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
			var facets facetsEnvelope
			require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &facets))
			setFacetIDs := []uuid.UUID{}
			for _, opt := range facets.Data.ExpansionSets {
				setFacetIDs = append(setFacetIDs, opt.ID)
			}
			assert.Equal(t, tc.seen, slices.Contains(setFacetIDs, hiddenSet.ID), "a selected hidden set in the facet options")
		})
	}
}
