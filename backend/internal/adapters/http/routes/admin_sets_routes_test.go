package routes

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type setEnvelope struct {
	Data dto.ExpansionSetSummary `json:"data"`
}

// adminSetFixture is a Game with one Series holding one dated set, the shape
// the Asia ingester leaves behind.
type adminSetFixture struct {
	api    testAPI
	token  string
	game   entity.Game
	series entity.Series
	suffix string
}

func newAdminSetFixture(t *testing.T) adminSetFixture {
	t.Helper()

	api := newTestAPIWithAdmins(t, []string{adminEmail})
	s := uuid.NewString()
	game := entity.Game{Slug: "adm-" + s, Name: "Adm " + s}
	require.NoError(t, api.db.Create(&game).Error)
	series := entity.Series{GameID: game.ID, Code: "ser-" + s, Name: "Series " + s}
	require.NoError(t, api.db.Create(&series).Error)

	return adminSetFixture{api: api, token: api.tokenFor(t, adminEmail), game: game, series: series, suffix: s[:8]}
}

func (f adminSetFixture) put(t *testing.T, code string, body map[string]any) (int, string) {
	t.Helper()
	resp := f.api.Put("/admin/expansion-sets/"+code, bearer(f.token), body)
	return resp.Code, resp.Body.String()
}

func (f adminSetFixture) setByCode(t *testing.T, code string) entity.ExpansionSet {
	t.Helper()
	var set entity.ExpansionSet
	require.NoError(t, f.api.db.Where("game_id = ? AND code = ?", f.game.ID, code).First(&set).Error)
	return set
}

func (f adminSetFixture) seedSet(t *testing.T, code string, releaseDate time.Time) entity.ExpansionSet {
	t.Helper()
	locale := entity.Locale{Code: "seed-" + uuid.NewString()}
	require.NoError(t, f.api.db.Create(&locale).Error)
	set := entity.ExpansionSet{GameID: f.game.ID, Code: code, Name: "Seeded " + code, LocaleID: locale.ID, SeriesID: &f.series.ID, ReleaseDate: &releaseDate}
	require.NoError(t, f.api.db.Create(&set).Error)
	return set
}

func (f adminSetFixture) browse(t *testing.T) dto.SeriesBrowseResult {
	t.Helper()
	resp := f.api.Get("/catalog/series")
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var body seriesEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	return body.Data
}

func TestAdminSets_CreateMakesACuratedSetThatBrowseShowsUnderItsSeries(t *testing.T) {
	f := newAdminSetFixture(t)
	code := "SM-P-" + f.suffix

	status, body := f.put(t, code, map[string]any{"name": "Kartu Promo Matahari & Bulan", "releaseDate": "2019-05-08", "seriesId": f.series.ID})

	require.Equal(t, http.StatusOK, status, body)
	var got setEnvelope
	require.NoError(t, json.Unmarshal([]byte(body), &got))
	assert.Equal(t, code, got.Data.Code)
	assert.Equal(t, "Kartu Promo Matahari & Bulan", got.Data.Name)
	assert.Empty(t, got.Data.ImageURL, "no cover")

	row := f.setByCode(t, code)
	assert.True(t, row.Curated)
	assert.Equal(t, f.series.ID, *row.SeriesID)
	assert.Equal(t, "2019-05-08", row.ReleaseDate.Format(time.DateOnly))
	assert.NotEqual(t, uuid.Nil, row.LocaleID)

	var inSeries []dto.ExpansionSetSummary
	for _, sr := range f.browse(t).Series {
		if sr.ID == f.series.ID {
			inSeries = sr.ExpansionSets
		}
	}
	require.Len(t, inSeries, 1, "the new set is listed under its Series")
	assert.Equal(t, row.ID, inSeries[0].ID)
	assert.Empty(t, inSeries[0].ImageURL, "a cover-less set carries no image address, as other cover-less sets do")
}

