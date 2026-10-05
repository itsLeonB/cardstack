package mapper

import (
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/stretchr/testify/assert"
)

const testImageBase = "https://img.example.test"

func TestImageHost_URL(t *testing.T) {
	tests := []struct {
		name string
		base string
		key  string
		want string
	}{
		{"base plus key", testImageBase, "cards/abc", testImageBase + "/cards/abc"},
		{"trailing slash on base", testImageBase + "/", "cards/abc", testImageBase + "/cards/abc"},
		{"leading slash on key", testImageBase, "/cards/abc", testImageBase + "/cards/abc"},
		{"no key is empty, never a fallback", testImageBase, "", ""},
		{"no base is empty", "", "cards/abc", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewImageHost(tt.base).URL(tt.key))
		})
	}
}

func TestToExpansionSetSummary_BuildsImageURLFromKey(t *testing.T) {
	host := NewImageHost(testImageBase)

	hosted := ToExpansionSetSummary(host, entity.ExpansionSet{Code: "MA6", CoverKey: "expansion-sets/x.0123abcd.webp", ImageKey: "expansion-sets/orig", SourceImageURL: "https://source.test/x.png"})
	assert.Equal(t, testImageBase+"/expansion-sets/x.0123abcd.webp", hosted.ImageURL)

	unhosted := ToExpansionSetSummary(host, entity.ExpansionSet{Code: "MA6", ImageKey: "expansion-sets/orig", SourceImageURL: "https://source.test/x.png"})
	assert.Empty(t, unhosted.ImageURL, "a set with no cover key must not fall back to its original or source address")
}

func TestToSeriesSummary_BuildsImageURLFromKey(t *testing.T) {
	host := NewImageHost(testImageBase)

	withLogo := ToSeriesSummary(host, entity.Series{Code: "evolusi-mega", ImageKey: "series/evolusi-mega.0123abcd.webp"}, nil)
	assert.Equal(t, testImageBase+"/series/evolusi-mega.0123abcd.webp", withLogo.ImageURL)

	// No key means no logo: the frontend renders the text heading.
	assert.Empty(t, ToSeriesSummary(host, entity.Series{Code: "evolusi-mega"}, nil).ImageURL)
}

func TestToCardSummary_BuildsImageURLsFromKeys(t *testing.T) {
	host := NewImageHost(testImageBase)

	got := ToCardSummary(host, repository.CardResult{ImageKey: "cards/c1", ExpansionSetCoverKey: "expansion-sets/s1.0123abcd.webp"})
	assert.Equal(t, testImageBase+"/cards/c1", got.ImageURL)
	assert.Equal(t, testImageBase+"/expansion-sets/s1.0123abcd.webp", got.ExpansionSet.ImageURL)

	unhosted := ToCardSummary(host, repository.CardResult{})
	assert.Empty(t, unhosted.ImageURL)
	assert.Empty(t, unhosted.ExpansionSet.ImageURL)
}
