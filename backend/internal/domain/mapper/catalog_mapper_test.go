package mapper

import (
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestToExpansionSetSummary_CopiesImageURL(t *testing.T) {
	got := ToExpansionSetSummary(entity.ExpansionSet{Code: "MA6", ImageURL: "https://example.test/x.png"})

	assert.Equal(t, "https://example.test/x.png", got.ImageURL)
}
