package routes

import (
	"encoding/json"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

const reprintsPath = "/scan/reprints"

type reprintEnvelope struct {
	Data dto.ReprintResult `json:"data"`
}

// planar is the unit vector at deg degrees in the first plane, so two Cards
// at angles a and b sit 1-cos(a-b) apart by cosine distance. at(0) is
// axisVector(0).
func planar(deg float64) []float32 {
	v := make([]float32, embedding.Dimensions)
	v[0] = float32(math.Cos(deg * math.Pi / 180))
	v[1] = float32(math.Sin(deg * math.Pi / 180))
	return v
}

// reprintSet is an Expansion Set and the Rarity its Cards use.
type reprintSet struct {
	set    entity.ExpansionSet
	rarity uuid.UUID
}

// newReprintSet creates an Expansion Set released on releaseDate (nil = unknown).
func (a testAPI) newReprintSet(t *testing.T, releaseDate *time.Time, published bool) reprintSet {
	t.Helper()
	s := uuid.NewString()
	game := entity.Game{Slug: "rp-" + s, Name: "Reprint " + s}
	require.NoError(t, a.db.Create(&game).Error)
	locale := entity.Locale{Code: "rp-" + s}
	require.NoError(t, a.db.Create(&locale).Error)
	rarity := entity.Rarity{GameID: game.ID, Code: "R-" + s, Name: "Rarity " + s}
	require.NoError(t, a.db.Create(&rarity).Error)
	set := entity.ExpansionSet{CreatedBy: entity.SourcePokemonAsia, UpdatedBy: entity.SourcePokemonAsia, GameID: game.ID, Code: "rp-" + s, Name: "Set " + s, LocaleID: locale.ID, ReleaseDate: releaseDate}
	require.NoError(t, a.db.Create(&set).Error)
	if !published {
		// Create would write the column default (true) for a false Published, so the hide is an update.
		require.NoError(t, a.db.Model(&set).Update("published", false).Error)
	}
	return reprintSet{set: set, rarity: rarity.ID}
}

// newNamedCard creates a Card called name in s and returns its id.
func (a testAPI) newNamedCard(t *testing.T, s reprintSet, name string) uuid.UUID {
	t.Helper()
	card := entity.Card{CreatedBy: entity.SourcePokemonAsia, UpdatedBy: entity.SourcePokemonAsia, ExpansionSetID: s.set.ID, LocalID: uuid.NewString(), Name: name, Category: "Pokémon",
		Tags: datatypes.JSONSlice[string]{}, RarityID: s.rarity, Attributes: datatypes.JSONMap{}}
	require.NoError(t, a.db.Create(&card).Error)
	return card.ID
}

// reprints gets the Reprints route with query and returns the response.
func (a testAPI) reprints(t *testing.T, token, query string) (int, dto.ReprintResult) {
	t.Helper()
	resp := a.Get(reprintsPath+"?"+query, bearer(token))
	var body reprintEnvelope
	if resp.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	}
	return resp.Code, body.Data
}

// reprintQuery is the query string for the ids, in order, at offset.
func reprintQuery(offset int, ids ...uuid.UUID) string {
	q := url.Values{"offset": {strconv.Itoa(offset)}}
	for _, id := range ids {
		q.Add("cardId", id.String())
	}
	return q.Encode()
}

func summaryIDs(cards []dto.CardSummary) []uuid.UUID {
	ids := make([]uuid.UUID, len(cards))
	for i, c := range cards {
		ids[i] = c.ID
	}
	return ids
}

// storeEmbeddingFrom stores a row of model for card under source, which is
// "catalog" for the catalog image.
func (a testAPI) storeEmbeddingFrom(t *testing.T, card uuid.UUID, source string, vector []float32) {
	t.Helper()
	row := entity.CardEmbedding{CardID: card, Model: a.model, Source: source, Embedding: pgvector.NewVector(vector)}
	require.NoError(t, a.db.Create(&row).Error)
}

