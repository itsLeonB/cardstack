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

func randomCardRows(n int) []repository.CardResult {
	rows := make([]repository.CardResult, n)
	for i := range rows {
		rows[i] = repository.CardResult{ID: uuid.New(), Name: "Card", LocalID: "001", ExpansionSetName: "Set", ImageKey: "cards/x"}
	}
	return rows
}

func TestMatchService_Match_RejectsBadUploads(t *testing.T) {
	tests := map[string][]byte{
		"empty":     nil,
		"text":      []byte("definitely not an image"),
		"gif":       gifBytes,
		"html":      []byte("<html><body>hi</body></html>"),
		"zero byte": {0},
	}
	for name, image := range tests {
		t.Run(name, func(t *testing.T) {
			svc := NewMatchService(mocks.NewMockMatchRepository(t), testImages)

			_, err := svc.Match(context.Background(), dto.MatchRequest{Image: image})

			appErr, ok := errors.AsType[ungerr.AppError](err)
			require.True(t, ok, "want an AppError, got %v", err)
			assert.Equal(t, http.StatusBadRequest, appErr.HttpStatus())
		})
	}
}

func TestMatchService_Match_AcceptsJPEGPNGAndWebP(t *testing.T) {
	for name, image := range map[string][]byte{"jpeg": jpegBytes, "png": pngBytes, "webp": webpBytes} {
		t.Run(name, func(t *testing.T) {
			repo := mocks.NewMockMatchRepository(t)
			repo.EXPECT().RandomCards(mock.Anything, mock.Anything).
				RunAndReturn(func(_ context.Context, n int) ([]repository.CardResult, error) { return randomCardRows(n), nil })
			svc := NewMatchService(repo, testImages)

			got, err := svc.Match(context.Background(), dto.MatchRequest{Image: image})

			require.NoError(t, err)
			assert.NotEmpty(t, got.Candidates)
		})
	}
}

// The stub flips between the two documented shapes; 200 calls make missing
// either one a (1/2)^200 fluke.
func TestMatchService_Match_ReturnsBothShapes(t *testing.T) {
	repo := mocks.NewMockMatchRepository(t)
	repo.EXPECT().RandomCards(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, n int) ([]repository.CardResult, error) { return randomCardRows(n), nil })
	svc := NewMatchService(repo, testImages)

	var confident, unsure int
	for range 200 {
		got, err := svc.Match(context.Background(), dto.MatchRequest{Image: jpegBytes})
		require.NoError(t, err)

		if got.Confident {
			confident++
			require.Len(t, got.Candidates, 1)
			assert.GreaterOrEqual(t, got.Candidates[0].Score, 0.9)
		} else {
			unsure++
			require.GreaterOrEqual(t, len(got.Candidates), 3)
			require.LessOrEqual(t, len(got.Candidates), 5)
		}
		for i, c := range got.Candidates {
			assert.Greater(t, c.Score, 0.0)
			assert.Less(t, c.Score, 1.0)
			assert.Equal(t, "https://img.example.test/cards/x", c.Card.ImageURL)
			if i > 0 {
				assert.Less(t, c.Score, got.Candidates[i-1].Score, "scores must descend")
			}
		}
	}
	assert.Positive(t, confident)
	assert.Positive(t, unsure)
}

func TestMatchService_Match_ReturnsRepositoryError(t *testing.T) {
	boom := errors.New("db down")
	repo := mocks.NewMockMatchRepository(t)
	repo.EXPECT().RandomCards(mock.Anything, mock.Anything).Return(nil, boom)
	svc := NewMatchService(repo, testImages)

	_, err := svc.Match(context.Background(), dto.MatchRequest{Image: jpegBytes})

	assert.Equal(t, boom, err)
}
