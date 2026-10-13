package mapper

import (
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestToAdminCard_ListsEveryTagAndNeverNull(t *testing.T) {
	host := NewImageHost(testImageBase)

	both := ToAdminCard(host, entity.Card{Tags: []string{"Item", "Supporter"}})
	none := ToAdminCard(host, entity.Card{})

	assert.Equal(t, []string{"Item", "Supporter"}, both.Tags)
	assert.NotNil(t, none.Tags, "a card without tags is an empty array, not null")
	assert.Empty(t, none.Tags)
}
