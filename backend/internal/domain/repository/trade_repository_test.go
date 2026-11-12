package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTradeItem_SourceCollectionDeleteKeepsTheItem proves the trade_items FK
// is ON DELETE SET NULL: deleting the Collection a Trade Item names must not
// drop the item — it is part of the agreed deal, locked Trades included — and
// the row stays readable with a nil source.
func TestTradeItem_SourceCollectionDeleteKeepsTheItem(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	fixture := newCatalogFixture(t, db)
	set := fixture.newExpansionSet(t, db, nil, nil)
	card := fixture.newCard(t, db, set.ID, nil)
	profile := newTestProfile(t, db, "Trade Owner")
	col := newTestCollection(t, db)

	trade := entity.Trade{
		Slug:           "trade-" + uuid.NewString(),
		OwnerProfileID: profile.ID,
		Status:         entity.TradeStatusLocked,
	}
	require.NoError(t, db.Create(&trade).Error)

	sourceID := col.ID
	item := entity.TradeItem{
		TradeID:            trade.ID,
		Side:               entity.SideOwner,
		CardID:             card.ID,
		SourceCollectionID: &sourceID,
		Quantity:           1,
	}
	require.NoError(t, db.Create(&item).Error)

	require.NoError(t, db.Delete(&entity.Collection{}, "id = ?", col.ID).Error)

	var got entity.TradeItem
	require.NoError(t, db.First(&got, "id = ?", item.ID).Error)
	assert.Nil(t, got.SourceCollectionID)

	// The Trade still reads its item through the repository's preload path.
	repo := NewTradeRepository(crud.NewRepository[entity.Trade](db))
	found, err := repo.FindBySlug(ctx, trade.Slug)
	require.NoError(t, err)
	require.Len(t, found.TradeItems, 1)
	assert.Nil(t, found.TradeItems[0].SourceCollectionID)
}