// TestScanReprints covers the Reprint rule of ADR-0023 at the HTTP boundary,
// against real Postgres with embeddings seeded directly.
func TestScanReprints(t *testing.T) {
	t.Run("requires a valid token", func(t *testing.T) {
		api := newTestAPI(t)

		assert.Equal(t, http.StatusUnauthorized, api.Get(reprintsPath+"?"+reprintQuery(0)).Code)
		assert.Equal(t, http.StatusUnauthorized, api.Get(reprintsPath+"?"+reprintQuery(0), bearer("not-a-token")).Code)
	})

	t.Run("a same-name Card with the same art in another set is a Reprint, different art or a different name is not", func(t *testing.T) {
		api := newTestAPI(t)
		name := "Rp " + uuid.NewString()
		source := api.newReprintSet(t, nil, true)
		other := api.newReprintSet(t, nil, true)
		src := api.newNamedCard(t, source, name)
		api.storeEmbeddingFrom(t, src, "catalog", planar(0))
		same := api.newNamedCard(t, other, name)
		api.storeEmbeddingFrom(t, same, "catalog", planar(1))
		diffArt := api.newNamedCard(t, other, name)
		api.storeEmbeddingFrom(t, diffArt, "catalog", planar(90))
		diffName := api.newNamedCard(t, other, name+" ex")
		api.storeEmbeddingFrom(t, diffName, "catalog", planar(1))

		code, got := api.reprints(t, api.newUserToken(t), reprintQuery(0, src))

		require.Equal(t, http.StatusOK, code)
		assert.Equal(t, []uuid.UUID{same}, summaryIDs(got.Cards))
		assert.False(t, got.HasMore)
	})

	t.Run("name matching folds case and whitespace and is otherwise exact", func(t *testing.T) {
		api := newTestAPI(t)
		name := "Rp " + uuid.NewString() + " ex"
		source := api.newReprintSet(t, nil, true)
		other := api.newReprintSet(t, nil, true)
		src := api.newNamedCard(t, source, name)
		api.storeEmbeddingFrom(t, src, "catalog", planar(0))
		folded := api.newNamedCard(t, other, "  "+upperAndSpaced(name))
		api.storeEmbeddingFrom(t, folded, "catalog", planar(1))
		withoutSuffix := api.newNamedCard(t, other, name[:len(name)-len(" ex")])
		api.storeEmbeddingFrom(t, withoutSuffix, "catalog", planar(1))

		code, got := api.reprints(t, api.newUserToken(t), reprintQuery(0, src))

		require.Equal(t, http.StatusOK, code)
		assert.Equal(t, []uuid.UUID{folded}, summaryIDs(got.Cards))
	})

	t.Run("every id sent is excluded, including one that is another id's source", func(t *testing.T) {
		api := newTestAPI(t)
		name := "Rp " + uuid.NewString()
		source := api.newReprintSet(t, nil, true)
		other := api.newReprintSet(t, nil, true)
		src := api.newNamedCard(t, source, name)
		api.storeEmbeddingFrom(t, src, "catalog", planar(0))
		sent := api.newNamedCard(t, other, name)
		api.storeEmbeddingFrom(t, sent, "catalog", planar(1))

		code, got := api.reprints(t, api.newUserToken(t), reprintQuery(0, src, sent))

		require.Equal(t, http.StatusOK, code)
		assert.Empty(t, got.Cards)
		assert.False(t, got.HasMore)
	})

	t.Run("results follow source position, then closeness, then newest set, and a Card reachable twice appears once", func(t *testing.T) {
		api := newTestAPI(t)
		name := "Rp " + uuid.NewString()
		older := api.newReprintSet(t, ptrTime(2010), true)
		mid := api.newReprintSet(t, ptrTime(2018), true)
		newer := api.newReprintSet(t, ptrTime(2025), true)
		// A and B are both sources of name; A sits at 0 degrees, B at 26.
		a := api.newNamedCard(t, older, name)
		api.storeEmbeddingFrom(t, a, "catalog", planar(0))
		b := api.newNamedCard(t, older, name)
		api.storeEmbeddingFrom(t, b, "catalog", planar(26))

		aTie := api.newNamedCard(t, newer, name)
		api.storeEmbeddingFrom(t, aTie, "catalog", planar(2))
		aClose := api.newNamedCard(t, older, name)
		api.storeEmbeddingFrom(t, aClose, "catalog", planar(2))
		aFar := api.newNamedCard(t, newer, name)
		api.storeEmbeddingFrom(t, aFar, "catalog", planar(-4))
		both := api.newNamedCard(t, mid, name)
		api.storeEmbeddingFrom(t, both, "catalog", planar(13))
		bOnly := api.newNamedCard(t, mid, name)
		api.storeEmbeddingFrom(t, bOnly, "catalog", planar(36))

		code, got := api.reprints(t, api.newUserToken(t), reprintQuery(0, a, b))

		require.Equal(t, http.StatusOK, code)
		assert.Equal(t, []uuid.UUID{aTie, aClose, aFar, both, bOnly}, summaryIDs(got.Cards))
		assert.False(t, got.HasMore)
	})

	t.Run("pages are five Cards, hasMore holds while more remain, and the same call gives the same page", func(t *testing.T) {
		api := newTestAPI(t)
		name := "Rp " + uuid.NewString()
		source := api.newReprintSet(t, nil, true)
		other := api.newReprintSet(t, nil, true)
		src := api.newNamedCard(t, source, name)
		api.storeEmbeddingFrom(t, src, "catalog", planar(0))
		for i := range 6 {
			card := api.newNamedCard(t, other, name)
			api.storeEmbeddingFrom(t, card, "catalog", planar(float64(i+1)*0.1))
		}

		code, first := api.reprints(t, api.newUserToken(t), reprintQuery(0, src))
		require.Equal(t, http.StatusOK, code)
		assert.Len(t, first.Cards, 5)
		assert.True(t, first.HasMore)

		_, again := api.reprints(t, api.newUserToken(t), reprintQuery(0, src))
		assert.Equal(t, summaryIDs(first.Cards), summaryIDs(again.Cards))

		_, second := api.reprints(t, api.newUserToken(t), reprintQuery(5, src))
		require.Len(t, second.Cards, 1)
		assert.False(t, second.HasMore)
		assert.NotContains(t, summaryIDs(first.Cards), second.Cards[0].ID)

		_, beyond := api.reprints(t, api.newUserToken(t), reprintQuery(10, src))
		assert.Empty(t, beyond.Cards)
		assert.False(t, beyond.HasMore)
	})

	t.Run("a Card without an embedding or in an unpublished set is never a Reprint", func(t *testing.T) {
		api := newTestAPI(t)
		name := "Rp " + uuid.NewString()
		source := api.newReprintSet(t, nil, true)
		hidden := api.newReprintSet(t, nil, false)
		src := api.newNamedCard(t, source, name)
		api.storeEmbeddingFrom(t, src, "catalog", planar(0))
		unembedded := api.newNamedCard(t, source, name)
		hiddenCard := api.newNamedCard(t, hidden, name)
		api.storeEmbeddingFrom(t, hiddenCard, "catalog", planar(1))

		code, got := api.reprints(t, api.newUserToken(t), reprintQuery(0, src))

		require.Equal(t, http.StatusOK, code)
		assert.NotContains(t, summaryIDs(got.Cards), unembedded)
		assert.NotContains(t, summaryIDs(got.Cards), hiddenCard)
		assert.Empty(t, got.Cards)
	})

	t.Run("a source without an embedding offers nothing", func(t *testing.T) {
		api := newTestAPI(t)
		name := "Rp " + uuid.NewString()
		source := api.newReprintSet(t, nil, true)
		other := api.newReprintSet(t, nil, true)
		src := api.newNamedCard(t, source, name)
		candidate := api.newNamedCard(t, other, name)
		api.storeEmbeddingFrom(t, candidate, "catalog", planar(1))

		code, got := api.reprints(t, api.newUserToken(t), reprintQuery(0, src))

		require.Equal(t, http.StatusOK, code)
		assert.Empty(t, got.Cards)
	})

	t.Run("a Card with several embeddings is judged by its nearest one", func(t *testing.T) {
		api := newTestAPI(t)
		name := "Rp " + uuid.NewString()
		source := api.newReprintSet(t, nil, true)
		other := api.newReprintSet(t, nil, true)
		src := api.newNamedCard(t, source, name)
		api.storeEmbeddingFrom(t, src, "catalog", planar(0))
		candidate := api.newNamedCard(t, other, name)
		api.storeEmbeddingFrom(t, candidate, "catalog", planar(20))
		api.storeEmbeddingFrom(t, candidate, "photo", planar(1))

		code, got := api.reprints(t, api.newUserToken(t), reprintQuery(0, src))

		require.Equal(t, http.StatusOK, code)
		assert.Equal(t, []uuid.UUID{candidate}, summaryIDs(got.Cards))
	})

	t.Run("invalid input is refused and unknown ids are dropped", func(t *testing.T) {
		api := newTestAPI(t)
		token := api.newUserToken(t)
		ids := make([]uuid.UUID, 6)
		for i := range ids {
			ids[i] = uuid.New()
		}

		assert.Equal(t, http.StatusUnprocessableEntity, api.getCode(reprintsPath+"?"+reprintQuery(0, ids...), bearer(token)), "more than five ids")
		assert.Equal(t, http.StatusUnprocessableEntity, api.getCode(reprintsPath+"?"+reprintQuery(-1, ids[0]), bearer(token)), "negative offset")
		assert.Equal(t, http.StatusBadRequest, api.getCode(reprintsPath+"?cardId=not-a-uuid", bearer(token)), "malformed id")

		code, got := api.reprints(t, token, reprintQuery(0, uuid.New()))
		require.Equal(t, http.StatusOK, code)
		assert.Empty(t, got.Cards)
		assert.False(t, got.HasMore)
	})
}

// getCode is the status code of a GET.
func (a testAPI) getCode(url string, headers ...any) int {
	return a.Get(url, headers...).Code
}

// upperAndSpaced is s with its letters upper-cased and its spaces doubled, so
// a name that folds to s differs from s in case and whitespace.
func upperAndSpaced(s string) string {
	out := make([]byte, 0, len(s)*2)
	for i := range len(s) {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		out = append(out, c)
		if c == ' ' {
			out = append(out, ' ')
		}
	}
	return string(out)
}

func ptrTime(year int) *time.Time {
	t := time.Date(year, time.June, 1, 0, 0, 0, 0, time.UTC)
	return &t
}
