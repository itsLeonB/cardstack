package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type matchEnvelope struct {
	Data dto.MatchResult `json:"data"`
}

var testJPEG = []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}

// TestScanMatchFlow covers the stub's contract and its guard; the stub's
// branches live in the service unit tests.
func TestScanMatchFlow(t *testing.T) {
	api := newTestAPI(t)
	newTestCards(t, 5)
	token := api.newUserToken(t)
	const path = "/scan/match"
	jpeg := "Content-Type: image/jpeg"

	t.Run("requires a valid token", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, api.Post(path, jpeg, bytes.NewReader(testJPEG)).Code)
		assert.Equal(t, http.StatusUnauthorized, api.Post(path, jpeg, bearer("not-a-token"), bytes.NewReader(testJPEG)).Code)
	})

	t.Run("a valid image returns both shapes over repeated calls", func(t *testing.T) {
		var confident, unsure int
		for range 40 {
			resp := api.Post(path, jpeg, bearer(token), bytes.NewReader(testJPEG))
			require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
			var body matchEnvelope
			require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))

			if body.Data.Confident {
				confident++
				require.Len(t, body.Data.Candidates, 1)
			} else {
				unsure++
				require.GreaterOrEqual(t, len(body.Data.Candidates), 3)
			}
			// Other tests share this database and may leave Cards with empty names.
			assert.NotEqual(t, uuid.Nil, body.Data.Candidates[0].Card.ID)
		}
		assert.Positive(t, confident)
		assert.Positive(t, unsure)
	})

	t.Run("an empty or non-JPEG body is a 400", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, api.Post(path, jpeg, bearer(token), bytes.NewReader(nil)).Code)
		assert.Equal(t, http.StatusBadRequest, api.Post(path, jpeg, bearer(token), bytes.NewReader([]byte("just text"))).Code)
		png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
		assert.Equal(t, http.StatusBadRequest, api.Post(path, jpeg, bearer(token), bytes.NewReader(png)).Code)
	})

	t.Run("an oversized body is rejected", func(t *testing.T) {
		big := append(append([]byte{}, testJPEG...), make([]byte, 3<<20)...)
		assert.Equal(t, http.StatusRequestEntityTooLarge, api.Post(path, jpeg, bearer(token), bytes.NewReader(big)).Code)
	})
}
