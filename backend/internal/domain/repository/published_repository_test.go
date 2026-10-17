package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// hideSet unpublishes set. Create would write the column default (true) for a
// false Published, so the hide has to be an update.
func hideSet(t *testing.T, db *gorm.DB, set entity.ExpansionSet) {
	t.Helper()
	require.NoError(t, db.Model(&set).Update("published", false).Error)
}

func TestCatalogRepository_UnpublishedSetsNeedIncludeUnpublished(t *testing.T) {
	db := testDB(t)
	repo := NewCatalogRepository(crud.NewRepository[entity.Card](db))
	ctx := context.Background()
	f := newCatalogFixture(t, db)

	series := f.newSeries(t, db)
	earlier := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	visible := f.newExpansionSet(t, db, &series.ID, &earlier)
	hidden := f.newExpansionSet(t, db, &series.ID, &later)
	hideSet(t, db, hidden)

	hiddenOnlySeries := f.newSeries(t, db)
	hiddenOnlySet := f.newExpansionSet(t, db, &hiddenOnlySeries.ID, nil)
	hideSet(t, db, hiddenOnlySet)

	category := "Cat-" + uuid.NewString()
	tag := "Tag-" + uuid.NewString()
	card := f.newCard(t, db, hidden.ID, func(c *entity.Card) {
		c.Category = category
		c.Tags = datatypes.JSONSlice[string]{tag}
	})

	t.Run("series", func(t *testing.T) {
		all, err := repo.ListSeries(ctx, true)
		require.NoError(t, err)
		assert.Contains(t, seriesIDsOf(all), hiddenOnlySeries.ID)

		published, err := repo.ListSeries(ctx, false)
		require.NoError(t, err)
		assert.Contains(t, seriesIDsOf(published), series.ID, "a Series with a published set stays")
		assert.NotContains(t, seriesIDsOf(published), hiddenOnlySeries.ID, "a Series whose sets are all unpublished is hidden")
	})

	t.Run("expansion sets", func(t *testing.T) {
		all, err := repo.ListExpansionSets(ctx, []uuid.UUID{series.ID}, true)
		require.NoError(t, err)
		assert.ElementsMatch(t, []uuid.UUID{visible.ID, hidden.ID}, setIDsOf(all))

		published, err := repo.ListExpansionSets(ctx, []uuid.UUID{series.ID}, false)
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{visible.ID}, setIDsOf(published))
	})

	t.Run("search and facets", func(t *testing.T) {
		filter := CardFilter{ExpansionSetIDs: []uuid.UUID{hidden.ID}, Limit: 10}

		_, total, err := repo.SearchCards(ctx, filter)
		require.NoError(t, err)
		assert.Zero(t, total)

		filter.IncludeUnpublished = true
		results, total, err := repo.SearchCards(ctx, filter)
		require.NoError(t, err)
		assert.EqualValues(t, 1, total)
		assert.Equal(t, card.ID, results[0].ID)

		// A selected unpublished set must not come back as a facet option for a non-Admin.
		filter.IncludeUnpublished = false
		facets, err := repo.ListCardFacets(ctx, filter)
		require.NoError(t, err)
		for _, opt := range facets.ExpansionSets {
			assert.NotEqual(t, hidden.ID, opt.ID)
		}
	})

	t.Run("categories and tags", func(t *testing.T) {
		categories, err := repo.ListDistinctCategories(ctx, false)
		require.NoError(t, err)
		assert.NotContains(t, categories, category)

		tags, err := repo.ListDistinctTags(ctx, false)
		require.NoError(t, err)
		assert.NotContains(t, tags, tag)

		categories, err = repo.ListDistinctCategories(ctx, true)
		require.NoError(t, err)
		assert.Contains(t, categories, category)

		tags, err = repo.ListDistinctTags(ctx, true)
		require.NoError(t, err)
		assert.Contains(t, tags, tag)
	})

	t.Run("inventory and master inventory", func(t *testing.T) {
		profile := newTestProfile(t, db, "Published Test")
		col := entity.Collection{ProfileID: profile.ID, Title: "Binder"}
		require.NoError(t, db.Create(&col).Error)
		require.NoError(t, db.Create(&entity.InventoryEntry{CollectionID: col.ID, CardID: card.ID, Quantity: 1}).Error)

		// Pickers never offer an unpublished Card, even to an Admin, since they are
		// user-facing views (GLOSSARY.md Published). Inventory leaves IncludeUnpublished unset.
		_, total, err := repo.SearchCards(ctx, CardFilter{CollectionID: col.ID, Limit: 10})
		require.NoError(t, err)
		assert.Zero(t, total)

		_, total, err = repo.SearchCards(ctx, CardFilter{ProfileID: profile.ID, Limit: 10})
		require.NoError(t, err)
		assert.Zero(t, total)
	})
}

func TestEmbeddingRepository_NearestCardsSkipsUnpublishedSets(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	visibleSet := fixture.newExpansionSet(t, db, nil, nil)
	hiddenSet := fixture.newExpansionSet(t, db, nil, nil)
	hideSet(t, db, hiddenSet)
	repo := NewEmbeddingRepository(crud.NewRepository[entity.CardEmbedding](db))
	ctx := context.Background()
	model := "published-model-" + uuid.NewString()

	hidden := newHostedCard(t, fixture, db, hiddenSet.ID, "cards/hidden-"+model)
	visible := newHostedCard(t, fixture, db, visibleSet.ID, "cards/visible-"+model)
	require.NoError(t, repo.Upsert(ctx, []entity.CardEmbedding{
		catalogEmbedding(hidden, model, unitVector(0)),
		catalogEmbedding(visible, model, mixedVector(0, 2)),
	}))

	// The hidden Card is the exact match, yet it must not take the one slot.
	got, err := repo.NearestCards(ctx, model, unitVector(0), 1)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, visible.ID, got[0].CardID)
}

func seriesIDsOf(series []entity.Series) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(series))
	for _, s := range series {
		ids = append(ids, s.ID)
	}
	return ids
}

func setIDsOf(sets []entity.ExpansionSet) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(sets))
	for _, s := range sets {
		ids = append(ids, s.ID)
	}
	return ids
}
