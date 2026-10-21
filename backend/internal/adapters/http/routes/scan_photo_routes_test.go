package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const scanPhotoPath = "/scan/photo-uploads"

type scanPhotoEnvelope struct {
	Data dto.ScanPhotoUploads `json:"data"`
}

// scanPhotoBody asks for count upload addresses.
func scanPhotoBody(count int) map[string]any {
	return map[string]any{"count": count}
}

// callerProfileID is the profile the Guard creates for the token's user on its
// first call, which the test reads back to know the key prefix it must get.
func (a testAPI) callerProfileID(t *testing.T, claims map[string]any) string {
	t.Helper()
	require.Equal(t, http.StatusOK, a.Get("/collections", bearer(a.sign(t, claims))).Code)
	users := a.usersWithSubject(t, claims["sub"].(string))
	require.Len(t, users, 1)
	profiles := a.profilesOf(t, users[0])
	require.Len(t, profiles, 1)
	return profiles[0].ID.String()
}

func TestScanPhotoUploads_SignsOneAddressPerPhotoUnderTheCallersPrefix(t *testing.T) {
	api := newTestAPI(t)
	claims := validClaims()
	token := api.sign(t, claims)
	prefix := api.callerProfileID(t, claims) + "/"
	expiresAt := time.Now().Add(5 * time.Minute).UTC().Truncate(time.Second)

	api.photos.EXPECT().PresignPut(mock.Anything, mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, prefix)
	}), "image/jpeg", mock.Anything).
		Return("https://photos.example.test/signed", expiresAt, nil).Times(3)

	resp := api.Post(scanPhotoPath, bearer(token), scanPhotoBody(3))

	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var body scanPhotoEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Len(t, body.Data.Uploads, 3)
	keys := map[string]bool{}
	for _, upload := range body.Data.Uploads {
		assert.True(t, strings.HasPrefix(upload.Key, prefix), "key %q is under the caller's prefix", upload.Key)
		assert.False(t, keys[upload.Key], "keys are not reused")
		keys[upload.Key] = true
		assert.Equal(t, "https://photos.example.test/signed", upload.URL)
		assert.True(t, expiresAt.Equal(upload.ExpiresAt), "expiry is the one the signer reports")
	}
}

func TestScanPhotoUploads_RefusesACountOutsideOneToOneHundred(t *testing.T) {
	api := newTestAPI(t)
	auth := bearer(api.newUserToken(t))
	api.photos.EXPECT().PresignPut(mock.Anything, mock.Anything, "image/jpeg", mock.Anything).
		Return("https://photos.example.test/signed", time.Now(), nil)

	assert.Equal(t, http.StatusUnprocessableEntity, api.Post(scanPhotoPath, auth, scanPhotoBody(0)).Code, "fewer than one")
	assert.Equal(t, http.StatusUnprocessableEntity, api.Post(scanPhotoPath, auth, scanPhotoBody(101)).Code, "more than 100")
	assert.Equal(t, http.StatusOK, api.Post(scanPhotoPath, auth, scanPhotoBody(1)).Code, "one photo")
	assert.Equal(t, http.StatusOK, api.Post(scanPhotoPath, auth, scanPhotoBody(100)).Code, "a full batch")
}

func TestScanPhotoUploads_RefusedClearlyWhenThePhotoBucketIsNotConfigured(t *testing.T) {
	api := newTestAPIWithoutPhotoBucket(t)
	auth := bearer(api.newUserToken(t))

	resp := api.Post(scanPhotoPath, auth, scanPhotoBody(1))

	assert.Equal(t, http.StatusServiceUnavailable, resp.Code, resp.Body.String())
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "photo_storage_not_configured", body.Code)
	assert.Equal(t, http.StatusOK, api.Get("/collections", auth).Code, "the rest of the API works")
}

func TestScanPhotoUploads_RejectsGuestsAndBadTokens(t *testing.T) {
	api := newTestAPI(t)

	assert.Equal(t, http.StatusUnauthorized, api.Post(scanPhotoPath, scanPhotoBody(1)).Code, "no token")
	assert.Equal(t, http.StatusUnauthorized, api.Post(scanPhotoPath, bearer("not-a-token"), scanPhotoBody(1)).Code, "invalid token")
}

func TestScanPhotoUploads_CountsAgainstThePerUserLimit(t *testing.T) {
	api, _ := newRateLimitedAPI(t)
	auth := bearer(api.newUserToken(t))
	api.photos.EXPECT().PresignPut(mock.Anything, mock.Anything, "image/jpeg", mock.Anything).
		Return("https://photos.example.test/signed", time.Now(), nil).Times(tightLimits.User.Burst)

	for i := range tightLimits.User.Burst {
		require.Equal(t, http.StatusOK, api.Post(scanPhotoPath, auth, scanPhotoBody(1)).Code, "request %d is under the limit", i+1)
	}
	assertRateLimited(t, api.Post(scanPhotoPath, auth, scanPhotoBody(1)), "1", "over the limit")
}
