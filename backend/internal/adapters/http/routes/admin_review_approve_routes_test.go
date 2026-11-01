package routes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// reviewedKey is the one key the Admin's cut of a Correction may be uploaded to
// and approved from (docs/adr/0027).
func reviewedKey(id uuid.UUID) string { return "reviewed/" + id.String() + ".jpg" }

// approvedKey is the server-only key the bytes embedded on approval are kept
// under. No presigned address is ever minted for it (docs/adr/0027).
func approvedKey(id uuid.UUID) string { return "approved/" + id.String() + ".jpg" }

func approveBody(t *testing.T, key string) string {
	t.Helper()
	return mustJSON(t, map[string]string{"key": key})
}

// approve posts the approval of the Correction id, naming key.
func (a testAPI) approve(t *testing.T, token string, id uuid.UUID, key string) *httptest.ResponseRecorder {
	t.Helper()
	return a.Post(reviewQueuePath+"/"+id.String()+"/approve", "Content-Type: application/json", bearer(token), strings.NewReader(approveBody(t, key)))
}

// expectCut sets the bucket and embedder expectations of one approval of the
// Correction id: the cut's size and bytes, the vector those bytes embed to, and
// the write of those bytes under the kept key.
func (a testAPI) expectCut(t *testing.T, id uuid.UUID, data []byte, vector []float32) {
	t.Helper()
	a.photos.EXPECT().Size(mock.Anything, reviewedKey(id)).Return(int64(len(data)), nil).Once()
	a.photos.EXPECT().Get(mock.Anything, reviewedKey(id)).Return(data, nil).Once()
	a.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", data).Return(vector, nil).Once()
	a.photos.EXPECT().Put(mock.Anything, approvedKey(id), "image/jpeg", data).Return(nil).Once()
}

// reviewFixture is a user who has Corrections and one Card they chose.
func reviewFixture(t *testing.T, api testAPI) (uuid.UUID, uuid.UUID) {
	t.Helper()
	_, profileID := feedbackCaller(t, api)
	profile, err := uuid.Parse(profileID)
	require.NoError(t, err)
	return profile, seedFeedbackCard(t, api)
}

// embeddingsOf returns the user-supplied embeddings of the Correction id.
func embeddingsOf(t *testing.T, api testAPI, id uuid.UUID) []entity.CardEmbedding {
	t.Helper()
	var rows []entity.CardEmbedding
	require.NoError(t, api.db.Where("source = ?", "correction:"+id.String()).Find(&rows).Error)
	return rows
}

func storedReview(t *testing.T, api testAPI, id uuid.UUID) entity.ScanFeedback {
	t.Helper()
	var row entity.ScanFeedback
	require.NoError(t, api.db.Where("id = ?", id).Take(&row).Error)
	return row
}

