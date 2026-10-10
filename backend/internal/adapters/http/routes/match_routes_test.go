package routes

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type matchEnvelope struct {
	Data dto.MatchResult `json:"data"`
}

var testJPEG = []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}

const matchPath = "/scan/match"

// axisVector is the unit vector along axis, the photo's embedding in these
// tests.
func axisVector(axis int) []float32 {
	v := make([]float32, embedding.Dimensions)
	v[axis] = 1
	return v
}

// similarTo is a unit vector whose cosine similarity to axisVector(0) is
// similarity.
func similarTo(similarity float64) []float32 {
	v := make([]float32, embedding.Dimensions)
	v[0] = float32(similarity)
	v[1] = float32(math.Sqrt(1 - similarity*similarity))
	return v
}

// storeEmbedding stores a catalog embedding of card for model.
func (a testAPI) storeEmbedding(t *testing.T, card uuid.UUID, model string, vector []float32) {
	t.Helper()
	row := entity.CardEmbedding{CardID: card, Model: model, Source: "catalog", Embedding: pgvector.NewVector(vector)}
	require.NoError(t, a.db.Create(&row).Error)
}

// scan posts the test photo and returns the response.
func (a testAPI) scan(t *testing.T, token string) (int, dto.MatchResult) {
	t.Helper()
	resp := a.Post(matchPath, "Content-Type: image/jpeg", bearer(token), bytes.NewReader(testJPEG))
	var body matchEnvelope
	if resp.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	}
	return resp.Code, body.Data
}

func matchedIDs(r dto.MatchResult) []uuid.UUID {
	ids := make([]uuid.UUID, len(r.Candidates))
	for i, c := range r.Candidates {
		ids[i] = c.Card.ID
	}
	return ids
}

// TestScanMatchFlow covers the contract, the auth guard and the matcher's
// outcomes against stored embeddings in real Postgres, with the provider
// faked; the confident rule's branches live in the service unit tests.
func TestScanMatchFlow(t *testing.T) {
	jpeg := "Content-Type: image/jpeg"

	t.Run("requires a valid token", func(t *testing.T) {
		api := newTestAPI(t)

		assert.Equal(t, http.StatusUnauthorized, api.Post(matchPath, jpeg, bytes.NewReader(testJPEG)).Code)
		assert.Equal(t, http.StatusUnauthorized, api.Post(matchPath, jpeg, bearer("not-a-token"), bytes.NewReader(testJPEG)).Code)
	})

	t.Run("a photo closest to one Card by a clear margin is confident", func(t *testing.T) {
		api := newTestAPI(t)
		cards := newTestCards(t, 4)
		api.storeEmbedding(t, cards[0], api.model, similarTo(0.95))
		api.storeEmbedding(t, cards[1], api.model, similarTo(0.70))
		api.storeEmbedding(t, cards[2], api.model, similarTo(0.60))
		// cards[3] has no embedding, and the other model's closer vector is not searched.
		api.storeEmbedding(t, cards[3], "other-"+api.model, similarTo(0.99))
		api.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", testJPEG).Return(axisVector(0), nil)

		code, got := api.scan(t, api.newUserToken(t))

		require.Equal(t, http.StatusOK, code)
		assert.True(t, got.Confident)
		assert.Equal(t, cards[:1], matchedIDs(got))
		assert.InDelta(t, 0.95, got.Candidates[0].Score, 1e-3)
	})

	t.Run("a near-tie offers both Cards and is not confident", func(t *testing.T) {
		api := newTestAPI(t)
		cards := newTestCards(t, 3)
		api.storeEmbedding(t, cards[0], api.model, similarTo(0.95))
		api.storeEmbedding(t, cards[1], api.model, similarTo(0.94))
		api.storeEmbedding(t, cards[2], api.model, similarTo(0.50))
		api.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", testJPEG).Return(axisVector(0), nil)

		code, got := api.scan(t, api.newUserToken(t))

		require.Equal(t, http.StatusOK, code)
		assert.False(t, got.Confident)
		assert.Equal(t, cards, matchedIDs(got), "ranked by score, highest first")
	})

	t.Run("a weak top score is not confident", func(t *testing.T) {
		api := newTestAPI(t)
		cards := newTestCards(t, 1)
		api.storeEmbedding(t, cards[0], api.model, similarTo(0.60))
		api.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", testJPEG).Return(axisVector(0), nil)

		code, got := api.scan(t, api.newUserToken(t))

		require.Equal(t, http.StatusOK, code)
		assert.False(t, got.Confident)
		assert.Equal(t, cards, matchedIDs(got))
	})

	t.Run("a catalog with nothing embedded matches nothing", func(t *testing.T) {
		api := newTestAPI(t)
		api.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", testJPEG).Return(axisVector(0), nil)

		code, got := api.scan(t, api.newUserToken(t))

		require.Equal(t, http.StatusOK, code)
		assert.False(t, got.Confident)
		assert.Empty(t, got.Candidates)
	})

	t.Run("a provider failure is a redacted 500", func(t *testing.T) {
		api := newTestAPI(t)
		api.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", testJPEG).
			Return(nil, errors.New("gemini quota exceeded for key AIza-secret"))

		resp := api.Post(matchPath, jpeg, bearer(api.newUserToken(t)), bytes.NewReader(testJPEG))

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.NotContains(t, resp.Body.String(), "gemini")
		assert.NotContains(t, resp.Body.String(), "AIza")
	})

	t.Run("an empty or non-JPEG body is a 400 and never reaches the provider", func(t *testing.T) {
		api := newTestAPI(t)
		token := api.newUserToken(t)

		assert.Equal(t, http.StatusBadRequest, api.Post(matchPath, jpeg, bearer(token), bytes.NewReader(nil)).Code)
		assert.Equal(t, http.StatusBadRequest, api.Post(matchPath, jpeg, bearer(token), bytes.NewReader([]byte("just text"))).Code)
		png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
		assert.Equal(t, http.StatusBadRequest, api.Post(matchPath, jpeg, bearer(token), bytes.NewReader(png)).Code)
	})

	t.Run("an oversized body is rejected", func(t *testing.T) {
		api := newTestAPI(t)
		big := append(append([]byte{}, testJPEG...), make([]byte, 3<<20)...)

		assert.Equal(t, http.StatusRequestEntityTooLarge, api.Post(matchPath, jpeg, bearer(api.newUserToken(t)), bytes.NewReader(big)).Code)
	})
}
