package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestCardRepository_CountsByExpansionSet_CountsOnlyTheGame(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	gameID := insertTestGame(t, db)
	otherGameID := insertTestGame(t, db)
	localeID := insertTestLocale(t, db)
	setID := insertTestSet(t, db, gameID, localeID)
	emptySetID := insertTestSet(t, db, gameID, localeID)
	otherSetID := insertTestSet(t, db, otherGameID, localeID)

	rarity, err := crud.NewRepository[entity.Rarity](db).Insert(ctx, entity.Rarity{
		GameID: gameID,
		Code:   uuid.NewString(),
		Name:   "test rarity",
	})
	require.NoError(t, err)

	insertCard := func(setID uuid.UUID, localID, imageKey string) {
		t.Helper()
		_, err := crud.NewRepository[entity.Card](db).Insert(ctx, entity.Card{
			CreatedBy:      entity.SourcePokemonAsia,
			UpdatedBy:      entity.SourcePokemonAsia,
			ExpansionSetID: setID,
			LocalID:        localID,
			Name:           "card",
			Category:       "Pokémon",
			RarityID:       rarity.ID,
			Tags:           datatypes.JSONSlice[string]{},
			Attributes:     datatypes.JSONMap{},
			ImageKey:       imageKey,
		})
		require.NoError(t, err)
	}
	insertCard(setID, "001", "")
	insertCard(setID, "002", "hosted.webp")
	insertCard(setID, "003", "")
	insertCard(otherSetID, "001", "")

	counts, err := NewCardRepository(crud.NewRepository[entity.Card](db)).CountsByExpansionSet(ctx, gameID)
	require.NoError(t, err)

	assert.Equal(t, CardCounts{Total: 3, WithoutImage: 2}, counts[setID])
	assert.NotContains(t, counts, otherSetID, "another Game's Cards are not counted")
	assert.NotContains(t, counts, emptySetID, "a set with no Cards is absent")
}

func insertTestGame(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	game, err := crud.NewRepository[entity.Game](db).Insert(context.Background(), entity.Game{
		Slug: uuid.NewString(),
		Name: "test game",
	})
	require.NoError(t, err)
	return game.ID
}

func insertTestLocale(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	locale, err := crud.NewRepository[entity.Locale](db).Insert(context.Background(), entity.Locale{Code: uuid.NewString()})
	require.NoError(t, err)
	return locale.ID
}

func insertTestSet(t *testing.T, db *gorm.DB, gameID, localeID uuid.UUID) uuid.UUID {
	t.Helper()
	set, err := crud.NewRepository[entity.ExpansionSet](db).Insert(context.Background(), entity.ExpansionSet{
		CreatedBy: entity.SourcePokemonAsia,
		UpdatedBy: entity.SourcePokemonAsia,
		GameID:    gameID,
		Code:      uuid.NewString(),
		Name:      "test set",
		LocaleID:  localeID,
	})
	require.NoError(t, err)
	return set.ID
}
