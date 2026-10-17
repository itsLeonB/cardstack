package tcgcollector

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSets_FixtureMapsCodesSeriesAndCounts(t *testing.T) {
	t.Parallel()

	page, err := os.ReadFile("testdata/sets-id.html")
	require.NoError(t, err)

	sets, err := parseSets(page)
	require.NoError(t, err)
	require.Len(t, sets, 8, "seven sets under known headings plus one under an unknown heading")

	byTCGCode := make(map[string]SetListing, len(sets))
	for _, set := range sets {
		byTCGCode[set.TCGCode] = set
	}

	mega := byTCGCode["MA6"]
	assert.Equal(t, "11928", mega.TCGCollectorID)
	assert.Equal(t, "MA6", mega.Code)
	assert.Equal(t, "30th Celebration", mega.Name)
	assert.Equal(t, 207, mega.CardCount)
	assert.Equal(t, time.Date(2026, time.September, 16, 0, 0, 0, 0, time.UTC), mega.ReleaseDate)
	assert.Equal(t, "https://static.tcgcollector.com/content/images/30/17/47/301747d3a8a3e62e6eb13c90a0b05cee33d1a466bbf57024e173fac995d5fa65.webp", mega.LogoURL, "widest srcset entry")
	assert.Equal(t, "Mega Evolution Series", mega.SeriesHeading)
	assert.Equal(t, "Evolusi Mega", mega.Series)

	assert.Equal(t, "SM-P", byTCGCode["SMP"].Code, "alias applied to the catalog code")
	assert.Equal(t, "Sun & Moon Promos", byTCGCode["SMP"].Name, "entity names are unescaped")
	assert.Equal(t, "AS4D", byTCGCode["AS4Di"].Code)
	assert.Equal(t, "Matahari & Bulan", byTCGCode["SMP"].Series)

	assert.Empty(t, byTCGCode["PGO"].Series, "an unknown heading leaves the Series empty")
	assert.Equal(t, "Pokémon GO Series", byTCGCode["PGO"].SeriesHeading)
	assert.Empty(t, byTCGCode["MP"].LogoURL, "a set without a logo has no URL")
}

func TestParseSets_PageWithoutSetsIsAnError(t *testing.T) {
	t.Parallel()

	_, err := parseSets([]byte("<html><body>Just a moment...</body></html>"))
	assert.Error(t, err)
}

func TestNormalizeSetCode(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"MP":    "M-P",
		"SVP":   "SV-P",
		"SMP":   "SM-P",
		"SP":    "S-P",
		"AS4Di": "AS4D",
		"AS3Di": "AS3D",
		"AS2Di": "AS2D",
		"S8a-P": "S8a-P",
		"SV4a":  "SV4a",
	}
	for code, want := range tests {
		assert.Equal(t, want, normalizeSetCode(code), code)
	}
}