func TestAdminSets_EditingAnIngestedSetKeepsItsIdentityAndMarksItCurated(t *testing.T) {
	f := newAdminSetFixture(t)
	code := "ASIA-" + f.suffix
	seeded := f.seedSet(t, code, time.Date(2016, 12, 2, 0, 0, 0, 0, time.UTC))
	require.False(t, seeded.Curated)

	status, body := f.put(t, code, map[string]any{"name": "Corrected name", "releaseDate": "2016-12-03", "seriesId": f.series.ID})

	require.Equal(t, http.StatusOK, status, body)
	row := f.setByCode(t, code)
	assert.Equal(t, seeded.ID, row.ID, "updated in place, not duplicated")
	assert.Equal(t, seeded.LocaleID, row.LocaleID)
	assert.Equal(t, "Corrected name", row.Name)
	assert.Equal(t, "2016-12-03", row.ReleaseDate.Format(time.DateOnly))
	assert.True(t, row.Curated)

	status, body = f.put(t, code, map[string]any{"name": "Corrected name", "releaseDate": "2016-12-03", "seriesId": f.series.ID})
	require.Equal(t, http.StatusOK, status, body)
	var n int64
	require.NoError(t, f.api.db.Model(&entity.ExpansionSet{}).Where("game_id = ? AND code = ?", f.game.ID, code).Count(&n).Error)
	assert.EqualValues(t, 1, n, "entering the same code again is not a second set")
}

func TestAdminSets_UnknownSeriesIsAValidationErrorAndWritesNothing(t *testing.T) {
	f := newAdminSetFixture(t)
	code := "NOPE-" + f.suffix

	status, body := f.put(t, code, map[string]any{"name": "Orphan", "releaseDate": "2019-05-08", "seriesId": uuid.New()})

	assert.Equal(t, http.StatusUnprocessableEntity, status, body)
	var n int64
	require.NoError(t, f.api.db.Model(&entity.ExpansionSet{}).Where("code = ?", code).Count(&n).Error)
	assert.Zero(t, n)
}

func TestAdminSets_AddingALaterSetDoesNotReorderSeries(t *testing.T) {
	f := newAdminSetFixture(t)
	f.seedSet(t, "OLD-"+f.suffix, time.Date(2016, 12, 2, 0, 0, 0, 0, time.UTC))
	newer := entity.Series{GameID: f.game.ID, Code: "new-" + f.suffix, Name: "Newer " + f.suffix}
	require.NoError(t, f.api.db.Create(&newer).Error)
	newerDate := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	locale := entity.Locale{Code: "seed-" + uuid.NewString()}
	require.NoError(t, f.api.db.Create(&locale).Error)
	require.NoError(t, f.api.db.Create(&entity.ExpansionSet{GameID: f.game.ID, Code: "NEW-" + f.suffix, Name: "n", LocaleID: locale.ID, SeriesID: &newer.ID, ReleaseDate: &newerDate}).Error)

	order := func() (older, newerAt int) {
		ids := []uuid.UUID{}
		for _, sr := range f.browse(t).Series {
			ids = append(ids, sr.ID)
		}
		return slices.Index(ids, f.series.ID), slices.Index(ids, newer.ID)
	}
	olderBefore, newerBefore := order()
	require.Less(t, newerBefore, olderBefore, "most recent Series first")

	status, body := f.put(t, "SM-P-"+f.suffix, map[string]any{"name": "Promo", "releaseDate": "2019-05-08", "seriesId": f.series.ID})
	require.Equal(t, http.StatusOK, status, body)

	olderAfter, newerAfter := order()
	assert.Less(t, newerAfter, olderAfter, "a set later than the Series' earliest leaves its place unchanged")
}

func TestAdminSets_OnlyAnAdminMayWrite(t *testing.T) {
	f := newAdminSetFixture(t)
	code := "SM-P-" + f.suffix
	payload := map[string]any{"name": "Promo", "releaseDate": "2019-05-08", "seriesId": f.series.ID}

	signedIn := f.api.Put("/admin/expansion-sets/"+code, bearer(f.api.newUserToken(t)), payload)
	guest := f.api.Put("/admin/expansion-sets/"+code, payload)

	assert.Equal(t, http.StatusNotFound, signedIn.Code)
	assert.Equal(t, http.StatusNotFound, guest.Code)
	var n int64
	require.NoError(t, f.api.db.Model(&entity.ExpansionSet{}).Where("code = ?", code).Count(&n).Error)
	assert.Zero(t, n)
}
