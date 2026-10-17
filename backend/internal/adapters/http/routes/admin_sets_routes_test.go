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

// create POSTs a new set with code; body carries the rest of its fields.
func (f adminSetFixture) create(t *testing.T, code string, body map[string]any) (int, string) {
	t.Helper()
	resp := f.api.Post("/admin/expansion-sets", bearer(f.token), withCode(code, body))
	return resp.Code, resp.Body.String()
}

func withCode(code string, body map[string]any) map[string]any {
	out := map[string]any{"code": code}
	for k, v := range body {
		out[k] = v
	}
	return out
}

func (f adminSetFixture) update(t *testing.T, id uuid.UUID, body map[string]any) (int, string) {
	t.Helper()
	resp := f.api.Put("/admin/expansion-sets/"+id.String(), bearer(f.token), body)
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
	set := entity.ExpansionSet{CreatedBy: entity.SourcePokemonAsia, UpdatedBy: entity.SourcePokemonAsia, GameID: f.game.ID, Code: code, Name: "Seeded " + code, LocaleID: locale.ID, SeriesID: &f.series.ID, ReleaseDate: &releaseDate}
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

	status, body := f.create(t, code, map[string]any{"name": "Kartu Promo Matahari & Bulan", "releaseDate": "2019-05-08", "seriesId": f.series.ID})

	require.Equal(t, http.StatusCreated, status, body)
	var got setEnvelope
	require.NoError(t, json.Unmarshal([]byte(body), &got))
	assert.Equal(t, code, got.Data.Code)
	assert.Equal(t, "Kartu Promo Matahari & Bulan", got.Data.Name)
	assert.Empty(t, got.Data.ImageURL, "no cover")

	row := f.setByCode(t, code)
	assert.Equal(t, entity.SourceAdmin, row.CreatedBy)
	assert.Equal(t, entity.SourceAdmin, row.UpdatedBy)
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
	require.Equal(t, entity.SourcePokemonAsia, seeded.UpdatedBy)

	status, body := f.update(t, seeded.ID, map[string]any{"name": "Corrected name", "releaseDate": "2016-12-03", "seriesId": f.series.ID})

	require.Equal(t, http.StatusOK, status, body)
	row := f.setByCode(t, code)
	assert.Equal(t, seeded.ID, row.ID, "updated in place, not duplicated")
	assert.Equal(t, code, row.Code, "the code is not editable")
	assert.Equal(t, seeded.LocaleID, row.LocaleID)
	assert.Equal(t, "Corrected name", row.Name)
	assert.Equal(t, "2016-12-03", row.ReleaseDate.Format(time.DateOnly))
	assert.Equal(t, entity.SourceAdmin, row.UpdatedBy)

	status, body = f.update(t, seeded.ID, map[string]any{"name": "Corrected name", "releaseDate": "2016-12-03", "seriesId": f.series.ID})
	require.Equal(t, http.StatusOK, status, body)
	var n int64
	require.NoError(t, f.api.db.Model(&entity.ExpansionSet{}).Where("game_id = ? AND code = ?", f.game.ID, code).Count(&n).Error)
	assert.EqualValues(t, 1, n, "putting the same set again is not a second set")
}

func TestAdminSets_UpdateMovesASetToAnotherSeriesOfItsGameButNotOfAnotherGame(t *testing.T) {
	f := newAdminSetFixture(t)
	seeded := f.seedSet(t, "MV-"+f.suffix, time.Date(2016, 12, 2, 0, 0, 0, 0, time.UTC))
	sameGame := entity.Series{GameID: f.game.ID, Code: "alt-" + f.suffix, Name: "Alt " + f.suffix}
	require.NoError(t, f.api.db.Create(&sameGame).Error)
	otherGame := entity.Game{Slug: "oth-" + f.suffix, Name: "Oth " + f.suffix}
	require.NoError(t, f.api.db.Create(&otherGame).Error)
	foreign := entity.Series{GameID: otherGame.ID, Code: "for-" + f.suffix, Name: "For " + f.suffix}
	require.NoError(t, f.api.db.Create(&foreign).Error)

	status, body := f.update(t, seeded.ID, map[string]any{"name": "Moved", "releaseDate": "2016-12-02", "seriesId": sameGame.ID})
	require.Equal(t, http.StatusOK, status, body)
	assert.Equal(t, sameGame.ID, *f.setByCode(t, seeded.Code).SeriesID)

	status, body = f.update(t, seeded.ID, map[string]any{"name": "Moved", "releaseDate": "2016-12-02", "seriesId": foreign.ID})
	assert.Equal(t, http.StatusUnprocessableEntity, status, body)
	assert.Equal(t, sameGame.ID, *f.setByCode(t, seeded.Code).SeriesID, "a refused move changes nothing")
}

