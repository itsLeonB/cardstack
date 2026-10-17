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

func TestExpansionSetRepository_ListMissingCovers(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	repo := NewExpansionSetRepository(crud.NewRepository[entity.ExpansionSet](db))

	withKeys := func(image, cover string) entity.ExpansionSet {
		set := fixture.newExpansionSet(t, db, nil, nil)
		require.NoError(t, db.Model(&set).Updates(map[string]any{"image_key": image, "cover_key": cover}).Error)
		return set
	}
	missing := withKeys("expansion-sets/a", "")
	done := withKeys("expansion-sets/b", "expansion-sets/b.0123abcd.webp")
	noOriginal := withKeys("", "")

	ids := func(sets []entity.ExpansionSet) []uuid.UUID {
		out := make([]uuid.UUID, len(sets))
		for i, s := range sets {
			out[i] = s.ID
		}
		return out
	}

	all, err := repo.ListMissingCovers(context.Background(), "")
	require.NoError(t, err)
	assert.Contains(t, ids(all), missing.ID)
	assert.NotContains(t, ids(all), done.ID, "a set that has its cover is skipped")
	assert.NotContains(t, ids(all), noOriginal.ID, "a set with no original cannot be resized")

	one, err := repo.ListMissingCovers(context.Background(), missing.Code)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{missing.ID}, ids(one))

	none, err := repo.ListMissingCovers(context.Background(), done.Code)
	require.NoError(t, err)
	assert.Empty(t, none)
}

// Insert would write the column default (published = true) for a zero
// Published, so InsertUnpublished must write the false explicitly.
func TestExpansionSetRepository_InsertUnpublished_StoresPublishedFalse(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := NewExpansionSetRepository(crud.NewRepository[entity.ExpansionSet](db))
	gameID := insertTestGame(t, db)
	localeID := insertTestLocale(t, db)

	set, err := repo.InsertUnpublished(ctx, entity.ExpansionSet{
		GameID:    gameID,
		Code:      uuid.NewString(),
		Name:      "Unpublished",
		LocaleID:  localeID,
		CreatedBy: entity.SourceTCGCollector,
		UpdatedBy: entity.SourceTCGCollector,
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, set.ID)
	assert.False(t, set.Published)

	stored, err := repo.FindFirst(ctx, crud.Specification[entity.ExpansionSet]{
		Model: entity.ExpansionSet{BaseEntity: crud.BaseEntity{ID: set.ID}},
	})
	require.NoError(t, err)
	assert.False(t, stored.Published)
	assert.Equal(t, entity.SourceTCGCollector, stored.CreatedBy)
}
