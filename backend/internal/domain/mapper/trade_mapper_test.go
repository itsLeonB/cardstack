package mapper

import (
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToTradeItemView covers the source Collection both ways: named while it
// exists, absent once it was deleted (the Trade Item outlives it).
func TestToTradeItemView(t *testing.T) {
	host := NewImageHost(testImageBase)
	sourceID := uuid.New()

	t.Run("with a source Collection", func(t *testing.T) {
		item := entity.TradeItem{
			SourceCollectionID: &sourceID,
			Quantity:           2,
			SourceCollection:   entity.Collection{Title: "Vintage Binder"},
		}

		got := ToTradeItemView(host, item)

		require.NotNil(t, got.Collection)
		assert.Equal(t, "Vintage Binder", *got.Collection)
		assert.Equal(t, 2, got.Quantity)
	})

	t.Run("source Collection deleted", func(t *testing.T) {
		got := ToTradeItemView(host, entity.TradeItem{Quantity: 1})

		assert.Nil(t, got.Collection)
	})
}
