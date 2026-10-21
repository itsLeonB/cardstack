package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

const scanFeedbackPath = "/scan/feedback"

// jpegHead is the start of a JPEG: the SOI marker and the marker after it.
var jpegHead = []byte{0xFF, 0xD8, 0xFF, 0xE0}

// feedbackCaller signs in a new user and returns its token and profile ID.
// Photo keys are under the profile ID, not the Clerk user ID (docs/adr/0022).
func feedbackCaller(t *testing.T, api testAPI) (string, string) {
	t.Helper()
	claims := validClaims()
	return api.sign(t, claims), api.callerProfileID(t, claims)
}

// feedbackCollection creates a Collection for the caller and returns its ID.
func feedbackCollection(t *testing.T, api testAPI, token string) string {
	t.Helper()
	resp := api.Post("/collections", bearer(token), map[string]any{"title": "Scans"})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	var c collectionEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &c))
	return c.Data.ID
}

// seedFeedbackCard creates one Card and returns its ID. Every name carries a
// uuid, because the database is shared with other tests.
func seedFeedbackCard(t *testing.T, api testAPI) uuid.UUID {
	t.Helper()
	s := uuid.NewString()
	game := entity.Game{Slug: "feedback-" + s, Name: "Feedback " + s}
	require.NoError(t, api.db.Create(&game).Error)
	locale := entity.Locale{Code: "feedback-" + s}
	require.NoError(t, api.db.Create(&locale).Error)
	rarity := entity.Rarity{GameID: game.ID, Code: "R-" + s, Name: "Rarity " + s}
	require.NoError(t, api.db.Create(&rarity).Error)
	set := entity.ExpansionSet{CreatedBy: entity.SourcePokemonAsia, UpdatedBy: entity.SourcePokemonAsia, GameID: game.ID, Code: "feedback-" + s, Name: "Feedback " + s, LocaleID: locale.ID}
	require.NoError(t, api.db.Create(&set).Error)
	card := entity.Card{CreatedBy: entity.SourcePokemonAsia, UpdatedBy: entity.SourcePokemonAsia, ExpansionSetID: set.ID, LocalID: "001", Name: "Feedback Card " + s, Category: "Pokémon",
		Tags: datatypes.JSONSlice[string]{}, RarityID: rarity.ID, Attributes: datatypes.JSONMap{}}
	require.NoError(t, api.db.Create(&card).Error)
	return card.ID
}

// feedbackDetection is a Detection as /scan/match returns it: opaque to the
// server, so the test only checks that it reads back unchanged.
func feedbackDetection() map[string]any {
	return map[string]any{"confident": true, "cards": []any{map[string]any{"cardId": "2d0f2f5e-7a7b-4c4e-9d2a-000000000001", "similarity": 0.93}}}
}

func feedbackCorners() []any {
	return []any{[]int{10, 10}, []int{200, 12}, []int{198, 300}, []int{12, 298}}
}

// cardScanRow is a row from the single card scan, with no card or key yet.
func cardScanRow(collection, kind string) map[string]any {
	return map[string]any{"source": "card_scan", "collectionId": collection, "kind": kind, "detection": feedbackDetection()}
}

// binderRow is a row from a binder Page Review cell, with no card or key yet.
func binderRow(collection, kind string) map[string]any {
	row := cardScanRow(collection, kind)
	row["source"] = "binder_page"
	row["page"] = 3
	row["slotRow"] = 2
	row["slotColumn"] = 4
	row["gridRows"] = 3
	row["gridColumns"] = 3
	row["corners"] = feedbackCorners()
	return row
}

func withCard(row map[string]any, card uuid.UUID, rank int, rankSource string) map[string]any {
	row["cardId"] = card.String()
	row["rank"] = rank
	row["rankSource"] = rankSource
	return row
}

func withKey(row map[string]any, key string) map[string]any {
	row["imageKey"] = key
	return row
}

func feedbackBatch(rows ...map[string]any) map[string]any {
	return map[string]any{"rows": rows}
}

