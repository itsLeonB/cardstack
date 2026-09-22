package tcgdex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngester_UpsertGame_Idempotent(t *testing.T) {
	in := NewIngester(testDB(t))
	ctx := context.Background()

	first, err := in.upsertGame(ctx)
	require.NoError(t, err)
	assert.Equal(t, gameSlug, first.Slug)

	second, err := in.upsertGame(ctx)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "second call must find the existing row, not create a duplicate")

	rows, err := in.games.FindAll(ctx, crud.Specification[entity.Game]{Model: entity.Game{Slug: gameSlug}})
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestIngester_UpsertExpansionSet_IdempotentAndUpdates(t *testing.T) {
	in := NewIngester(testDB(t))
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)

	code := uniqueCode(t)
	setRef := seriesSetRef{ID: code, Name: "Original Name"}

	first, err := in.upsertExpansionSet(ctx, game.ID, "id", setRef)
	require.NoError(t, err)
	assert.Equal(t, "Original Name", first.Name)

	setRef.Name = "Updated Name"
	second, err := in.upsertExpansionSet(ctx, game.ID, "id", setRef)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "must update the existing row, not create a duplicate")
	assert.Equal(t, "Updated Name", second.Name)

	rows, err := in.sets.FindAll(ctx, crud.Specification[entity.ExpansionSet]{
		Model: entity.ExpansionSet{GameID: game.ID, Code: code},
	})
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestIngester_ExpansionSet_UniqueConstraint(t *testing.T) {
	in := NewIngester(testDB(t))
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	locale, err := in.upsertLocale(ctx, "id")
	require.NoError(t, err)
	code := uniqueCode(t)

	_, err = in.sets.Insert(ctx, entity.ExpansionSet{GameID: game.ID, Code: code, Name: "A", LocaleID: locale.ID})
	require.NoError(t, err)

	_, err = in.sets.Insert(ctx, entity.ExpansionSet{GameID: game.ID, Code: code, Name: "B", LocaleID: locale.ID})
	assert.Error(t, err, "duplicate (game_id, code) must be rejected by the unique index")
}

func TestIngester_UpsertCard_IdempotentAndUpdates(t *testing.T) {
	in := NewIngester(testDB(t))
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	set, err := in.upsertExpansionSet(ctx, game.ID, "id", seriesSetRef{ID: uniqueCode(t), Name: "Set"})
	require.NoError(t, err)

	localID := "008"
	card, err := mapCard(cardResponse{LocalID: localID, Rarity: "Common", Image: "img1"}, set.ID, map[string]string{"id": "First"})
	require.NoError(t, err)

	first, err := in.upsertCard(ctx, card)
	require.NoError(t, err)
	assert.Equal(t, "Common", first.Rarity)

	updated, err := mapCard(cardResponse{LocalID: localID, Rarity: "Rare", Image: "img2"}, set.ID, map[string]string{"id": "First", "ja": "Second"})
	require.NoError(t, err)
	second, err := in.upsertCard(ctx, updated)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "must update the existing row, not create a duplicate")
	assert.Equal(t, "Rare", second.Rarity)
	assert.Equal(t, "img2", second.ImageURL)

	rows, err := in.cards.FindAll(ctx, crud.Specification[entity.Card]{
		Model: entity.Card{ExpansionSetID: set.ID, LocalID: localID},
	})
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestIngester_Card_UniqueConstraint(t *testing.T) {
	in := NewIngester(testDB(t))
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	set, err := in.upsertExpansionSet(ctx, game.ID, "id", seriesSetRef{ID: uniqueCode(t), Name: "Set"})
	require.NoError(t, err)

	localID := "001"
	_, err = in.cards.Insert(ctx, entity.Card{ExpansionSetID: set.ID, LocalID: localID, Names: mapNames(nil), Attributes: mapAttributes(cardResponse{})})
	require.NoError(t, err)

	_, err = in.cards.Insert(ctx, entity.Card{ExpansionSetID: set.ID, LocalID: localID, Names: mapNames(nil), Attributes: mapAttributes(cardResponse{})})
	assert.Error(t, err, "duplicate (expansion_set_id, local_id) must be rejected by the unique index")
}

