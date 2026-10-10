package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchRepository_CardsByIDs(t *testing.T) {
	db := testDB(t)
	repo := NewMatchRepository(crud.NewRepository[entity.Card](db))
	f := newCatalogFixture(t, db)
	set := f.newExpansionSet(t, db, nil, nil)
	cards := make([]entity.Card, 4)
	for i := range cards {
		cards[i] = f.newCard(t, db, set.ID, func(c *entity.Card) { c.LocalID = fmt.Sprintf("%03d", i) })
	}

	got, err := repo.CardsByIDs(context.Background(), []uuid.UUID{cards[0].ID, cards[2].ID, uuid.New()})

	require.NoError(t, err)
	ids := make([]uuid.UUID, len(got))
	for i, c := range got {
		ids[i] = c.ID
		// Shared database: other tests' Cards may have empty names, so check
		// the joined ids rather than names.
		assert.Equal(t, set.ID, c.ExpansionSetID)
		assert.NotEqual(t, uuid.Nil, c.RarityID)
	}
	assert.ElementsMatch(t, []uuid.UUID{cards[0].ID, cards[2].ID}, ids, "only the Cards asked for, and none for an unknown id")
}

func TestMatchRepository_CardsByIDs_NoIDsIsEmpty(t *testing.T) {
	db := testDB(t)
	repo := NewMatchRepository(crud.NewRepository[entity.Card](db))

	got, err := repo.CardsByIDs(context.Background(), nil)

	require.NoError(t, err)
	assert.Empty(t, got)
}