// expectJPEG makes the bucket answer a valid photo under key.
func expectJPEG(api testAPI, key string) {
	api.photos.EXPECT().Size(mock.Anything, key).Return(int64(1024), nil).Once()
	api.photos.EXPECT().ReadHead(mock.Anything, key, mock.Anything).Return(jpegHead, nil).Once()
}

// rowsOf reads back every stored row of a Collection, which the operator's SQL
// reads too.
func rowsOf(t *testing.T, api testAPI, collection string) []entity.ScanFeedback {
	t.Helper()
	var rows []entity.ScanFeedback
	require.NoError(t, api.db.Where("collection_id = ?", collection).Find(&rows).Error)
	return rows
}

func TestScanFeedback_StoresEveryFieldOfAValidBatch(t *testing.T) {
	api := newTestAPI(t)
	token, profileID := feedbackCaller(t, api)
	collection := feedbackCollection(t, api, token)
	card := seedFeedbackCard(t, api)
	key := profileID + "/" + uuid.NewString()
	expectJPEG(api, key)

	confirmed := withCard(cardScanRow(collection, "confirmed"), card, 1, "top")
	switched := withKey(withCard(binderRow(collection, "switched"), card, 4, "top"), key)
	resp := api.Post(scanFeedbackPath, bearer(token), feedbackBatch(confirmed, switched))

	require.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
	rows := rowsOf(t, api, collection)
	require.Len(t, rows, 2)
	byKind := map[string]entity.ScanFeedback{}
	for _, row := range rows {
		byKind[row.Kind] = row
	}

	c := byKind["confirmed"]
	assert.Equal(t, profileID, c.ProfileID.String())
	assert.Equal(t, "card_scan", c.Source)
	assert.Equal(t, card, *c.CardID)
	assert.Equal(t, 1, *c.CardRank)
	assert.Equal(t, "top", *c.CardRankSource)
	assert.Nil(t, c.ImageKey, "a confirmation carries no photo")
	assert.Nil(t, c.Page, "a card scan has no Page")
	assert.Nil(t, c.Corners)
	assert.JSONEq(t, mustJSON(t, feedbackDetection()), string(c.Detection))
	assert.False(t, c.CreatedAt.IsZero(), "the server sets the timestamp")

	s := byKind["switched"]
	assert.Equal(t, "binder_page", s.Source)
	assert.Equal(t, card, *s.CardID)
	assert.Equal(t, 4, *s.CardRank)
	assert.Equal(t, "top", *s.CardRankSource)
	assert.Equal(t, key, *s.ImageKey)
	assert.Equal(t, 3, *s.Page)
	assert.Equal(t, 2, *s.SlotRow)
	assert.Equal(t, 4, *s.SlotColumn)
	assert.Equal(t, 3, *s.GridRows)
	assert.Equal(t, 3, *s.GridColumns)
	assert.JSONEq(t, mustJSON(t, feedbackCorners()), string(s.Corners))
}

func TestScanFeedback_StoresEmptyAndUnidentifiedWithoutACard(t *testing.T) {
	api := newTestAPI(t)
	token, profileID := feedbackCaller(t, api)
	collection := feedbackCollection(t, api, token)
	key := profileID + "/" + uuid.NewString()
	expectJPEG(api, key)

	resp := api.Post(scanFeedbackPath, bearer(token), feedbackBatch(
		withKey(cardScanRow(collection, "empty"), key),
		cardScanRow(collection, "unidentified"),
	))

	require.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
	rows := rowsOf(t, api, collection)
	require.Len(t, rows, 2)
	for _, row := range rows {
		assert.Nil(t, row.CardID, "%s has no Card", row.Kind)
		assert.Nil(t, row.CardRank)
		assert.Nil(t, row.CardRankSource)
	}
}

