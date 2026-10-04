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

	hosted := ToExpansionSetSummary(host, entity.ExpansionSet{Code: "MA6", ImageKey: "expansion-sets/x", SourceImageURL: "https://source.test/x.png"})
	assert.Equal(t, testImageBase+"/expansion-sets/x", hosted.ImageURL)

	unhosted := ToExpansionSetSummary(host, entity.ExpansionSet{Code: "MA6", SourceImageURL: "https://source.test/x.png"})
	assert.Empty(t, unhosted.ImageURL, "a set with no key must not fall back to its source address")
}

func TestToCardSummary_BuildsImageURLsFromKeys(t *testing.T) {
	host := NewImageHost(testImageBase)

	got := ToCardSummary(host, repository.CardResult{ImageKey: "cards/c1", ExpansionSetImageKey: "expansion-sets/s1"})
	assert.Equal(t, testImageBase+"/cards/c1", got.ImageURL)
	assert.Equal(t, testImageBase+"/expansion-sets/s1", got.ExpansionSet.ImageURL)

	unhosted := ToCardSummary(host, repository.CardResult{})
	assert.Empty(t, unhosted.ImageURL)
	assert.Empty(t, unhosted.ExpansionSet.ImageURL)
}
