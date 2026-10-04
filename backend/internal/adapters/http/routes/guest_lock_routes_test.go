package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

const (
	lockedCardCount = 30
	guestPageSize   = 24
	loginRequired   = "login_required"
)

type lockCatalog struct {
	set    entity.ExpansionSet
	rarity entity.Rarity
}

// seedLockCatalog creates one Expansion Set of lockedCardCount Cards, so a Guest
// has more behind the lock than its one page of 24. Every query below is scoped
// to this set, because the database is shared with other tests.
func seedLockCatalog(t *testing.T, api testAPI) lockCatalog {
	t.Helper()

	s := uuid.NewString()
	game := entity.Game{Slug: "lock-" + s, Name: "Lock " + s}
	require.NoError(t, api.db.Create(&game).Error)
	locale := entity.Locale{Code: "lock-" + s}
	require.NoError(t, api.db.Create(&locale).Error)
	rarity := entity.Rarity{GameID: game.ID, Code: "R-" + s, Name: "Rarity " + s}
	require.NoError(t, api.db.Create(&rarity).Error)
	set := entity.ExpansionSet{GameID: game.ID, Code: "lock-" + s, Name: "Lock " + s, LocaleID: locale.ID}
	require.NoError(t, api.db.Create(&set).Error)

	for i := 1; i <= lockedCardCount; i++ {
		card := entity.Card{ExpansionSetID: set.ID, LocalID: strconv.Itoa(1000 + i), Name: "Lock Card " + s, Category: "Pokémon",
			Tags: datatypes.JSONSlice[string]{"tag-" + s}, RarityID: rarity.ID, Attributes: datatypes.JSONMap{}}
		require.NoError(t, api.db.Create(&card).Error)
	}

	return lockCatalog{set: set, rarity: rarity}
}

type lockedPage struct {
	Data []dto.CardSummary  `json:"data"`
	Meta dto.PaginationMeta `json:"meta"`
}

func decodePage(t *testing.T, resp *httptest.ResponseRecorder) lockedPage {
	t.Helper()
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var page lockedPage
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &page))
	return page
}

// assertLoginRequired is the stable signal the UI keys on: a 401 whose body
// carries the login_required code, not a message to parse.
func assertLoginRequired(t *testing.T, resp *httptest.ResponseRecorder, msgAndArgs ...any) {
	t.Helper()
	require.Equal(t, http.StatusUnauthorized, resp.Code, append([]any{resp.Body.String()}, msgAndArgs...)...)
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, loginRequired, body.Code, msgAndArgs...)
}