func TestScanFeedback_RefusesAConfirmationWithAKey(t *testing.T) {
	api := newTestAPI(t)
	token, profileID := feedbackCaller(t, api)
	collection := feedbackCollection(t, api, token)
	card := seedFeedbackCard(t, api)

	row := withKey(withCard(cardScanRow(collection, "confirmed"), card, 1, "top"), profileID+"/"+uuid.NewString())
	resp := api.Post(scanFeedbackPath, bearer(token), feedbackBatch(row))

	assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
	assert.Empty(t, rowsOf(t, api, collection))
}

func TestScanFeedback_RefusesAKindThatDisagreesWithItsFields(t *testing.T) {
	cases := map[string]func(collection string, card uuid.UUID) map[string]any{
		"empty with a Card": func(c string, card uuid.UUID) map[string]any {
			return withCard(cardScanRow(c, "empty"), card, 1, "top")
		},
		"unidentified with a Card": func(c string, card uuid.UUID) map[string]any {
			return withCard(cardScanRow(c, "unidentified"), card, 1, "top")
		},
		"switched without a Card": func(c string, _ uuid.UUID) map[string]any {
			return cardScanRow(c, "switched")
		},
		"reprint without a Card": func(c string, _ uuid.UUID) map[string]any {
			return cardScanRow(c, "reprint")
		},
		"placed without a Card": func(c string, _ uuid.UUID) map[string]any {
			return cardScanRow(c, "placed")
		},
		"a Card without its rank": func(c string, card uuid.UUID) map[string]any {
			row := cardScanRow(c, "switched")
			row["cardId"] = card.String()
			return row
		},
		"a rank below one": func(c string, card uuid.UUID) map[string]any {
			return withCard(cardScanRow(c, "switched"), card, 0, "top")
		},
		"a card scan with a Page": func(c string, card uuid.UUID) map[string]any {
			row := withCard(cardScanRow(c, "switched"), card, 1, "top")
			row["page"] = 3
			return row
		},
		"a binder cell without its Page": func(c string, card uuid.UUID) map[string]any {
			row := withCard(binderRow(c, "switched"), card, 1, "top")
			delete(row, "page")
			return row
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			api := newTestAPI(t)
			token, _ := feedbackCaller(t, api)
			collection := feedbackCollection(t, api, token)
			card := seedFeedbackCard(t, api)

			resp := api.Post(scanFeedbackPath, bearer(token), feedbackBatch(build(collection, card)))

			assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
			assert.Empty(t, rowsOf(t, api, collection))
		})
	}
}

func TestScanFeedback_RefusesAKeyThatFailsAndDeletesTheObject(t *testing.T) {
	t.Run("a non-JPEG is deleted and the batch refused", func(t *testing.T) {
		api := newTestAPI(t)
		token, profileID := feedbackCaller(t, api)
		collection := feedbackCollection(t, api, token)
		key := profileID + "/" + uuid.NewString()
		api.photos.EXPECT().Size(mock.Anything, key).Return(int64(1024), nil).Once()
		api.photos.EXPECT().ReadHead(mock.Anything, key, mock.Anything).Return([]byte("\x89PNG"), nil).Once()
		api.photos.EXPECT().Delete(mock.Anything, key).Return(nil).Once()

		resp := api.Post(scanFeedbackPath, bearer(token), feedbackBatch(withKey(cardScanRow(collection, "unidentified"), key)))

		assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
		assert.Empty(t, rowsOf(t, api, collection))
	})

	t.Run("an oversized object is deleted before its head is read", func(t *testing.T) {
		api := newTestAPI(t)
		token, profileID := feedbackCaller(t, api)
		collection := feedbackCollection(t, api, token)
		key := profileID + "/" + uuid.NewString()
		api.photos.EXPECT().Size(mock.Anything, key).Return(int64(service.MaxScanPhotoBytes+1), nil).Once()
		api.photos.EXPECT().Delete(mock.Anything, key).Return(nil).Once()

		resp := api.Post(scanFeedbackPath, bearer(token), feedbackBatch(withKey(cardScanRow(collection, "unidentified"), key)))

		assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
		assert.Empty(t, rowsOf(t, api, collection))
	})

	t.Run("a missing object is refused without a delete", func(t *testing.T) {
		api := newTestAPI(t)
		token, profileID := feedbackCaller(t, api)
		collection := feedbackCollection(t, api, token)
		key := profileID + "/" + uuid.NewString()
		api.photos.EXPECT().Size(mock.Anything, key).Return(int64(0), objectstore.ErrNotFound).Once()

		resp := api.Post(scanFeedbackPath, bearer(token), feedbackBatch(withKey(cardScanRow(collection, "unidentified"), key)))

		assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
		assert.Empty(t, rowsOf(t, api, collection))
	})

	t.Run("a key outside the caller's prefix is refused and its object left alone", func(t *testing.T) {
		api := newTestAPI(t)
		token, _ := feedbackCaller(t, api)
		collection := feedbackCollection(t, api, token)
		// Another profile's photo: no call to the bucket may touch it, so the
		// mock has no expectations and fails on any call.
		otherKey := uuid.NewString() + "/" + uuid.NewString()

		resp := api.Post(scanFeedbackPath, bearer(token), feedbackBatch(withKey(cardScanRow(collection, "unidentified"), otherKey)))

		assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
		assert.Empty(t, rowsOf(t, api, collection))
	})
}