func TestIngester_UpsertVariant_Idempotent(t *testing.T) {
	in := NewIngester(testDB(t))
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	set, err := in.upsertExpansionSet(ctx, game.ID, "id", seriesSetRef{ID: uniqueCode(t), Name: "Set"})
	require.NoError(t, err)
	card, err := in.upsertCard(ctx, entity.Card{ExpansionSetID: set.ID, LocalID: "001", Names: mapNames(nil), Attributes: mapAttributes(cardResponse{})})
	require.NoError(t, err)

	require.NoError(t, in.upsertVariant(ctx, card.ID, "holo"))
	require.NoError(t, in.upsertVariant(ctx, card.ID, "holo"))

	finishID, err := in.resolveFinishID(ctx, "holo")
	require.NoError(t, err)
	rows, err := in.variants.FindAll(ctx, crud.Specification[entity.CardVariant]{
		Model: entity.CardVariant{CardID: card.ID, FinishID: finishID},
	})
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestIngester_CardVariant_UniqueConstraint(t *testing.T) {
	in := NewIngester(testDB(t))
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	set, err := in.upsertExpansionSet(ctx, game.ID, "id", seriesSetRef{ID: uniqueCode(t), Name: "Set"})
	require.NoError(t, err)
	card, err := in.upsertCard(ctx, entity.Card{ExpansionSetID: set.ID, LocalID: "001", Names: mapNames(nil), Attributes: mapAttributes(cardResponse{})})
	require.NoError(t, err)

	finishID, err := in.resolveFinishID(ctx, "holo")
	require.NoError(t, err)

	_, err = in.variants.Insert(ctx, entity.CardVariant{CardID: card.ID, FinishID: finishID})
	require.NoError(t, err)

	_, err = in.variants.Insert(ctx, entity.CardVariant{CardID: card.ID, FinishID: finishID})
	assert.Error(t, err, "duplicate (card_id, finish_id) must be rejected by the unique index")
}

func TestIngester_CardVariant_FinishForeignKey(t *testing.T) {
	in := NewIngester(testDB(t))
	ctx := context.Background()

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	set, err := in.upsertExpansionSet(ctx, game.ID, "id", seriesSetRef{ID: uniqueCode(t), Name: "Set"})
	require.NoError(t, err)
	card, err := in.upsertCard(ctx, entity.Card{ExpansionSetID: set.ID, LocalID: "001", Names: mapNames(nil), Attributes: mapAttributes(cardResponse{})})
	require.NoError(t, err)

	_, err = in.variants.Insert(ctx, entity.CardVariant{CardID: card.ID, FinishID: uuid.New()})
	assert.Error(t, err, "the finish_id foreign key must reject a nonexistent finish")
}

// TestIngester_Run_EndToEnd exercises the full Run() flow (series -> set ->
// card -> variant, with bounded concurrency) against a fake TCGDex server,
// and confirms a second run is idempotent (no duplicate rows).
func TestIngester_Run_EndToEnd(t *testing.T) {
	setCode := uniqueCode(t)
	cardID := setCode + "-001"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/id/series/sv":
			_, _ = fmt.Fprintf(w, `{"id":"SV","name":"Scarlet & Violet","sets":[{"id":%q,"name":"Test Set"}]}`, setCode)
		case "/id/sets/" + setCode:
			_, _ = fmt.Fprintf(w, `{"cardCount":{"total":1,"official":1},"cards":[{"id":%q,"localId":"001","name":"Test Card","image":"https://example.com/001"}]}`, cardID)
		case "/id/cards/" + cardID:
			_, _ = fmt.Fprintf(w, `{"id":%q,"localId":"001","name":"Test Card","category":"Pokemon","rarity":"Common","image":"https://example.com/001","variants":{"normal":true}}`, cardID)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	db := testDB(t)
	in := NewIngester(db)
	in.client = &client{baseURL: server.URL, httpClient: server.Client()}
	ctx := context.Background()

	summary, err := in.Run(ctx, "id", "sv")
	require.NoError(t, err)
	assert.Equal(t, Summary{Sets: 1, Cards: 1, Variants: 1}, summary)

	// Re-running must be idempotent: same row counts, not duplicated.
	summary, err = in.Run(ctx, "id", "sv")
	require.NoError(t, err)
	assert.Equal(t, Summary{Sets: 1, Cards: 1, Variants: 1}, summary)

	game, err := in.upsertGame(ctx)
	require.NoError(t, err)
	set, err := in.sets.FindFirst(ctx, crud.Specification[entity.ExpansionSet]{
		Model: entity.ExpansionSet{GameID: game.ID, Code: setCode},
	})
	require.NoError(t, err)
	require.False(t, set.IsZero())

	cards, err := in.cards.FindAll(ctx, crud.Specification[entity.Card]{
		Model: entity.Card{ExpansionSetID: set.ID, LocalID: "001"},
	})
	require.NoError(t, err)
	require.Len(t, cards, 1)
	assert.Equal(t, "Common", cards[0].Rarity)
	assert.Equal(t, map[string]any{"id": "Test Card"}, map[string]any(cards[0].Names))

	variants, err := in.variants.FindAll(ctx, crud.Specification[entity.CardVariant]{
		Model: entity.CardVariant{CardID: cards[0].ID},
	})
	require.NoError(t, err)
	require.Len(t, variants, 1)
	finish, err := in.finishes.FindFirst(ctx, crud.Specification[entity.Finish]{
		Model: entity.Finish{BaseEntity: crud.BaseEntity{ID: variants[0].FinishID}},
	})
	require.NoError(t, err)
	assert.Equal(t, "normal", finish.Code)

	// The raw column must carry the full upstream card response through.
	assert.NotEmpty(t, cards[0].Raw)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(cards[0].Raw, &raw))
	assert.Equal(t, "Common", raw["rarity"])
}