func TestAdminReviews_UploadAddressIsSignedForAPendingCorrectionOnly(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	token, _ := adminCaller(t, api)
	profile, card := reviewFixture(t, api)
	pending := insertFeedback(t, api, correctionFor(t, profile, card, "switched", time.Now()))
	decided := correctionFor(t, profile, card, "switched", time.Now())
	decided.ReviewStatus, decided.ReviewedAt = refString("approved"), refTime(time.Now())
	decided = insertFeedback(t, api, decided)
	confirmed := correctionFor(t, profile, card, "confirmed", time.Now())
	confirmed.ImageKey = nil
	confirmed = insertFeedback(t, api, confirmed)

	const signed = "https://photos.example.test/cut"
	api.photos.EXPECT().PresignPut(mock.Anything, reviewedKey(pending.ID), "image/jpeg", mock.MatchedBy(func(ttl time.Duration) bool {
		return ttl > 0 && ttl <= 5*time.Minute
	})).Return(signed, time.Now().Add(5*time.Minute), nil).Once()

	resp := api.Post(reviewQueuePath+"/"+pending.ID.String()+"/upload", bearer(token))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var body struct {
		Data struct {
			Key string `json:"key"`
			URL string `json:"url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, reviewedKey(pending.ID), body.Data.Key, "the server chooses the key")
	assert.Equal(t, signed, body.Data.URL)

	// No bucket expectation is set for the refusals below: a signing call fails the test.
	assert.Equal(t, http.StatusConflict, api.Post(reviewQueuePath+"/"+decided.ID.String()+"/upload", bearer(token)).Code, "a decided Correction's cut is never overwritten")
	assert.Equal(t, http.StatusNotFound, api.Post(reviewQueuePath+"/"+confirmed.ID.String()+"/upload", bearer(token)).Code, "a confirmation has no photo to cut")
	assert.Equal(t, http.StatusNotFound, api.Post(reviewQueuePath+"/"+uuid.NewString()+"/upload", bearer(token)).Code, "an unknown Correction")
	assert.Equal(t, http.StatusNotFound, api.Post(reviewQueuePath+"/"+pending.ID.String()+"/upload", bearer(api.newUserToken(t))).Code, "a user-origin token gets the Admin not-found")
}

func TestAdminReviews_ApproveEmbedsTheCutForTheChosenCardOnly(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	token, adminProfile := adminCaller(t, api)
	profile, card := reviewFixture(t, api)
	other := seedFeedbackCard(t, api)
	row := insertFeedback(t, api, correctionFor(t, profile, card, "switched", time.Now()))
	key := reviewedKey(row.ID)
	api.expectCut(t, row.ID, testJPEG, axisVector(0))

	// The bucket has no delete expectation: the cut stays in the bucket.
	assert.Equal(t, http.StatusNotFound, api.approve(t, api.newUserToken(t), row.ID, key).Code, "a user-origin token gets the Admin not-found")

	resp := api.approve(t, token, row.ID, key)
	require.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())

	stored := storedReview(t, api, row.ID)
	require.NotNil(t, stored.ReviewStatus)
	assert.Equal(t, "approved", *stored.ReviewStatus)
	require.NotNil(t, stored.ReviewedBy)
	assert.Equal(t, adminProfile, *stored.ReviewedBy, "the reviewer is the Admin's profile")
	assert.NotNil(t, stored.ReviewedAt)

	embeddings := embeddingsOf(t, api, row.ID)
	require.Len(t, embeddings, 1, "exactly one embedding, for the chosen Card")
	assert.Equal(t, card, embeddings[0].CardID)
	assert.Equal(t, api.model, embeddings[0].Model, "the configured model")
	assert.Equal(t, axisVector(0), embeddings[0].Embedding.Slice())
	var otherRows int64
	require.NoError(t, api.db.Model(&entity.CardEmbedding{}).Where("card_id = ? AND source LIKE ?", other, "correction:%").Count(&otherRows).Error)
	assert.Zero(t, otherRows, "the Card's Reprint gets no embedding")

	assert.NotContains(t, queueIDs(t, api, token), row.ID, "an approved photo leaves the queue")
	assert.Equal(t, http.StatusNotFound, api.Get(reviewQueuePath+"/"+row.ID.String()+"/photo", bearer(token)).Code)
}

func TestAdminReviews_ApproveRefusesAKeyOutsideTheItemOrANonJPEGOrAnOversizedObject(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	token, _ := adminCaller(t, api)
	profile, card := reviewFixture(t, api)

	// No bucket or embedder expectation is set on any row here: a refusal reads
	// no object and embeds nothing.
	t.Run("a key outside the item's prefix", func(t *testing.T) {
		row := insertFeedback(t, api, correctionFor(t, profile, card, "switched", time.Now()))
		for _, key := range []string{
			"reviewed/" + uuid.NewString() + ".jpg",
			"reviewed/" + row.ID.String() + "-other.jpg",
			profile.String() + "/" + uuid.NewString() + ".jpg",
		} {
			resp := api.approve(t, token, row.ID, key)
			assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, "key %q: %s", key, resp.Body.String())
		}
		assert.Nil(t, storedReview(t, api, row.ID).ReviewStatus)
	})

	t.Run("an object that is not there", func(t *testing.T) {
		row := insertFeedback(t, api, correctionFor(t, profile, card, "switched", time.Now()))
		api.photos.EXPECT().Size(mock.Anything, reviewedKey(row.ID)).Return(int64(0), objectstore.ErrNotFound).Once()

		resp := api.approve(t, token, row.ID, reviewedKey(row.ID))
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Nil(t, storedReview(t, api, row.ID).ReviewStatus)
	})

	t.Run("an object over the size cap", func(t *testing.T) {
		row := insertFeedback(t, api, correctionFor(t, profile, card, "switched", time.Now()))
		api.photos.EXPECT().Size(mock.Anything, reviewedKey(row.ID)).Return(int64(service.MaxFeedbackPhotoBytes+1), nil).Once()

		resp := api.approve(t, token, row.ID, reviewedKey(row.ID))
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Nil(t, storedReview(t, api, row.ID).ReviewStatus)
	})

	t.Run("an object that grows past the cap after its size was read", func(t *testing.T) {
		row := insertFeedback(t, api, correctionFor(t, profile, card, "switched", time.Now()))
		grown := append([]byte{0xff, 0xd8, 0xff}, make([]byte, service.MaxFeedbackPhotoBytes)...)
		api.photos.EXPECT().Size(mock.Anything, reviewedKey(row.ID)).Return(int64(len(testJPEG)), nil).Once()
		api.photos.EXPECT().Get(mock.Anything, reviewedKey(row.ID)).Return(grown, nil).Once()

		resp := api.approve(t, token, row.ID, reviewedKey(row.ID))
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Nil(t, storedReview(t, api, row.ID).ReviewStatus)
		assert.Empty(t, embeddingsOf(t, api, row.ID))
	})

	t.Run("an object that is not a JPEG", func(t *testing.T) {
		row := insertFeedback(t, api, correctionFor(t, profile, card, "switched", time.Now()))
		png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
		api.photos.EXPECT().Size(mock.Anything, reviewedKey(row.ID)).Return(int64(len(png)), nil).Once()
		api.photos.EXPECT().Get(mock.Anything, reviewedKey(row.ID)).Return(png, nil).Once()

		resp := api.approve(t, token, row.ID, reviewedKey(row.ID))
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Nil(t, storedReview(t, api, row.ID).ReviewStatus)
		assert.Empty(t, embeddingsOf(t, api, row.ID))
	})
}

func TestAdminReviews_AFailedEmbeddingStoresNothingAndTheItemCanBeRetried(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	token, _ := adminCaller(t, api)
	profile, card := reviewFixture(t, api)
	row := insertFeedback(t, api, correctionFor(t, profile, card, "switched", time.Now()))
	key := reviewedKey(row.ID)

	api.photos.EXPECT().Size(mock.Anything, key).Return(int64(len(testJPEG)), nil).Twice()
	api.photos.EXPECT().Get(mock.Anything, key).Return(testJPEG, nil).Twice()
	api.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", testJPEG).Return(nil, errors.New("embedding service down")).Once()
	api.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", testJPEG).Return(axisVector(0), nil).Once()
	api.photos.EXPECT().Put(mock.Anything, approvedKey(row.ID), "image/jpeg", testJPEG).Return(nil).Once()

	failed := api.approve(t, token, row.ID, key)
	assert.Equal(t, http.StatusInternalServerError, failed.Code, failed.Body.String())
	assert.Nil(t, storedReview(t, api, row.ID).ReviewStatus, "the item stays pending")
	assert.Empty(t, embeddingsOf(t, api, row.ID), "nothing is stored")
	assert.Contains(t, queueIDs(t, api, token), row.ID, "the item stays in the queue")

	retried := api.approve(t, token, row.ID, key)
	require.Equal(t, http.StatusNoContent, retried.Code, retried.Body.String())
	assert.Len(t, embeddingsOf(t, api, row.ID), 1)
}

func TestAdminReviews_ApprovingAgainAnswersConflictAndEmbedsNothingMore(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	token, _ := adminCaller(t, api)
	profile, card := reviewFixture(t, api)
	row := insertFeedback(t, api, correctionFor(t, profile, card, "switched", time.Now()))
	key := reviewedKey(row.ID)
	api.expectCut(t, row.ID, testJPEG, axisVector(0))
	require.Equal(t, http.StatusNoContent, api.approve(t, token, row.ID, key).Code)

	// The second approval, and a rejection after it, read no bucket and embed nothing.
	again := api.approve(t, token, row.ID, key)
	assert.Equal(t, http.StatusConflict, again.Code, again.Body.String())
	assert.Equal(t, http.StatusConflict, api.Post(reviewQueuePath+"/"+row.ID.String()+"/reject", bearer(token)).Code)
	assert.Len(t, embeddingsOf(t, api, row.ID), 1, "one embedding")
	assert.Equal(t, "approved", *storedReview(t, api, row.ID).ReviewStatus, "the first decision stands")
}

func TestAdminReviews_TheKeptCutIsTheEmbeddedBytesAndALaterUploadDoesNotChangeIt(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	token, _ := adminCaller(t, api)
	profile, card := reviewFixture(t, api)
	row := insertFeedback(t, api, correctionFor(t, profile, card, "switched", time.Now()))
	key := reviewedKey(row.ID)

	// A map stands in for the private bucket, keyed as the service keys it. No
	// presigned address is minted for the kept key: no expectation covers one.
	bucket := map[string][]byte{key: testJPEG}
	api.photos.EXPECT().Size(mock.Anything, key).Return(int64(len(testJPEG)), nil).Once()
	api.photos.EXPECT().Get(mock.Anything, key).Return(testJPEG, nil).Once()
	api.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", testJPEG).Return(axisVector(0), nil).Once()
	api.photos.EXPECT().Put(mock.Anything, approvedKey(row.ID), "image/jpeg", testJPEG).RunAndReturn(func(_ context.Context, k, _ string, body []byte) error {
		bucket[k] = body
		return nil
	}).Once()

	require.Equal(t, http.StatusNoContent, api.approve(t, token, row.ID, key).Code)

	// A late upload to the reviewed key after approval replaces the Admin's cut.
	bucket[key] = append([]byte{0xff, 0xd8, 0xff}, []byte("a later upload")...)

	assert.Equal(t, testJPEG, bucket[approvedKey(row.ID)], "the kept cut is the bytes that were embedded")
}

func TestAdminReviews_AFailedKeptCutWriteStoresNothingAndTheItemStaysPending(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	token, _ := adminCaller(t, api)
	profile, card := reviewFixture(t, api)
	row := insertFeedback(t, api, correctionFor(t, profile, card, "switched", time.Now()))
	key := reviewedKey(row.ID)

	api.photos.EXPECT().Size(mock.Anything, key).Return(int64(len(testJPEG)), nil).Twice()
	api.photos.EXPECT().Get(mock.Anything, key).Return(testJPEG, nil).Twice()
	api.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", testJPEG).Return(axisVector(0), nil).Twice()
	api.photos.EXPECT().Put(mock.Anything, approvedKey(row.ID), "image/jpeg", testJPEG).Return(errors.New("bucket write failed")).Once()
	api.photos.EXPECT().Put(mock.Anything, approvedKey(row.ID), "image/jpeg", testJPEG).Return(nil).Once()

	failed := api.approve(t, token, row.ID, key)
	assert.Equal(t, http.StatusInternalServerError, failed.Code, failed.Body.String())
	assert.Nil(t, storedReview(t, api, row.ID).ReviewStatus, "the item stays pending")
	assert.Empty(t, embeddingsOf(t, api, row.ID), "nothing is stored")
	assert.Contains(t, queueIDs(t, api, token), row.ID, "the item stays in the queue")

	retried := api.approve(t, token, row.ID, key)
	require.Equal(t, http.StatusNoContent, retried.Code, retried.Body.String())
	assert.Len(t, embeddingsOf(t, api, row.ID), 1)
}

func TestAdminReviews_AnApprovedPhotoRanksItsCardHigherForTheSamePhoto(t *testing.T) {
	api := newTestAPIWithAdmins(t, []string{adminEmail})
	token, _ := adminCaller(t, api)
	profile, _ := reviewFixture(t, api)
	cards := newTestCards(t, 2)
	chosen, rival := cards[0], cards[1]
	api.storeEmbedding(t, rival, api.model, similarTo(0.95))
	api.storeEmbedding(t, chosen, api.model, similarTo(0.90))
	row := insertFeedback(t, api, correctionFor(t, profile, chosen, "switched", time.Now()))
	key := reviewedKey(row.ID)

	// The photo embeds to axisVector(0) for both matches and for the approval.
	api.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", testJPEG).Return(axisVector(0), nil).Times(3)
	api.photos.EXPECT().Size(mock.Anything, key).Return(int64(len(testJPEG)), nil).Once()
	api.photos.EXPECT().Get(mock.Anything, key).Return(testJPEG, nil).Once()
	api.photos.EXPECT().Put(mock.Anything, approvedKey(row.ID), "image/jpeg", testJPEG).Return(nil).Once()

	userToken := api.newUserToken(t)
	code, before := api.scan(t, userToken)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, []uuid.UUID{rival, chosen}, matchedIDs(before), "before: the rival's catalog image is closer")

	require.Equal(t, http.StatusNoContent, api.approve(t, token, row.ID, key).Code)

	code, after := api.scan(t, userToken)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, []uuid.UUID{chosen, rival}, matchedIDs(after), "after: the approved photo lifts the chosen Card")
	assert.InDelta(t, 1.0, after.Candidates[0].Score, 1e-3)
}