func TestScanFeedback_RowsSharingAKeyCheckTheObjectOnce(t *testing.T) {
	api := newTestAPI(t)
	token, profileID := feedbackCaller(t, api)
	collection := feedbackCollection(t, api, token)
	card := seedFeedbackCard(t, api)
	key := profileID + "/" + uuid.NewString()
	expectJPEG(api, key)

	shared := func(kind string) map[string]any {
		return withKey(withCard(binderRow(collection, kind), card, 2, "reprint"), key)
	}
	resp := api.Post(scanFeedbackPath, bearer(token), feedbackBatch(shared("reprint"), shared("switched"), shared("placed")))

	require.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
	rows := rowsOf(t, api, collection)
	require.Len(t, rows, 3)
	for _, row := range rows {
		assert.Equal(t, key, *row.ImageKey)
	}
}

func TestScanFeedback_RefusesMoreThanOneHundredRowsAndOversizedDetections(t *testing.T) {
	api := newTestAPI(t)
	token, _ := feedbackCaller(t, api)
	collection := feedbackCollection(t, api, token)

	rows := make([]map[string]any, 101)
	for i := range rows {
		rows[i] = cardScanRow(collection, "unidentified")
	}
	assert.Equal(t, http.StatusUnprocessableEntity, api.Post(scanFeedbackPath, bearer(token), feedbackBatch(rows...)).Code, "101 rows")

	huge := cardScanRow(collection, "unidentified")
	huge["detection"] = map[string]any{"pad": strings.Repeat("x", service.MaxScanJSONBytes)}
	resp := api.Post(scanFeedbackPath, bearer(token), feedbackBatch(huge))
	assert.Equal(t, http.StatusBadRequest, resp.Code, "a Detection over the size cap")
	assert.Empty(t, rowsOf(t, api, collection))
}

func TestScanFeedback_RefusedBatchStoresNoRows(t *testing.T) {
	api := newTestAPI(t)
	token, profileID := feedbackCaller(t, api)
	collection := feedbackCollection(t, api, token)
	card := seedFeedbackCard(t, api)
	goodKey := profileID + "/" + uuid.NewString()
	expectJPEG(api, goodKey)
	badKey := profileID + "/" + uuid.NewString()
	api.photos.EXPECT().Size(mock.Anything, badKey).Return(int64(1024), nil).Once()
	api.photos.EXPECT().ReadHead(mock.Anything, badKey, mock.Anything).Return([]byte("GIF"), nil).Once()
	api.photos.EXPECT().Delete(mock.Anything, badKey).Return(nil).Once()

	resp := api.Post(scanFeedbackPath, bearer(token), feedbackBatch(
		withKey(withCard(binderRow(collection, "switched"), card, 1, "top"), goodKey),
		withKey(cardScanRow(collection, "unidentified"), badKey),
	))

	assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
	assert.Empty(t, rowsOf(t, api, collection), "the valid row is not stored either")
}

