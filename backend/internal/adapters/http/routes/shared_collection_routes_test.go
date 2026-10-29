package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

// sharedSlugShape is a Slug (ADR-0025): the stem, a hyphen, then 5 hex characters.
var sharedSlugShape = regexp.MustCompile(`^base-set-binder-[0-9a-f]{5}$`)

type sharedCollectionEnvelope struct {
	Data struct {
		ID   string  `json:"id"`
		Slug *string `json:"slug"`
	} `json:"data"`
}

type sharedHeaderEnvelope struct {
	Data struct {
		OwnerName   string `json:"ownerName"`
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"data"`
}

type sharedEntriesEnvelope struct {
	Data []struct {
		Card struct {
			ID string `json:"id"`
		} `json:"card"`
		Quantity int `json:"quantity"`
	} `json:"data"`
	Meta struct{ Total, Page, Limit int } `json:"meta"`
}

type sharedFacetsEnvelope struct {
	Data dto.CatalogFacets `json:"data"`
}

// sharedOwner is a signed-in user who owns Collections, with the email it signs in with.
type sharedOwner struct {
	token string
	email string
}

func newSharedOwner(t *testing.T, api testAPI) sharedOwner {
	t.Helper()
	email := uuid.NewString() + "@example.com"
	return sharedOwner{token: api.userTokenFor(t, email), email: email}
}

// profileSlug is the Slug of the profile the user with email owns. The profile
// exists once that user has made any request.
func (a testAPI) profileSlug(t *testing.T, email string) string {
	t.Helper()
	var profile entity.UserProfile
	require.NoError(t, a.db.Joins("JOIN users ON users.id = user_profiles.user_id").Where("users.email = ?", email).Take(&profile).Error)
	return profile.Slug
}

func sharedPath(profileSlug, collectionSlug string) string {
	return "/shared/" + profileSlug + "/" + collectionSlug
}

func createCollection(t *testing.T, api testAPI, owner sharedOwner, title string) string {
	t.Helper()
	resp := api.Post("/collections", bearer(owner.token), map[string]any{"title": title})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
	var created collectionEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &created))
	return created.Data.ID
}

func shareCollection(t *testing.T, api testAPI, owner sharedOwner, id string) sharedCollectionEnvelope {
	t.Helper()
	resp := api.Post("/collections/"+id+"/share", bearer(owner.token))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var out sharedCollectionEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
	require.NotNil(t, out.Data.Slug)
	return out
}

func unshareCollection(t *testing.T, api testAPI, owner sharedOwner, id string) {
	t.Helper()
	resp := api.Delete("/collections/"+id+"/share", bearer(owner.token))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
}

func addSharedEntry(t *testing.T, api testAPI, owner sharedOwner, id string, cardID uuid.UUID, quantity int) {
	t.Helper()
	resp := api.Post("/collections/"+id+"/entries", bearer(owner.token), map[string]any{"cardId": cardID, "quantity": quantity})
	require.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())
}

// sharedSet is a seeded Expansion Set: its Cards and the Rarity they share.
type sharedSet struct {
	cards  []uuid.UUID
	rarity uuid.UUID
}

// seedSharedSet inserts an Expansion Set of n Cards with one category and one
// tag. The published column defaults to true, so an unpublished set is updated
// after the insert.
func seedSharedSet(t *testing.T, api testAPI, published bool, category, tag string, n int) sharedSet {
	t.Helper()
	s := uuid.NewString()
	game := entity.Game{Slug: "shr-" + s, Name: "Shared " + s}
	require.NoError(t, api.db.Create(&game).Error)
	locale := entity.Locale{Code: "shr-" + s}
	require.NoError(t, api.db.Create(&locale).Error)
	rarity := entity.Rarity{GameID: game.ID, Code: "R-" + s, Name: "Rarity " + s}
	require.NoError(t, api.db.Create(&rarity).Error)
	set := entity.ExpansionSet{CreatedBy: entity.SourcePokemonAsia, UpdatedBy: entity.SourcePokemonAsia, GameID: game.ID, Code: "shr-" + s, Name: "Set " + s, LocaleID: locale.ID}
	require.NoError(t, api.db.Create(&set).Error)
	if !published {
		require.NoError(t, api.db.Model(&set).Update("published", false).Error)
	}

	out := sharedSet{cards: make([]uuid.UUID, n), rarity: rarity.ID}
	for i := range out.cards {
		card := entity.Card{CreatedBy: entity.SourcePokemonAsia, UpdatedBy: entity.SourcePokemonAsia, ExpansionSetID: set.ID, LocalID: strconv.Itoa(i + 1), Name: "Shared Card " + s, Category: category,
			Tags: datatypes.JSONSlice[string]{tag}, RarityID: rarity.ID, Attributes: datatypes.JSONMap{}}
		require.NoError(t, api.db.Create(&card).Error)
		out.cards[i] = card.ID
	}
	return out
}

// slugTail is the last 5 characters of an id, the random part of a Slug.
func slugTail(id string) string {
	return id[len(id)-5:]
}

// TestSharePutsASlugOnTheTitleAndKeepsItThroughARename covers turning sharing
// on, sharing again while on, and a rename that must not move the link.
func TestSharePutsASlugOnTheTitleAndKeepsItThroughARename(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)
	id := createCollection(t, api, owner, "Base Set Binder")

	slug := *shareCollection(t, api, owner, id).Data.Slug
	assert.Regexp(t, sharedSlugShape, slug)

	again := *shareCollection(t, api, owner, id).Data.Slug
	assert.Equal(t, slug, again, "sharing while already Shared keeps the link")

	resp := api.Put("/collections/"+id, bearer(owner.token), map[string]any{"title": "Renamed Binder"})
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var renamed sharedCollectionEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &renamed))
	require.NotNil(t, renamed.Data.Slug)
	assert.Equal(t, slug, *renamed.Data.Slug, "a rename never changes a Shared Collection's Slug")

	resp = api.Get(sharedPath(api.profileSlug(t, owner.email), slug))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var header sharedHeaderEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &header))
	assert.Equal(t, "Renamed Binder", header.Data.Title)
}

// TestUnshareEndsTheLinkAndResharingNeverBringsItBack covers turning sharing
// off, which clears the Slug, and turning it on again, which makes a new one.
func TestUnshareEndsTheLinkAndResharingNeverBringsItBack(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)
	id := createCollection(t, api, owner, "Base Set Binder")

	first := *shareCollection(t, api, owner, id).Data.Slug
	profile := api.profileSlug(t, owner.email)
	require.Equal(t, http.StatusOK, api.Get(sharedPath(profile, first)).Code)

	unshareCollection(t, api, owner, id)
	resp := api.Get("/collections/"+id, bearer(owner.token))
	var private sharedCollectionEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &private))
	assert.Nil(t, private.Data.Slug, "a private Collection has no Slug")
	assert.Equal(t, http.StatusNotFound, api.Get(sharedPath(profile, first)).Code, "the old link stops working at once")

	// Unsharing a Collection that is already private changes nothing and still succeeds.
	again := api.Delete("/collections/"+id+"/share", bearer(owner.token))
	require.Equal(t, http.StatusOK, again.Code, again.Body.String())
	var stillPrivate sharedCollectionEnvelope
	require.NoError(t, json.Unmarshal(again.Body.Bytes(), &stillPrivate))
	assert.Nil(t, stillPrivate.Data.Slug)

	second := *shareCollection(t, api, owner, id).Data.Slug
	assert.NotEqual(t, first, second, "sharing again makes a fresh Slug")
	assert.Equal(t, http.StatusNotFound, api.Get(sharedPath(profile, first)).Code, "the old link never comes back")
	assert.Equal(t, http.StatusOK, api.Get(sharedPath(profile, second)).Code)
}

// TestSharedViewIsTheSameForAGuestASignedInUserAndTheOwner checks the header and
// the entries are identical for each viewer and that the owner's email never
// appears in the response.
func TestSharedViewIsTheSameForAGuestASignedInUserAndTheOwner(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)
	other := newSharedOwner(t, api)
	set := seedSharedSet(t, api, true, "Pokémon", "tag-"+uuid.NewString(), 1)
	id := createCollection(t, api, owner, "Shared Binder")
	addSharedEntry(t, api, owner, id, set.cards[0], 2)
	base := sharedPath(api.profileSlug(t, owner.email), *shareCollection(t, api, owner, id).Data.Slug)

	viewers := map[string]string{"guest": "", "signed-in": bearer(other.token), "owner": bearer(owner.token)}
	headers := map[string]*httptest.ResponseRecorder{}
	entries := map[string]*httptest.ResponseRecorder{}
	for name, auth := range viewers {
		var args []any
		if auth != "" {
			args = append(args, auth)
		}
		headers[name] = api.Get(base, args...)
		entries[name] = api.Get(base+"/entries", args...)
		require.Equal(t, http.StatusOK, headers[name].Code, name+": "+headers[name].Body.String())
		require.Equal(t, http.StatusOK, entries[name].Code, name+": "+entries[name].Body.String())
		assert.NotContains(t, headers[name].Body.String(), owner.email, name+" must never see the owner's email")
		assert.NotContains(t, entries[name].Body.String(), owner.email, name+" must never see the owner's email")
		facets := api.Get(base+"/facets", args...)
		require.Equal(t, http.StatusOK, facets.Code, name+": "+facets.Body.String())
		assert.NotContains(t, facets.Body.String(), owner.email, name+" must never see the owner's email")
	}
	assert.JSONEq(t, headers["guest"].Body.String(), headers["signed-in"].Body.String())
	assert.JSONEq(t, headers["guest"].Body.String(), headers["owner"].Body.String())
	assert.JSONEq(t, entries["guest"].Body.String(), entries["signed-in"].Body.String())
	assert.JSONEq(t, entries["guest"].Body.String(), entries["owner"].Body.String())

	var header sharedHeaderEnvelope
	require.NoError(t, json.Unmarshal(headers["guest"].Body.Bytes(), &header))
	assert.Equal(t, "Test User", header.Data.OwnerName, "the owner's display name, not their email")
	assert.Equal(t, "Shared Binder", header.Data.Title)

	var listed sharedEntriesEnvelope
	require.NoError(t, json.Unmarshal(entries["guest"].Body.Bytes(), &listed))
	require.Len(t, listed.Data, 1)
	assert.Equal(t, set.cards[0].String(), listed.Data[0].Card.ID)
	assert.Equal(t, 2, listed.Data[0].Quantity)
}

// TestEveryMissingSharedTargetGetsTheSameNotFound: a private, unshared,
// nonexistent or another profile's target all answer the same way on each
// shared endpoint.
func TestEveryMissingSharedTargetGetsTheSameNotFound(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)
	other := newSharedOwner(t, api)
	// A real row that was never shared: its slug is NULL, so the probe below
	// names a Slug nothing holds.
	privateID := createCollection(t, api, owner, "Base Set Binder")
	gone := createCollection(t, api, owner, "Once Shared")
	goneSlug := *shareCollection(t, api, owner, gone).Data.Slug
	unshareCollection(t, api, owner, gone)
	createCollection(t, api, other, "Other Binder")
	ownerProfile := api.profileSlug(t, owner.email)
	otherProfile := api.profileSlug(t, other.email)

	targets := map[string]string{
		"private":             sharedPath(ownerProfile, "base-set-binder-"+slugTail(privateID)),
		"unshared":            sharedPath(ownerProfile, goneSlug),
		"nonexistent-binder":  sharedPath(ownerProfile, "no-such-binder-00000"),
		"nonexistent-profile": sharedPath("no-such-user-00000", goneSlug),
		"other-profile":       sharedPath(otherProfile, goneSlug),
	}
	for _, suffix := range []string{"", "/entries", "/facets", "/pages/1"} {
		var reference, referenceName string
		for name, base := range targets {
			resp := api.Get(base + suffix)
			require.Equal(t, http.StatusNotFound, resp.Code, name+suffix+": "+resp.Body.String())
			shape := sharedMissShape(resp)
			if reference == "" {
				reference, referenceName = shape, name
				continue
			}
			assert.Equal(t, reference, shape, name+suffix+" must look exactly like "+referenceName)
		}
	}
}

// sharedMissShape is every part of a response a caller can see, except the
// per-request headers (Date and the request id), which differ on each call.
func sharedMissShape(resp *httptest.ResponseRecorder) string {
	headers := resp.Result().Header.Clone()
	headers.Del("Date")
	headers.Del("X-Request-Id")
	return fmt.Sprintf("%d\n%v\n%s", resp.Code, headers, resp.Body.String())
}

// TestSharedEntriesUseTheOwnersFiltersAndFacetsForAGuest: a Guest is not held to
// the catalog's preview lock on a Shared Collection, so later pages and the
// rarity, category and tag filters all answer.
func TestSharedEntriesUseTheOwnersFiltersAndFacetsForAGuest(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)
	tag := "tag-" + uuid.NewString()
	pokemon := seedSharedSet(t, api, true, "Pokémon", tag, 1)
	trainer := seedSharedSet(t, api, true, "Trainer", tag, 1)
	id := createCollection(t, api, owner, "Mixed Binder")
	addSharedEntry(t, api, owner, id, pokemon.cards[0], 1)
	addSharedEntry(t, api, owner, id, trainer.cards[0], 3)
	base := sharedPath(api.profileSlug(t, owner.email), *shareCollection(t, api, owner, id).Data.Slug)

	list := func(query url.Values) sharedEntriesEnvelope {
		t.Helper()
		resp := api.Get(base + "/entries?" + query.Encode())
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		var out sharedEntriesEnvelope
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
		return out
	}

	paged := list(url.Values{"limit": {"1"}, "page": {"2"}})
	assert.Equal(t, 2, paged.Meta.Total)
	assert.Len(t, paged.Data, 1)

	byCategory := list(url.Values{"category": {"Pokémon"}})
	require.Len(t, byCategory.Data, 1)
	assert.Equal(t, pokemon.cards[0].String(), byCategory.Data[0].Card.ID)

	byRarity := list(url.Values{"rarityId": {pokemon.rarity.String()}})
	require.Len(t, byRarity.Data, 1)
	assert.Equal(t, pokemon.cards[0].String(), byRarity.Data[0].Card.ID)

	byTag := list(url.Values{"tag": {tag}})
	assert.Equal(t, 2, byTag.Meta.Total)

	resp := api.Get(base + "/facets")
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var facets sharedFacetsEnvelope
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &facets))
	var categories []string
	for _, c := range facets.Data.Categories {
		categories = append(categories, c.Value)
	}
	assert.ElementsMatch(t, []string{"Pokémon", "Trainer"}, categories)
}

// TestSharedEntriesLeaveOutUnpublishedSets: a Card of an unpublished Expansion
// Set is absent from the entries and the facets for every viewer.
func TestSharedEntriesLeaveOutUnpublishedSets(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)
	tag := "tag-" + uuid.NewString()
	live := seedSharedSet(t, api, true, "Pokémon", tag, 1)
	hidden := seedSharedSet(t, api, false, "Trainer", tag, 1)
	id := createCollection(t, api, owner, "Mixed Binder")
	addSharedEntry(t, api, owner, id, live.cards[0], 1)
	// The API refuses a Card of an unpublished set, so the entry is written
	// directly: it is the case where a set was unpublished after it was added.
	require.NoError(t, api.db.Create(&entity.InventoryEntry{CollectionID: uuid.MustParse(id), CardID: hidden.cards[0], Quantity: 1}).Error)
	base := sharedPath(api.profileSlug(t, owner.email), *shareCollection(t, api, owner, id).Data.Slug)

	for name, auth := range map[string]string{"guest": "", "owner": bearer(owner.token)} {
		var args []any
		if auth != "" {
			args = append(args, auth)
		}
		entries := api.Get(base+"/entries?tag="+url.QueryEscape(tag), args...)
		require.Equal(t, http.StatusOK, entries.Code, name+": "+entries.Body.String())
		var listed sharedEntriesEnvelope
		require.NoError(t, json.Unmarshal(entries.Body.Bytes(), &listed))
		assert.Equal(t, 1, listed.Meta.Total, name)
		assert.NotContains(t, entries.Body.String(), hidden.cards[0].String(), name)

		facets := api.Get(base+"/facets", args...)
		require.Equal(t, http.StatusOK, facets.Code, name+": "+facets.Body.String())
		assert.NotContains(t, facets.Body.String(), hidden.cards[0].String(), name)
		assert.NotContains(t, facets.Body.String(), hidden.rarity.String(), name)
		assert.NotContains(t, facets.Body.String(), "Trainer", name)
	}
}

// TestDeletingASharedCollectionEndsItsLink: the link is the Collection's Slug,
// so deleting the Collection leaves nothing to resolve.
func TestDeletingASharedCollectionEndsItsLink(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)
	id := createCollection(t, api, owner, "Base Set Binder")
	slug := *shareCollection(t, api, owner, id).Data.Slug
	base := sharedPath(api.profileSlug(t, owner.email), slug)
	require.Equal(t, http.StatusOK, api.Get(base).Code)

	resp := api.Delete("/collections/"+id, bearer(owner.token))
	require.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
	assert.Equal(t, http.StatusNotFound, api.Get(base).Code)
}

// TestShareEndpointsAreOwnerOnly: a Guest is refused, and another user sees
// the same missing Collection they would for any other verb.
func TestShareEndpointsAreOwnerOnly(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)
	other := newSharedOwner(t, api)
	id := createCollection(t, api, owner, "Base Set Binder")

	assert.Equal(t, http.StatusUnauthorized, api.Post("/collections/"+id+"/share").Code)
	assert.Equal(t, http.StatusUnauthorized, api.Delete("/collections/"+id+"/share").Code)
	assert.Equal(t, http.StatusNotFound, api.Post("/collections/"+id+"/share", bearer(other.token)).Code)
	assert.Equal(t, http.StatusNotFound, api.Delete("/collections/"+id+"/share", bearer(other.token)).Code)
	assert.Equal(t, http.StatusOK, api.Get("/collections/"+id, bearer(owner.token)).Code)
}

// sharedBinderEnvelope is the header's optional binder block: absent for a plain Collection.
type sharedBinderEnvelope struct {
	Data struct {
		Binder *struct {
			Layout            binderLayoutJSON `json:"layout"`
			UnidentifiedCount int              `json:"unidentifiedCount"`
		} `json:"binder"`
	} `json:"data"`
}

// TestSharedBinderShowsItsLayoutPagesAndSlotsToEveryViewer covers a Shared binder
// as a Guest, a signed-in viewer and the owner. A Slot holding a Card of an
// unpublished set reads as empty with no Card, and is not counted.
func TestSharedBinderShowsItsLayoutPagesAndSlotsToEveryViewer(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)
	other := newSharedOwner(t, api)
	live := newTestCards(t, 1)[0]
	hidden := seedSharedSet(t, api, false, "Trainer", "tag-"+uuid.NewString(), 1).cards[0]
	id := createCollection(t, api, owner, "Shared Binder")
	createBinder(t, api, owner.token, id, 2, 2, 1)
	placeSlot(t, api, id, 1, 1, 2, &live, false)
	placeSlot(t, api, id, 1, 2, 2, nil, true)
	placeSlot(t, api, id, 2, 1, 1, &hidden, false)
	require.NoError(t, api.db.Create(&entity.InventoryEntry{CollectionID: uuid.MustParse(id), CardID: live, Quantity: 1}).Error)
	base := sharedPath(api.profileSlug(t, owner.email), *shareCollection(t, api, owner, id).Data.Slug)

	viewers := map[string]string{"guest": "", "signed-in": bearer(other.token), "owner": bearer(owner.token)}
	for name, auth := range viewers {
		var args []any
		if auth != "" {
			args = append(args, auth)
		}

		header := api.Get(base, args...)
		require.Equal(t, http.StatusOK, header.Code, name+": "+header.Body.String())
		assert.NotContains(t, header.Body.String(), owner.email, name+" must never see the owner's email")
		var h sharedBinderEnvelope
		require.NoError(t, json.Unmarshal(header.Body.Bytes(), &h))
		require.NotNil(t, h.Data.Binder, name)
		assert.Equal(t, binderLayoutJSON{Rows: 2, Columns: 2, LeafCount: 1, Capacity: 8}, h.Data.Binder.Layout, name)
		assert.Equal(t, 1, h.Data.Binder.UnidentifiedCount, name)

		page1 := api.Get(base+"/pages/1", args...)
		require.Equal(t, http.StatusOK, page1.Code, name+": "+page1.Body.String())
		assert.NotContains(t, page1.Body.String(), owner.email, name+" must never see the owner's email")
		var p1 binderPageEnvelope
		require.NoError(t, json.Unmarshal(page1.Body.Bytes(), &p1))
		assert.Equal(t, 2, p1.Data.CardTotal, name+": hidden Cards are not counted")
		assert.Equal(t, 1, p1.Data.UnidentifiedCount, name)
		require.Len(t, p1.Data.Slots, 4, name)
		assert.Equal(t, "empty", p1.Data.Slots[0].State, name)
		assert.Equal(t, "card", p1.Data.Slots[1].State, name)
		require.NotNil(t, p1.Data.Slots[1].Card, name)
		assert.Equal(t, live.String(), p1.Data.Slots[1].Card.ID, name)
		assert.Equal(t, "unidentified", p1.Data.Slots[3].State, name)
		assert.Nil(t, p1.Data.Slots[3].Card, name)

		page2 := api.Get(base+"/pages/2", args...)
		require.Equal(t, http.StatusOK, page2.Code, name+": "+page2.Body.String())
		assert.NotContains(t, page2.Body.String(), hidden.String(), name+" must never see a Card of an unpublished set")
		assert.NotContains(t, page2.Body.String(), owner.email, name+" must never see the owner's email")
		var p2 binderPageEnvelope
		require.NoError(t, json.Unmarshal(page2.Body.Bytes(), &p2))
		assert.Equal(t, 2, p2.Data.CardTotal, name)
		require.Len(t, p2.Data.Slots, 4, name)
		assert.Equal(t, "empty", p2.Data.Slots[0].State, name)
		assert.Nil(t, p2.Data.Slots[0].Card, name)
	}

	// The shared Card matches the one the guest entries listing shows.
	entries := api.Get(base + "/entries")
	require.Equal(t, http.StatusOK, entries.Code, entries.Body.String())
	var listed struct {
		Data []struct {
			Card json.RawMessage `json:"card"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(entries.Body.Bytes(), &listed))
	require.Len(t, listed.Data, 1)
	shared := api.Get(base + "/pages/1")
	var slotCards struct {
		Data struct {
			Slots []struct {
				Card json.RawMessage `json:"card"`
			} `json:"slots"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(shared.Body.Bytes(), &slotCards))
	assert.JSONEq(t, string(listed.Data[0].Card), string(slotCards.Data.Slots[1].Card))

	// The owner's own Page read still shows the hidden Card, so only the shared view hides it.
	ownerPage := getPage(t, api, owner.token, id, 2)
	assert.Equal(t, 3, ownerPage.Data.CardTotal)
	assert.Equal(t, "card", ownerPage.Data.Slots[0].State)
}

// TestSharedBinderPagesAnswerEveryMissWithTheSameNotFound covers a plain Collection
// (no binder block, Pages absent), a Page past the last Leaf, an unshared binder
// and a missing one: each Page read is the same 404 as the header's.
func TestSharedBinderPagesAnswerEveryMissWithTheSameNotFound(t *testing.T) {
	api := newTestAPI(t)
	owner := newSharedOwner(t, api)
	plain := createCollection(t, api, owner, "Plain Shared")
	profile := api.profileSlug(t, owner.email)
	plainBase := sharedPath(profile, *shareCollection(t, api, owner, plain).Data.Slug)

	binder := createCollection(t, api, owner, "Binder Shared")
	createBinder(t, api, owner.token, binder, 2, 2, 1)
	binderBase := sharedPath(profile, *shareCollection(t, api, owner, binder).Data.Slug)

	private := createCollection(t, api, owner, "Unshared Binder")
	createBinder(t, api, owner.token, private, 2, 2, 1)

	plainHeader := api.Get(plainBase)
	require.Equal(t, http.StatusOK, plainHeader.Code, plainHeader.Body.String())
	var raw struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(plainHeader.Body.Bytes(), &raw))
	assert.NotContains(t, raw.Data, "binder", "a plain Collection's header has no binder block")

	reference := sharedMissShape(api.Get(sharedPath(profile, "no-such-binder-00000") + "/pages/1"))
	targets := map[string]string{
		"plain-collection": plainBase + "/pages/1",
		"page-past-last":   binderBase + "/pages/3",
		"unshared-binder":  sharedPath(profile, "unshared-binder-"+slugTail(private)) + "/pages/1",
	}
	for name, path := range targets {
		resp := api.Get(path)
		require.Equal(t, http.StatusNotFound, resp.Code, name+": "+resp.Body.String())
		assert.Equal(t, reference, sharedMissShape(resp), name+" must look exactly like a missing target")
	}
}
