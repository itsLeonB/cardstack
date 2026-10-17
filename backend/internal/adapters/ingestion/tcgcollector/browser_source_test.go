package tcgcollector

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCheckPageURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"sets page", "https://www.tcgcollector.com/sets/id", false},
		{"card list", "https://www.tcgcollector.com/sets/id/SM-P", false},
		{"page parameter", "https://www.tcgcollector.com/sets/id?page=2", true},
		{"empty page parameter", "https://www.tcgcollector.com/sets/id?page=", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkPageURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsChallenge(t *testing.T) {
	challenge := `<html><head><title>Just a moment...</title></head><body><div id="challenge-platform"></div></body></html>`
	page := `<html><head><title>Sets</title></head><body><main>Sets</main></body></html>`

	assert.True(t, isChallenge(challenge))
	assert.False(t, isChallenge(page))
}

func TestPaceWaitsForGapSinceLastRequest(t *testing.T) {
	s := &browserSource{gap: 50 * time.Millisecond, last: time.Now()}

	start := time.Now()
	assert.NoError(t, s.pace(context.Background()))
	assert.GreaterOrEqual(t, time.Since(start), 40*time.Millisecond)
}

func TestPaceReturnsAtOnceWhenNoRecentRequest(t *testing.T) {
	s := &browserSource{gap: time.Hour}

	start := time.Now()
	assert.NoError(t, s.pace(context.Background()))
	assert.Less(t, time.Since(start), time.Second)
}

func TestFetchRefusesPageParameterBeforeLoading(t *testing.T) {
	source, cleanup := NewBrowserSource(t.TempDir())
	defer cleanup()

	_, err := source.Fetch(context.Background(), "https://www.tcgcollector.com/sets/id?page=2")

	assert.Error(t, err)
}