func TestGuestCatalogLock(t *testing.T) {
	api := newTestAPI(t)
	cat := seedLockCatalog(t, api)
	inSet := "expansionSetId=" + cat.set.ID.String()

	t.Run("a Guest gets one page of 24 with the true total", func(t *testing.T) {
		page := decodePage(t, api.Get("/catalog/cards?"+inSet))
		assert.Equal(t, guestPageSize, len(page.Data))
		assert.Equal(t, dto.PaginationMeta{Total: lockedCardCount, Page: 1, Limit: guestPageSize}, page.Meta)
	})

	t.Run("a requested page size above 24 is reduced to 24", func(t *testing.T) {
		page := decodePage(t, api.Get("/catalog/cards?limit=100&"+inSet))
		assert.Equal(t, guestPageSize, len(page.Data))
		assert.Equal(t, guestPageSize, page.Meta.Limit)
		assert.Equal(t, lockedCardCount, page.Meta.Total)
	})

	t.Run("a smaller page size is kept", func(t *testing.T) {
		page := decodePage(t, api.Get("/catalog/cards?limit=5&"+inSet))
		assert.Equal(t, 5, len(page.Data))
		assert.Equal(t, 5, page.Meta.Limit)
	})

	t.Run("name search and the first page stay open", func(t *testing.T) {
		page := decodePage(t, api.Get("/catalog/cards?page=1&name=Lock+Card&"+inSet))
		assert.Equal(t, guestPageSize, len(page.Data))
	})

	t.Run("any page after the first is login_required", func(t *testing.T) {
		for _, page := range []string{"2", "3"} {
			assertLoginRequired(t, api.Get("/catalog/cards?page="+page+"&"+inSet), "page "+page)
		}
		assertLoginRequired(t, api.Get("/catalog/cards?page=2&limit=5&"+inSet), "page 2 of a small page size")
	})

	t.Run("a rarity, category or tag filter is login_required", func(t *testing.T) {
		filters := map[string]string{
			"rarity":   "rarityId=" + cat.rarity.ID.String(),
			"category": "category=Pok%C3%A9mon",
			"tag":      "tag=anything",
		}
		for name, filter := range filters {
			assertLoginRequired(t, api.Get("/catalog/cards?"+filter+"&"+inSet), name)
		}
	})

	t.Run("the facets endpoint is login_required", func(t *testing.T) {
		assertLoginRequired(t, api.Get("/catalog/facets"))
		assertLoginRequired(t, api.Get("/catalog/facets?"+inSet))
	})

	t.Run("the browse and filter-value lists stay open", func(t *testing.T) {
		for _, path := range []string{"/catalog/series", "/catalog/rarities", "/catalog/categories", "/catalog/tags"} {
			assert.Equal(t, http.StatusOK, api.Get(path).Code, path)
		}
	})
}

func TestSignedInCatalogIsUnlocked(t *testing.T) {
	api := newTestAPI(t)
	cat := seedLockCatalog(t, api)
	inSet := "expansionSetId=" + cat.set.ID.String()
	auth := bearer(api.newUserToken(t))

	t.Run("the page size ceiling is still 100", func(t *testing.T) {
		page := decodePage(t, api.Get("/catalog/cards?limit=100&"+inSet, auth))
		assert.Equal(t, lockedCardCount, len(page.Data))
		assert.Equal(t, dto.PaginationMeta{Total: lockedCardCount, Page: 1, Limit: 100}, page.Meta)

		resp := api.Get("/catalog/cards?limit=101&"+inSet, auth)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	})

	t.Run("later pages are served", func(t *testing.T) {
		page := decodePage(t, api.Get("/catalog/cards?page=2&"+inSet, auth))
		assert.Equal(t, lockedCardCount-guestPageSize, len(page.Data))
		assert.Equal(t, 2, page.Meta.Page)
	})

	t.Run("rarity, category and tag filters are served", func(t *testing.T) {
		for _, filter := range []string{"rarityId=" + cat.rarity.ID.String(), "category=Pok%C3%A9mon", "tag=nothing"} {
			assert.Equal(t, http.StatusOK, api.Get("/catalog/cards?"+filter+"&"+inSet, auth).Code, filter)
		}
	})

	t.Run("facets are served", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, api.Get("/catalog/facets?"+inSet, auth).Code)
	})
}

// TestInvalidTokenIsNotAGuestOnLockedCatalog: a present-but-bad token stays the
// plain 401 of ticket 07, never login_required, so the client refreshes its
// token instead of showing a sign-in prompt, and never the Guest preview.
func TestInvalidTokenIsNotAGuestOnLockedCatalog(t *testing.T) {
	api := newTestAPI(t)
	cat := seedLockCatalog(t, api)
	inSet := "expansionSetId=" + cat.set.ID.String()
	expired := api.sign(t, with(validClaims(), "exp", time.Now().Add(-time.Hour).Unix()))

	for _, path := range []string{"/catalog/cards?" + inSet, "/catalog/cards?page=2&" + inSet, "/catalog/facets"} {
		resp := api.Get(path, bearer(expired))
		require.Equal(t, http.StatusUnauthorized, resp.Code, path)
		var body struct {
			Code string `json:"code"`
		}
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		assert.NotEqual(t, loginRequired, body.Code, path)
	}
}
