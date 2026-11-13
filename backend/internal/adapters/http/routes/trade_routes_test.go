package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tradeSlugEnvelope struct {
	Data struct {
		Slug string `json:"slug"`
	} `json:"data"`
}

type tradeViewEnvelope struct {
	Data struct {
		Slug          string  `json:"slug"`
		Status        string  `json:"status"`
		OwnerName     string  `json:"ownerName"`
		OtherName     *string `json:"otherName"`
		IsOwner       bool    `json:"isOwner"`
		IsParticipant bool    `json:"isParticipant"`
		OwnerItems    []struct {
			ID   string `json:"id"`
			Card struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				ImageURL     string `json:"imageUrl"`
				ExpansionSet struct {
					Name string `json:"name"`
				} `json:"expansionSet"`
				Rarity struct {
					Name string `json:"name"`
				} `json:"rarity"`
			} `json:"card"`
			Quantity   int    `json:"quantity"`
			Collection string `json:"collection"`
		} `json:"ownerItems"`
		OtherItems []struct {
			ID   string `json:"id"`
			Card struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				ImageURL     string `json:"imageUrl"`
				ExpansionSet struct {
					Name string `json:"name"`
				} `json:"expansionSet"`
				Rarity struct {
					Name string `json:"name"`
				} `json:"rarity"`
			} `json:"card"`
			Quantity int `json:"quantity"`
		} `json:"otherItems"`
	} `json:"data"`
}

// createTrade posts POST /trades and returns the slug of the new Trade.
func createTrade(t *testing.T, api testAPI, token string) string {
	t.Helper()
	resp := api.Post("/trades", bearer(token), map[string]any{})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	var out tradeSlugEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
	require.NotEmpty(t, out.Data.Slug)
	return out.Data.Slug
}

func getTrade(api testAPI, token, slug string) *httptest.ResponseRecorder {
	return api.Get("/trades/"+slug, bearer(token))
}

func joinTrade(api testAPI, token, slug string) *httptest.ResponseRecorder {
	return api.Post("/trades/"+slug+"/join", bearer(token), map[string]any{})
}

func decodeTradeView(t *testing.T, resp *httptest.ResponseRecorder) tradeViewEnvelope {
	t.Helper()
	var out tradeViewEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
	return out
}

// TestTradeCreateReadJoin covers the happy path: create puts the caller on
// the owner seat in state open with a slug; reading by slug shows the state,
// the display name and empty item lists without claiming the seat; joining
// fills the second seat and moves the Trade to negotiating.
func TestTradeCreateReadJoin(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	joiner := api.newUserToken(t)

	slug := createTrade(t, api, owner)

	resp := getTrade(api, joiner, slug)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	view := decodeTradeView(t, resp)
	assert.Equal(t, slug, view.Data.Slug)
	assert.Equal(t, "open", view.Data.Status)
	assert.NotEmpty(t, view.Data.OwnerName)
	assert.Nil(t, view.Data.OtherName)
	assert.False(t, view.Data.IsParticipant) // open Trades read link-public, but the reader holds no seat yet
	assert.Empty(t, view.Data.OwnerItems)
	assert.Empty(t, view.Data.OtherItems)
	assert.NotContains(t, resp.Body.String(), "@") // no email leaks

	// Reading before joining left the seat free: a different user can still join.
	resp = joinTrade(api, joiner, slug)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	view = decodeTradeView(t, resp)
	assert.Equal(t, "negotiating", view.Data.Status)
	assert.NotNil(t, view.Data.OtherName)
	assert.True(t, view.Data.IsParticipant)
	assert.False(t, view.Data.IsOwner)

	// The owner now reads the same room with both names.
	resp = getTrade(api, owner, slug)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	view = decodeTradeView(t, resp)
	assert.Equal(t, "negotiating", view.Data.Status)
	assert.True(t, view.Data.IsOwner)
	assert.NotNil(t, view.Data.OtherName)
}

