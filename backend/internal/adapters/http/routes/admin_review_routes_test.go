package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

const reviewQueuePath = "/admin/reviews"

// reviewItemBody is one item of GET /admin/reviews as the Admin app reads it.
type reviewItemBody struct {
	ID        uuid.UUID `json:"id"`
	Kind      string    `json:"kind"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"createdAt"`
	Card      struct {
		ID       uuid.UUID `json:"id"`
		ImageURL string    `json:"imageUrl"`
	} `json:"card"`
	Scores []struct {
		CardID uuid.UUID `json:"cardId"`
		Score  float64   `json:"score"`
	} `json:"scores"`
	Binder *struct {
		Page        int `json:"page"`
		SlotRow     int `json:"slotRow"`
		SlotColumn  int `json:"slotColumn"`
		GridRows    int `json:"gridRows"`
		GridColumns int `json:"gridColumns"`
		Corners     []struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"corners"`
	} `json:"binder"`
}

type reviewQueueBody struct {
	Data []reviewItemBody `json:"data"`
	Meta struct {
		Total int `json:"total"`
		Page  int `json:"page"`
		Limit int `json:"limit"`
	} `json:"meta"`
}

// queueItems pages through the whole queue. The database is shared with other
// tests, whose pending rows the queue lists too, so tests look for their own
// rows in it rather than counting it.
func queueItems(t *testing.T, api testAPI, token string) []reviewItemBody {
	t.Helper()
	var items []reviewItemBody
	for page := 1; ; page++ {
		resp := api.Get(fmt.Sprintf("%s?page=%d&limit=100", reviewQueuePath, page), bearer(token))
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		var body reviewQueueBody
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		items = append(items, body.Data...)
		if len(body.Data) < 100 {
			return items
		}
	}
}

func queueIDs(t *testing.T, api testAPI, token string) []uuid.UUID {
	t.Helper()
	var ids []uuid.UUID
	for _, item := range queueItems(t, api, token) {
		ids = append(ids, item.ID)
	}
	return ids
}

// adminCaller is a token for the Admin on the Admin app origin, and that
// Admin's profile ID.
func adminCaller(t *testing.T, api testAPI) (string, uuid.UUID) {
	t.Helper()
	claims := validClaims()
	claims["email"] = adminEmail
	claims["azp"] = testAdminOrigin
	id, err := uuid.Parse(api.callerProfileID(t, claims))
	require.NoError(t, err)
	return api.sign(t, claims), id
}

// candidatesDetection is a Detection as /scan/match returns it, offering top
// first and second after it.
func candidatesDetection(t *testing.T, top, second uuid.UUID) datatypes.JSON {
	t.Helper()
	return datatypes.JSON(mustJSON(t, map[string]any{
		"empty":     false,
		"confident": false,
		"candidates": []any{
			map[string]any{"score": 0.91, "card": map[string]any{"id": top}},
			map[string]any{"score": 0.88, "card": map[string]any{"id": second}},
		},
	}))
}

// correctionFor is a switched-or-other Correction of profile at card, with a
// photo under the profile's prefix and the given creation time.
func correctionFor(t *testing.T, profile, card uuid.UUID, kind string, created time.Time) entity.ScanFeedback {
	t.Helper()
	key := profile.String() + "/" + uuid.NewString() + ".jpg"
	row := entity.ScanFeedback{
		ProfileID:      profile,
		CollectionID:   uuid.New(),
		Source:         "card_scan",
		Kind:           kind,
		Detection:      candidatesDetection(t, card, uuid.New()),
		CardID:         &card,
		CardRank:       refInt(1),
		CardRankSource: refString("top"),
		ImageKey:       &key,
	}
	row.CreatedAt = created
	return row
}

// reviewCorners are a Page's four corners as the Admin app sends them.
func reviewCorners() []map[string]float64 {
	return []map[string]float64{{"x": 0.1, "y": 0.1}, {"x": 0.9, "y": 0.1}, {"x": 0.9, "y": 0.9}, {"x": 0.1, "y": 0.9}}
}

func refInt(v int) *int          { return &v }
func refString(v string) *string { return &v }

func insertFeedback(t *testing.T, api testAPI, row entity.ScanFeedback) entity.ScanFeedback {
	t.Helper()
	require.NoError(t, api.db.Create(&row).Error)
	return row
}

// seedReviewCard is a feedback card with a catalog image, so an item can show it.
func seedReviewCard(t *testing.T, api testAPI) uuid.UUID {
	t.Helper()
	card := seedFeedbackCard(t, api)
	require.NoError(t, api.db.Model(&entity.Card{}).Where("id = ?", card).Update("image_key", "cards/"+card.String()+".webp").Error)
	return card
}

func TestAdminReviews_ListsOnlyPendingCorrectionsWithAPhotoAndACard_OldestFirst(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	token, _ := adminCaller(t, api)
	_, profileID := feedbackCaller(t, api)
	profile, err := uuid.Parse(profileID)
	require.NoError(t, err)
	card := seedFeedbackCard(t, api)
	now := time.Now()

	// Eligible: switched, reprint and placed, each with a photo and a Card. The
	// switched one was recorded before the feature shipped, and still counts.
	oldSwitched := insertFeedback(t, api, correctionFor(t, profile, card, "switched", now.Add(-3*time.Hour)))
	binderReprint := correctionFor(t, profile, card, "reprint", now.Add(-2*time.Hour))
	binderReprint.Source = "binder_page"
	binderReprint.Page, binderReprint.SlotRow, binderReprint.SlotColumn = refInt(4), refInt(2), refInt(3)
	binderReprint.GridRows, binderReprint.GridColumns = refInt(3), refInt(3)
	binderReprint.Corners = datatypes.JSON(mustJSON(t, reviewCorners()))
	binderReprint = insertFeedback(t, api, binderReprint)
	placed := insertFeedback(t, api, correctionFor(t, profile, card, "placed", now.Add(-time.Hour)))

	// Not eligible: a confirmation (no photo), a row without a Card, a row of an
	// ineligible kind with a photo, a Correction with no photo, and two already
	// decided.
	confirmed := correctionFor(t, profile, card, "confirmed", now.Add(-4*time.Hour))
	confirmed.ImageKey = nil
	confirmed = insertFeedback(t, api, confirmed)
	emptyWithPhoto := correctionFor(t, profile, card, "empty", now.Add(-4*time.Hour))
	emptyWithPhoto.CardID, emptyWithPhoto.CardRank, emptyWithPhoto.CardRankSource = nil, nil, nil
	emptyWithPhoto = insertFeedback(t, api, emptyWithPhoto)
	unidentifiedWithPhoto := correctionFor(t, profile, card, "unidentified", now.Add(-4*time.Hour))
	unidentifiedWithPhoto.CardID, unidentifiedWithPhoto.CardRank, unidentifiedWithPhoto.CardRankSource = nil, nil, nil
	unidentifiedWithPhoto = insertFeedback(t, api, unidentifiedWithPhoto)
	noPhoto := correctionFor(t, profile, card, "switched", now.Add(-4*time.Hour))
	noPhoto.ImageKey = nil
	noPhoto = insertFeedback(t, api, noPhoto)
	rejected := correctionFor(t, profile, card, "switched", now.Add(-4*time.Hour))
	rejected.ReviewStatus, rejected.ReviewedAt = refString("rejected"), &now
	rejected = insertFeedback(t, api, rejected)

	ids := queueIDs(t, api, token)

	var ours []uuid.UUID
	for _, id := range ids {
		if id == oldSwitched.ID || id == binderReprint.ID || id == placed.ID {
			ours = append(ours, id)
		}
		for _, excluded := range []entity.ScanFeedback{confirmed, emptyWithPhoto, unidentifiedWithPhoto, noPhoto, rejected} {
			assert.NotEqual(t, excluded.ID, id, "a %s row without a pending photo and Card is never listed", excluded.Kind)
		}
	}
	assert.Equal(t, []uuid.UUID{oldSwitched.ID, binderReprint.ID, placed.ID}, ours, "eligible rows, oldest first")
}

func TestAdminReviews_AnItemCarriesWhatTheAdminDecidesOn(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	token, _ := adminCaller(t, api)
	_, profileID := feedbackCaller(t, api)
	profile, err := uuid.Parse(profileID)
	require.NoError(t, err)
	card := seedReviewCard(t, api)
	second := uuid.New()
	now := time.Now()

	scan := correctionFor(t, profile, card, "switched", now.Add(-2*time.Hour))
	scan.Detection = candidatesDetection(t, second, card)
	scan = insertFeedback(t, api, scan)
	binder := correctionFor(t, profile, card, "reprint", now.Add(-time.Hour))
	binder.Source = "binder_page"
	binder.Page, binder.SlotRow, binder.SlotColumn = refInt(4), refInt(2), refInt(3)
	binder.GridRows, binder.GridColumns = refInt(3), refInt(3)
	binder.Corners = datatypes.JSON(mustJSON(t, reviewCorners()))
	binder = insertFeedback(t, api, binder)

	items := map[uuid.UUID]reviewItemBody{}
	for _, item := range queueItems(t, api, token) {
		items[item.ID] = item
	}

	gotScan := items[scan.ID]
	assert.Equal(t, "switched", gotScan.Kind)
	assert.Equal(t, "card_scan", gotScan.Source)
	assert.Equal(t, card, gotScan.Card.ID)
	assert.Equal(t, testImageBase+"/cards/"+card.String()+".webp", gotScan.Card.ImageURL, "the chosen Card's catalog image")
	require.Len(t, gotScan.Scores, 2)
	assert.Equal(t, []uuid.UUID{second, card}, []uuid.UUID{gotScan.Scores[0].CardID, gotScan.Scores[1].CardID}, "candidates in the order the matcher offered them")
	assert.InDelta(t, 0.91, gotScan.Scores[0].Score, 1e-9)
	assert.Nil(t, gotScan.Binder, "a card scan carries no binder cell")

	gotBinder := items[binder.ID]
	assert.Equal(t, "binder_page", gotBinder.Source)
	require.NotNil(t, gotBinder.Binder)
	assert.Equal(t, 4, gotBinder.Binder.Page)
	assert.Equal(t, 2, gotBinder.Binder.SlotRow)
	assert.Equal(t, 3, gotBinder.Binder.SlotColumn)
	assert.Equal(t, 3, gotBinder.Binder.GridRows)
	assert.Equal(t, 3, gotBinder.Binder.GridColumns)
	require.Len(t, gotBinder.Binder.Corners, 4)
	assert.Equal(t, 0.1, gotBinder.Binder.Corners[0].X)
}

func TestAdminReviews_PhotoAddressIsSignedOnlyForAListedCorrection(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	token, _ := adminCaller(t, api)
	_, profileID := feedbackCaller(t, api)
	profile, err := uuid.Parse(profileID)
	require.NoError(t, err)
	card := seedFeedbackCard(t, api)

	pending := insertFeedback(t, api, correctionFor(t, profile, card, "switched", time.Now()))
	decided := correctionFor(t, profile, card, "switched", time.Now())
	decided.ReviewStatus, decided.ReviewedAt = refString("rejected"), refTime(time.Now())
	decided = insertFeedback(t, api, decided)
	confirmed := correctionFor(t, profile, card, "confirmed", time.Now())
	confirmed.ImageKey = nil
	confirmed = insertFeedback(t, api, confirmed)

	const signed = "https://photos.example.test/signed"
	expiresAt := time.Now().Add(5 * time.Minute)
	api.photos.EXPECT().PresignGet(mock.Anything, *pending.ImageKey, mock.MatchedBy(func(ttl time.Duration) bool {
		return ttl > 0 && ttl <= 5*time.Minute
	})).Return(signed, expiresAt, nil).Once()

	resp := api.Get(reviewQueuePath+"/"+pending.ID.String()+"/photo", bearer(token))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var body struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, signed, body.Data.URL)

	// No expectation is set on the bucket for these: a call to it fails the test.
	assert.Equal(t, http.StatusNotFound, api.Get(reviewQueuePath+"/"+decided.ID.String()+"/photo", bearer(token)).Code, "a decided Correction is not in the queue")
	assert.Equal(t, http.StatusNotFound, api.Get(reviewQueuePath+"/"+confirmed.ID.String()+"/photo", bearer(token)).Code, "a confirmation has no photo")
	assert.Equal(t, http.StatusNotFound, api.Get(reviewQueuePath+"/"+uuid.NewString()+"/photo", bearer(token)).Code, "an unknown Correction")
}

func TestAdminReviews_RejectMarksTheRowAndEmbedsNothing(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	token, adminProfile := adminCaller(t, api)
	_, profileID := feedbackCaller(t, api)
	profile, err := uuid.Parse(profileID)
	require.NoError(t, err)
	card := seedFeedbackCard(t, api)
	row := insertFeedback(t, api, correctionFor(t, profile, card, "switched", time.Now()))

	var embeddingsBefore int64
	require.NoError(t, api.db.Model(&entity.CardEmbedding{}).Where("card_id = ?", card).Count(&embeddingsBefore).Error)

	// The bucket has no expectations: a delete or a read would fail the test.
	resp := api.Post(reviewQueuePath+"/"+row.ID.String()+"/reject", bearer(token))
	require.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())

	var stored entity.ScanFeedback
	require.NoError(t, api.db.Where("id = ?", row.ID).Take(&stored).Error)
	require.NotNil(t, stored.ReviewStatus)
	assert.Equal(t, "rejected", *stored.ReviewStatus)
	require.NotNil(t, stored.ReviewedBy)
	assert.Equal(t, adminProfile, *stored.ReviewedBy, "the reviewer is the Admin's profile")
	assert.NotNil(t, stored.ReviewedAt)

	var embeddingsAfter int64
	require.NoError(t, api.db.Model(&entity.CardEmbedding{}).Where("card_id = ?", card).Count(&embeddingsAfter).Error)
	assert.Equal(t, embeddingsBefore, embeddingsAfter, "a rejection embeds nothing")

	assert.NotContains(t, queueIDs(t, api, token), row.ID, "a rejected photo leaves the queue for good")
	assert.Equal(t, http.StatusNotFound, api.Get(reviewQueuePath+"/"+row.ID.String()+"/photo", bearer(token)).Code)
}

func TestAdminReviews_RejectingAgainAnswersConflictAndKeepsTheFirstDecision(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	token, _ := adminCaller(t, api)
	_, profileID := feedbackCaller(t, api)
	profile, err := uuid.Parse(profileID)
	require.NoError(t, err)
	card := seedFeedbackCard(t, api)
	row := insertFeedback(t, api, correctionFor(t, profile, card, "switched", time.Now()))

	first := api.Post(reviewQueuePath+"/"+row.ID.String()+"/reject", bearer(token))
	require.Equal(t, http.StatusNoContent, first.Code, first.Body.String())
	var before entity.ScanFeedback
	require.NoError(t, api.db.Where("id = ?", row.ID).Take(&before).Error)

	again := api.Post(reviewQueuePath+"/"+row.ID.String()+"/reject", bearer(token))

	assert.Equal(t, http.StatusConflict, again.Code, again.Body.String())
	var after entity.ScanFeedback
	require.NoError(t, api.db.Where("id = ?", row.ID).Take(&after).Error)
	assert.Equal(t, before.ReviewedAt.UnixMicro(), after.ReviewedAt.UnixMicro(), "the first decision stands")
}

func TestAdminReviews_RejectRefusesACorrectionOutsideTheQueue(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	token, _ := adminCaller(t, api)
	_, profileID := feedbackCaller(t, api)
	profile, err := uuid.Parse(profileID)
	require.NoError(t, err)
	card := seedFeedbackCard(t, api)
	confirmed := correctionFor(t, profile, card, "confirmed", time.Now())
	confirmed.ImageKey = nil
	confirmed = insertFeedback(t, api, confirmed)

	resp := api.Post(reviewQueuePath+"/"+confirmed.ID.String()+"/reject", bearer(token))

	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	var stored entity.ScanFeedback
	require.NoError(t, api.db.Where("id = ?", confirmed.ID).Take(&stored).Error)
	assert.Nil(t, stored.ReviewStatus, "a confirmation is never reviewed")
}

func refTime(t time.Time) *time.Time { return &t }
