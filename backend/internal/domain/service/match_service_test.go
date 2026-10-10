package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	"github.com/itsLeonB/ungerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	jpegBytes = []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}
	pngBytes  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	webpBytes = []byte("RIFF\x24\x00\x00\x00WEBPVP8 ")
	gifBytes  = []byte("GIF89a\x01\x00\x01\x00")
)

var testMatchSettings = MatchSettings{Model: "test-model", Threshold: 0.85, Margin: 0.03}

// matchFixture wires a MatchService over mocks. Embed answers queryVector.
type matchFixture struct {
	svc        MatchService
	embedder   *mocks.MockImageEmbedder
	embeddings *mocks.MockEmbeddingRepository
	cards      *mocks.MockMatchRepository
}

var queryVector = []float32{0.1, 0.2}

func newMatchFixture(t *testing.T) matchFixture {
	t.Helper()
	f := matchFixture{
		embedder:   mocks.NewMockImageEmbedder(t),
		embeddings: mocks.NewMockEmbeddingRepository(t),
		cards:      mocks.NewMockMatchRepository(t),
	}
	f.svc = NewMatchService(f.embedder, f.embeddings, f.cards, testImages, testMatchSettings)
	return f
}

// search makes the nearest-card search answer distances (nearest first) for a
// new Card each, and the card lookup return those Cards. It returns their ids.
func (f matchFixture) search(distances ...float64) []uuid.UUID {
	ids := make([]uuid.UUID, len(distances))
	nearest := make([]repository.CardDistance, len(distances))
	rows := make([]repository.CardResult, len(distances))
	for i, d := range distances {
		ids[i] = uuid.New()
		nearest[i] = repository.CardDistance{CardID: ids[i], Distance: d}
		rows[i] = repository.CardResult{ID: ids[i], Name: "Card", LocalID: "001", ExpansionSetName: "Set", ImageKey: "cards/x"}
	}
	f.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", jpegBytes).Return(queryVector, nil)
	f.embeddings.EXPECT().NearestCards(mock.Anything, testMatchSettings.Model, queryVector, matchCandidates).Return(nearest, nil)
	f.cards.EXPECT().CardsByIDs(mock.Anything, ids).Return(rows, nil)
	return ids
}

func (f matchFixture) match(t *testing.T) dto.MatchResult {
	t.Helper()
	got, err := f.svc.Match(context.Background(), dto.MatchRequest{Image: jpegBytes})
	require.NoError(t, err)
	return got
}

func candidateIDs(r dto.MatchResult) []uuid.UUID {
	ids := make([]uuid.UUID, len(r.Candidates))
	for i, c := range r.Candidates {
		ids[i] = c.Card.ID
	}
	return ids
}

func TestMatchService_Match_RejectsBadUploads(t *testing.T) {
	tests := map[string][]byte{
		"empty":     nil,
		"text":      []byte("definitely not an image"),
		"gif":       gifBytes,
		"png":       pngBytes,
		"webp":      webpBytes,
		"html":      []byte("<html><body>hi</body></html>"),
		"zero byte": {0},
	}
	for name, image := range tests {
		t.Run(name, func(t *testing.T) {
			f := newMatchFixture(t) // no mock expectation: nothing may be embedded or searched

			_, err := f.svc.Match(context.Background(), dto.MatchRequest{Image: image})

			appErr, ok := errors.AsType[ungerr.AppError](err)
			require.True(t, ok, "want an AppError, got %v", err)
			assert.Equal(t, http.StatusBadRequest, appErr.HttpStatus())
		})
	}
}

func TestMatchService_Match_ClearWinnerIsConfidentAndAloneInTheResult(t *testing.T) {
	f := newMatchFixture(t)
	ids := f.search(0.05, 0.30, 0.40) // similarities 0.95, 0.70, 0.60

	got := f.match(t)

	assert.True(t, got.Confident)
	assert.Equal(t, ids[:1], candidateIDs(got))
	assert.InDelta(t, 0.95, got.Candidates[0].Score, 1e-9)
	assert.Equal(t, "https://img.example.test/cards/x", got.Candidates[0].Card.ImageURL)
}

func TestMatchService_Match_NearTieOffersBothAndIsNotConfident(t *testing.T) {
	f := newMatchFixture(t)
	ids := f.search(0.05, 0.06) // 0.95 and 0.94: a reprint of the same artwork

	got := f.match(t)

	assert.False(t, got.Confident)
	assert.Equal(t, ids, candidateIDs(got))
}

func TestMatchService_Match_WeakTopScoreIsNotConfidentEvenWithoutARunnerUp(t *testing.T) {
	f := newMatchFixture(t)
	ids := f.search(0.40) // 0.60, below the threshold

	got := f.match(t)

	assert.False(t, got.Confident)
	assert.Equal(t, ids, candidateIDs(got))
}

