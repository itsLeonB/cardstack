package repository

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var rawColumn = regexp.MustCompile(`\braw\b`)

// selectsRaw is true for a query that names the raw column or reads every
// column with SELECT *, which includes it.
func selectsRaw(sql string) bool {
	return rawColumn.MatchString(sql) || strings.Contains(sql, "SELECT *")
}

// captureSQL records the SQL of every query run on db from now on.
func captureSQL(t *testing.T, db *gorm.DB) *[]string {
	t.Helper()

	var statements []string
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("test:capture_sql", func(tx *gorm.DB) {
		statements = append(statements, tx.Statement.SQL.String())
	}))
	return &statements
}

// The raw HTML is large and only tools read it, so no query behind a card
// endpoint a user calls may select it. The catalog search, the scan match,
// the inventory list and the inventory's card check cover those endpoints.
func TestUserFacingCardQueriesDoNotSelectRaw(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	statements := captureSQL(t, db)

	catalog := NewCatalogRepository(crud.NewRepository[entity.Card](db))
	_, _, err := catalog.SearchCards(ctx, CardFilter{Limit: 10})
	require.NoError(t, err)
	// The inventory list reads the same search with a Collection filter.
	_, _, err = catalog.SearchCards(ctx, CardFilter{CollectionID: uuid.New(), Limit: 10})
	require.NoError(t, err)
	_, err = NewMatchRepository(crud.NewRepository[entity.Card](db)).CardsByIDs(ctx, []uuid.UUID{uuid.New()})
	require.NoError(t, err)
	_, err = NewCardRepository(crud.NewRepository[entity.Card](db)).Exists(ctx, uuid.New())
	require.NoError(t, err)

	require.NotEmpty(t, *statements, "the queries ran through the capture")
	for _, sql := range *statements {
		assert.False(t, selectsRaw(sql), sql)
	}
}

// Control: a plain row read is SELECT * and so selects raw, which selectsRaw
// must catch.
func TestRawColumnCheckCatchesARowRead(t *testing.T) {
	db := testDB(t)
	statements := captureSQL(t, db)

	_, err := crud.NewRepository[entity.Card](db).FindAll(context.Background(), crud.Specification[entity.Card]{
		Model: entity.Card{ExpansionSetID: uuid.New()},
	})
	require.NoError(t, err)

	require.NotEmpty(t, *statements)
	assert.True(t, selectsRaw((*statements)[len(*statements)-1]))
}