// TestIngester_Run_ReportsCardCountMismatch confirms Run surfaces a set
// where TCGDex's own cardCount.official disagrees with the number of cards
// actually ingested, as an advisory Summary.Mismatches entry rather than a
// failure (see the plan's "cardCount is a sanity total to log/compare").
func TestIngester_Run_ReportsCardCountMismatch(t *testing.T) {
	setCode := uniqueCode(t)
	cardID := setCode + "-001"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/id/series/sv":
			_, _ = fmt.Fprintf(w, `{"id":"SV","name":"Scarlet & Violet","sets":[{"id":%q,"name":"Test Set"}]}`, setCode)
		case "/id/sets/" + setCode:
			// official (2) deliberately disagrees with the single card
			// actually listed, to exercise the mismatch path.
			_, _ = fmt.Fprintf(w, `{"cardCount":{"total":2,"official":2},"cards":[{"id":%q,"localId":"001","name":"Test Card","image":"https://example.com/001"}]}`, cardID)
		case "/id/cards/" + cardID:
			_, _ = fmt.Fprintf(w, `{"id":%q,"localId":"001","name":"Test Card","category":"Pokemon","rarity":"Common","image":"https://example.com/001","variants":{"normal":true}}`, cardID)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	in := NewIngester(testDB(t))
	in.client = &client{baseURL: server.URL, httpClient: server.Client()}

	summary, err := in.Run(context.Background(), "id", "sv")
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Cards)
	require.Len(t, summary.Mismatches, 1)
	assert.Contains(t, summary.Mismatches[0], setCode)
	assert.Contains(t, summary.Mismatches[0], "ingested 1 cards, TCGDex reports 2 official")
}