// TestTradeJoinSeatTaken: a second join gets the stable seat-taken code, and
// two joins issued at once leave exactly one winner.
func TestTradeJoinSeatTaken(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	first := api.newUserToken(t)
	second := api.newUserToken(t)

	slug := createTrade(t, api, owner)

	// The owner cannot take the Trade's own second seat.
	resp := joinTrade(api, owner, slug)
	require.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
	assert.Equal(t, "trade_owner_join", errorCode(t, resp))

	resp = joinTrade(api, first, slug)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

	resp = joinTrade(api, second, slug)
	require.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
	assert.Equal(t, "trade_seat_taken", errorCode(t, resp))

	// The loser's join did not take the seat: the winner still reads as participant.
	resp = getTrade(api, first, slug)
	require.Equal(t, http.StatusOK, resp.Code)
	assert.True(t, decodeTradeView(t, resp).Data.IsParticipant)
}

// TestTradeJoinRace: two joins at once — exactly one wins, the other gets
// trade_seat_taken.
func TestTradeJoinRace(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	a := api.newUserToken(t)
	b := api.newUserToken(t)

	slug := createTrade(t, api, owner)

	var wg sync.WaitGroup
	results := make([]*httptest.ResponseRecorder, 2)
	tokens := []string{a, b}
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = joinTrade(api, tokens[i], slug)
		}(i)
	}
	wg.Wait()

	var winners, losers int
	for _, resp := range results {
		switch resp.Code {
		case http.StatusOK:
			winners++
		case http.StatusConflict:
			losers++
			assert.Equal(t, "trade_seat_taken", errorCode(t, resp))
		default:
			t.Fatalf("unexpected status %d: %s", resp.Code, resp.Body.String())
		}
	}
	assert.Equal(t, 1, winners)
	assert.Equal(t, 1, losers)
}

// TestTradeReadRefusals: a Guest gets 401 on every Trade route; an unknown
// slug, and a signed-in non-participant of a joined Trade, get the same 404.
func TestTradeReadRefusals(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)
	joiner := api.newUserToken(t)
	outsider := api.newUserToken(t)

	slug := createTrade(t, api, owner)

	// Guest: create and join need the Auth Identity (401); the read answers
	// the same not-found as an unknown slug (user story 43).
	assert.Equal(t, http.StatusUnauthorized, api.Post("/trades", "", map[string]any{}).Code)
	assert.Equal(t, http.StatusUnauthorized, api.Post("/trades/"+slug+"/join", "", map[string]any{}).Code)
	guestRead := api.Get("/trades/" + slug)
	assert.Equal(t, http.StatusNotFound, guestRead.Code)
	assert.Equal(t, getTrade(api, outsider, "no-such-trade").Body.String(), guestRead.Body.String())

	// Unknown slug: same 404 for a signed-in user.
	assert.Equal(t, http.StatusNotFound, getTrade(api, outsider, "no-such-trade").Code)

	// While open, any signed-in user reads; after the join the room closes.
	require.Equal(t, http.StatusOK, joinTrade(api, joiner, slug).Code)
	resp := getTrade(api, outsider, slug)
	require.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	assert.Equal(t, getTrade(api, outsider, "no-such-trade").Body.String(), resp.Body.String())
}

// TestTradeReadCancelled: a cancelled Trade answers the same not-found as an
// unknown slug — to its owner and on join too. No cancel endpoint exists yet
// in this issue, so the fixture flips the status in the database directly.
func TestTradeReadCancelled(t *testing.T) {
	api := newTestAPI(t)
	owner := api.newUserToken(t)

	slug := createTrade(t, api, owner)
	require.NoError(t, api.db.Model(&entity.Trade{}).Where("slug = ?", slug).
		Update("status", entity.TradeStatusCancelled).Error)

	resp := getTrade(api, owner, slug)
	require.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	assert.Equal(t, getTrade(api, owner, "no-such-trade").Body.String(), resp.Body.String())

	assert.Equal(t, http.StatusNotFound, joinTrade(api, owner, slug).Code)
}