func TestMatchService_Match_ASingleCandidateIsConfidentWhenItClearsTheThreshold(t *testing.T) {
	f := newMatchFixture(t)
	ids := f.search(0.10) // 0.90

	got := f.match(t)

	assert.True(t, got.Confident)
	assert.Equal(t, ids, candidateIDs(got))
}

func TestMatchService_Match_NothingEmbeddedYetIsAnEmptyNonConfidentResult(t *testing.T) {
	f := newMatchFixture(t)
	f.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", jpegBytes).Return(queryVector, nil)
	f.embeddings.EXPECT().NearestCards(mock.Anything, testMatchSettings.Model, queryVector, matchCandidates).Return(nil, nil)
	f.cards.EXPECT().CardsByIDs(mock.Anything, []uuid.UUID{}).Return(nil, nil)

	got := f.match(t)

	assert.False(t, got.Confident)
	assert.Empty(t, got.Candidates)
}

func TestMatchService_Match_ACardWithTwoEmbeddingsCountsOnceAtItsBestRow(t *testing.T) {
	f := newMatchFixture(t)
	a, b := uuid.New(), uuid.New()
	f.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", jpegBytes).Return(queryVector, nil)
	f.embeddings.EXPECT().NearestCards(mock.Anything, testMatchSettings.Model, queryVector, matchCandidates).
		Return([]repository.CardDistance{{CardID: a, Distance: 0.05}, {CardID: a, Distance: 0.06}, {CardID: b, Distance: 0.30}}, nil)
	f.cards.EXPECT().CardsByIDs(mock.Anything, []uuid.UUID{a, b}).
		Return([]repository.CardResult{{ID: b, ImageKey: "cards/b"}, {ID: a, ImageKey: "cards/a"}}, nil)

	got := f.match(t)

	// Counted twice, a's second row would be a 0.01 runner-up and block the match.
	assert.True(t, got.Confident)
	assert.Equal(t, []uuid.UUID{a}, candidateIDs(got))
}

func TestMatchService_Match_ScoreStaysInsideZeroToOne(t *testing.T) {
	t.Run("a distance rounding below zero scores 1", func(t *testing.T) {
		f := newMatchFixture(t)
		f.search(-0.0001)

		assert.Equal(t, 1.0, f.match(t).Candidates[0].Score)
	})

	t.Run("a distance past 1 scores 0", func(t *testing.T) {
		f := newMatchFixture(t)
		f.search(1.5)

		assert.Equal(t, 0.0, f.match(t).Candidates[0].Score)
	})
}

func TestMatchService_Match_ReturnsProviderErrorWithoutSearching(t *testing.T) {
	boom := errors.New("provider down")
	f := newMatchFixture(t)
	f.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", jpegBytes).Return(nil, boom)

	_, err := f.svc.Match(context.Background(), dto.MatchRequest{Image: jpegBytes})

	assert.Equal(t, boom, err)
}

func TestMatchService_Match_ReturnsSearchAndLookupErrors(t *testing.T) {
	boom := errors.New("db down")

	t.Run("search", func(t *testing.T) {
		f := newMatchFixture(t)
		f.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", jpegBytes).Return(queryVector, nil)
		f.embeddings.EXPECT().NearestCards(mock.Anything, testMatchSettings.Model, queryVector, matchCandidates).Return(nil, boom)

		_, err := f.svc.Match(context.Background(), dto.MatchRequest{Image: jpegBytes})

		assert.Equal(t, boom, err)
	})

	t.Run("card lookup", func(t *testing.T) {
		f := newMatchFixture(t)
		id := uuid.New()
		f.embedder.EXPECT().Embed(mock.Anything, "image/jpeg", jpegBytes).Return(queryVector, nil)
		f.embeddings.EXPECT().NearestCards(mock.Anything, testMatchSettings.Model, queryVector, matchCandidates).
			Return([]repository.CardDistance{{CardID: id, Distance: 0.1}}, nil)
		f.cards.EXPECT().CardsByIDs(mock.Anything, []uuid.UUID{id}).Return(nil, boom)

		_, err := f.svc.Match(context.Background(), dto.MatchRequest{Image: jpegBytes})

		assert.Equal(t, boom, err)
	})
}

func TestMatchSettings_confident(t *testing.T) {
	scores := func(s ...float64) []dto.MatchCandidate {
		out := make([]dto.MatchCandidate, len(s))
		for i, v := range s {
			out[i].Score = v
		}
		return out
	}
	m := MatchSettings{Threshold: 0.8, Margin: 0.1}

	tests := map[string]struct {
		candidates []dto.MatchCandidate
		want       bool
	}{
		"no candidates":                 {nil, false},
		"clear win":                     {scores(0.95, 0.70), true},
		"alone and above the threshold": {scores(0.81), true},
		"top below the threshold":       {scores(0.79, 0.10), false},
		"runner-up within the margin":   {scores(0.95, 0.90), false},
		"runner-up just outside it":     {scores(0.95, 0.80), true},
		"tie":                           {scores(0.95, 0.95), false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, m.confident(tt.candidates))
		})
	}
}
