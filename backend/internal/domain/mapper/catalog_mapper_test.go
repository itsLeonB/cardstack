package mapper

import (
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/stretchr/testify/assert"
)

func TestToExpansionSetSummary_CopiesImageURL(t *testing.T) {
	got := ToExpansionSetSummary(entity.ExpansionSet{Code: "MA6", ImageURL: "https://example.test/x.png"})

	assert.Equal(t, "https://example.test/x.png", got.ImageURL)
}

func TestToCardSummary_CopiesExpansionSetImageURL(t *testing.T) {
	got := ToCardSummary(repository.CardResult{ExpansionSetImageURL: "https://example.test/x.png"})

	assert.Equal(t, "https://example.test/x.png", got.ExpansionSet.ImageURL)
}