func TestScanFeedback_RejectsGuestsAndBadTokens(t *testing.T) {
	api := newTestAPI(t)
	body := feedbackBatch(cardScanRow(uuid.NewString(), "unidentified"))

	assert.Equal(t, http.StatusUnauthorized, api.Post(scanFeedbackPath, body).Code, "no token")
	assert.Equal(t, http.StatusUnauthorized, api.Post(scanFeedbackPath, bearer("not-a-token"), body).Code, "invalid token")
}

func TestScanFeedback_OnlyTheCallersOwnCollection(t *testing.T) {
	api := newTestAPI(t)
	ownerToken, _ := feedbackCaller(t, api)
	otherCollection := feedbackCollection(t, api, ownerToken)
	intruderToken, _ := feedbackCaller(t, api)

	resp := api.Post(scanFeedbackPath, bearer(intruderToken), feedbackBatch(cardScanRow(otherCollection, "unidentified")))

	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	assert.Empty(t, rowsOf(t, api, otherCollection))
}

func TestScanFeedback_DeletingACollectionKeepsItsRows(t *testing.T) {
	api := newTestAPI(t)
	token, _ := feedbackCaller(t, api)
	collection := feedbackCollection(t, api, token)

	resp := api.Post(scanFeedbackPath, bearer(token), feedbackBatch(cardScanRow(collection, "unidentified")))
	require.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
	require.Len(t, rowsOf(t, api, collection), 1)

	require.Equal(t, http.StatusNoContent, api.Delete("/collections/"+collection, bearer(token)).Code)
	assert.Len(t, rowsOf(t, api, collection), 1, "no foreign key: the rows outlive their Collection")
}

func TestScanFeedback_RefusedClearlyWhenThePhotoBucketIsNotConfigured(t *testing.T) {
	api := newTestAPIWithoutPhotoBucket(t)
	token, _ := feedbackCaller(t, api)
	collection := feedbackCollection(t, api, token)

	resp := api.Post(scanFeedbackPath, bearer(token), feedbackBatch(cardScanRow(collection, "unidentified")))

	assert.Equal(t, http.StatusServiceUnavailable, resp.Code, resp.Body.String())
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "photo_storage_not_configured", body.Code)
	assert.Empty(t, rowsOf(t, api, collection))
}

func TestScanFeedbackMigration_IndexesServeTheOperatorQueries(t *testing.T) {
	api := newTestAPI(t)

	var names []string
	require.NoError(t, api.db.Raw(`SELECT indexname FROM pg_indexes WHERE tablename = 'scan_feedback'`).Scan(&names).Error)
	assert.ElementsMatch(t, []string{
		"scan_feedback_pkey",
		"idx_scan_feedback_card_kind",
		"idx_scan_feedback_kind_created_at",
		"idx_scan_feedback_profile_id",
	}, names)
}

func TestScanFeedback_RefusesARowMissingItsKindOrSource(t *testing.T) {
	api := newTestAPI(t)
	token, _ := feedbackCaller(t, api)
	collection := feedbackCollection(t, api, token)

	noKind := cardScanRow(collection, "unidentified")
	delete(noKind, "kind")
	noSource := cardScanRow(collection, "unidentified")
	delete(noSource, "source")

	assert.Equal(t, http.StatusUnprocessableEntity, api.Post(scanFeedbackPath, bearer(token), feedbackBatch(noKind)).Code, "no kind")
	assert.Equal(t, http.StatusUnprocessableEntity, api.Post(scanFeedbackPath, bearer(token), feedbackBatch(noSource)).Code, "no source")
	assert.Empty(t, rowsOf(t, api, collection))
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}