func TestTradeItemsAddReadRemoveAndLock(t *testing.T) {
	api := newTestAPI(t)
	cards := newTestCards(t, 1)
	owner := api.newUserToken(t)
	joiner := api.newUserToken(t)
	outsider := api.newUserToken(t)
	slug := createTrade(t, api, owner)
	require.Equal(t, http.StatusOK, joinTrade(api, joiner, slug).Code)

	createCollection := func(token, title string) string {
		t.Helper()
		resp := api.Post("/collections", bearer(token), map[string]any{"title": title})
		require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
		var created struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &created))
		return created.Data.ID
	}
	first := createCollection(owner, "First source")
	second := createCollection(owner, "Second source")
	foreign := createCollection(outsider, "Foreign source")
	firstEntries := "/collections/" + first + "/entries"
	resp := api.Post(firstEntries, bearer(owner), map[string]any{"cardId": cards[0], "quantity": 1})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

	add := func(collectionID string, quantity int) *httptest.ResponseRecorder {
		t.Helper()
		return api.Post("/trades/"+slug+"/items", bearer(owner), map[string]any{
			"cardId": cards[0], "sourceCollectionId": collectionID, "quantity": quantity,
		})
	}
	assert.Equal(t, http.StatusBadRequest, add(first, 0).Code)
	assert.Equal(t, http.StatusBadRequest, add(first, -1).Code)
	assert.Equal(t, http.StatusNotFound, add(foreign, 1).Code)

	// Offering more than the source currently holds is allowed before execution.
	resp = add(first, 5)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	view := decodeTradeView(t, resp)
	require.Len(t, view.Data.OwnerItems, 1)
	assert.Equal(t, cards[0].String(), view.Data.OwnerItems[0].Card.ID)
	assert.NotEmpty(t, view.Data.OwnerItems[0].Card.Name)
	assert.NotEmpty(t, view.Data.OwnerItems[0].Card.ExpansionSet.Name)
	assert.NotEmpty(t, view.Data.OwnerItems[0].Card.Rarity.Name)
	assert.Equal(t, "First source", view.Data.OwnerItems[0].Collection)
	itemID := view.Data.OwnerItems[0].ID

	assert.Equal(t, http.StatusConflict, add(first, 2).Code)
	resp = add(second, 2)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	view = decodeTradeView(t, resp)
	require.Len(t, view.Data.OwnerItems, 2)
	joinerSource := createCollection(joiner, "Joiner source")
	resp = api.Post("/trades/"+slug+"/items", bearer(joiner), map[string]any{
		"cardId": cards[0], "sourceCollectionId": joinerSource, "quantity": 1,
	})
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	view = decodeTradeView(t, resp)
	require.Len(t, view.Data.OtherItems, 1)
	assert.Equal(t, cards[0].String(), view.Data.OtherItems[0].Card.ID)
	assert.NotEmpty(t, view.Data.OtherItems[0].Card.ExpansionSet.Name)
	assert.NotEmpty(t, view.Data.OtherItems[0].Card.Rarity.Name)

	resp = api.Delete("/trades/"+slug+"/items/"+itemID, bearer(joiner))
	assert.Equal(t, http.StatusNotFound, resp.Code, resp.Body.String())
	resp = api.Delete("/trades/"+slug+"/items/"+itemID, bearer(owner))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.Len(t, decodeTradeView(t, resp).Data.OwnerItems, 1)

	require.NoError(t, api.db.Model(&entity.Trade{}).Where("slug = ?", slug).
		Update("status", entity.TradeStatusLocked).Error)
	assert.Equal(t, http.StatusConflict, add(second, 1).Code)
	resp = api.Delete("/trades/"+slug+"/items/"+decodeTradeView(t, getTrade(api, owner, slug)).Data.OwnerItems[0].ID, bearer(owner))
	assert.Equal(t, http.StatusConflict, resp.Code, resp.Body.String())
}