func TestAdminSets_UpdateOfAnUnknownSetIsNotFound(t *testing.T) {
	f := newAdminSetFixture(t)

	status, body := f.update(t, uuid.New(), map[string]any{"name": "Ghost", "releaseDate": "2019-05-08", "seriesId": f.series.ID})

	assert.Equal(t, http.StatusNotFound, status, body)
}

func TestAdminSets_CreateOfAnExistingCodeIsAConflictAndChangesNothing(t *testing.T) {
	f := newAdminSetFixture(t)
	code := "ASIA-" + f.suffix
	seeded := f.seedSet(t, code, time.Date(2016, 12, 2, 0, 0, 0, 0, time.UTC))

	status, body := f.create(t, code, map[string]any{"name": "Other", "releaseDate": "2019-05-08", "seriesId": f.series.ID})

	assert.Equal(t, http.StatusConflict, status, body)
	row := f.setByCode(t, code)
	assert.Equal(t, seeded.ID, row.ID)
	assert.Equal(t, seeded.Name, row.Name, "a duplicate is not an update")
	assert.NotEqual(t, entity.SourceAdmin, row.UpdatedBy)
}

func TestAdminSets_UnknownSeriesIsAValidationErrorAndWritesNothing(t *testing.T) {
	f := newAdminSetFixture(t)
	code := "NOPE-" + f.suffix

	status, body := f.create(t, code, map[string]any{"name": "Orphan", "releaseDate": "2019-05-08", "seriesId": uuid.New()})

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
	require.NoError(t, f.api.db.Create(&entity.ExpansionSet{CreatedBy: entity.SourcePokemonAsia, UpdatedBy: entity.SourcePokemonAsia, GameID: f.game.ID, Code: "NEW-" + f.suffix, Name: "n", LocaleID: locale.ID, SeriesID: &newer.ID, ReleaseDate: &newerDate}).Error)

	order := func() (older, newerAt int) {
		ids := []uuid.UUID{}
		for _, sr := range f.browse(t).Series {
			ids = append(ids, sr.ID)
		}
		return slices.Index(ids, f.series.ID), slices.Index(ids, newer.ID)
	}
	olderBefore, newerBefore := order()
	require.Less(t, newerBefore, olderBefore, "most recent Series first")

	status, body := f.create(t, "SM-P-"+f.suffix, map[string]any{"name": "Promo", "releaseDate": "2019-05-08", "seriesId": f.series.ID})
	require.Equal(t, http.StatusCreated, status, body)

	olderAfter, newerAfter := order()
	assert.Less(t, newerAfter, olderAfter, "a set later than the Series' earliest leaves its place unchanged")
}

func TestAdminSets_OnlyAnAdminMayWrite(t *testing.T) {
	f := newAdminSetFixture(t)
	code := "SM-P-" + f.suffix
	payload := map[string]any{"name": "Promo", "releaseDate": "2019-05-08", "seriesId": f.series.ID}
	seeded := f.seedSet(t, "OWN-"+f.suffix, time.Date(2016, 12, 2, 0, 0, 0, 0, time.UTC))
	nonAdmin := bearer(f.api.newUserToken(t))

	for name, resp := range map[string]int{
		"signed-in create": f.api.Post("/admin/expansion-sets", nonAdmin, withCode(code, payload)).Code,
		"guest create":     f.api.Post("/admin/expansion-sets", withCode(code, payload)).Code,
		"signed-in update": f.api.Put("/admin/expansion-sets/"+seeded.ID.String(), nonAdmin, payload).Code,
		"guest update":     f.api.Put("/admin/expansion-sets/"+seeded.ID.String(), payload).Code,
	} {
		assert.Equal(t, http.StatusNotFound, resp, name)
	}
	assert.Equal(t, seeded.Name, f.setByCode(t, seeded.Code).Name)
	var n int64
	require.NoError(t, f.api.db.Model(&entity.ExpansionSet{}).Where("code = ?", code).Count(&n).Error)
	assert.Zero(t, n)
}
