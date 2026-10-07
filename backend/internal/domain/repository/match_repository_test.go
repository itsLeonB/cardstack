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

func TestMatchRepository_RandomCards(t *testing.T) {
	db := testDB(t)
	repo := NewMatchRepository(crud.NewRepository[entity.Card](db))
	f := newCatalogFixture(t, db)
	set := f.newExpansionSet(t, db, nil, nil)
	for i := range 6 {
		f.newCard(t, db, set.ID, func(c *entity.Card) { c.LocalID = fmt.Sprintf("%03d", i) })
	}

	got, err := repo.RandomCards(context.Background(), 4)

	require.NoError(t, err)
	require.Len(t, got, 4)
	for _, c := range got {
		// Shared database: other tests' Cards may have empty names, so check
		// the joined ids rather than names.
		assert.NotEqual(t, uuid.Nil, c.ID)
		assert.NotEqual(t, uuid.Nil, c.ExpansionSetID)
		assert.NotEqual(t, uuid.Nil, c.RarityID)
	}
}
