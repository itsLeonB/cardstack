package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// catalogFixture inserts a Game, Locale, Rarity and (optionally) Series row
// unique to this test run, since this database is shared with other
// packages' tests (see repository_test_helper_test.go's testDB doc
// comment) and never truncated between runs.
type catalogFixture struct {
	game   entity.Game
	locale entity.Locale
	rarity entity.Rarity
}

func newCatalogFixture(t *testing.T, db *gorm.DB) catalogFixture {
	t.Helper()
	suffix := uuid.NewString()

	game := entity.Game{Slug: "catalog-test-" + suffix, Name: "Catalog Test Game " + suffix}
	if err := db.Create(&game).Error; err != nil {
		t.Fatalf("creating game fixture: %v", err)
	}

	locale := entity.Locale{Code: "catalog-test-" + suffix}
	if err := db.Create(&locale).Error; err != nil {
		t.Fatalf("creating locale fixture: %v", err)
	}

	rarity := entity.Rarity{GameID: game.ID, Code: "CT-" + suffix, Name: "Catalog Test Rarity " + suffix}
	if err := db.Create(&rarity).Error; err != nil {
		t.Fatalf("creating rarity fixture: %v", err)
	}

	return catalogFixture{game: game, locale: locale, rarity: rarity}
}

func (f catalogFixture) newSeries(t *testing.T, db *gorm.DB) entity.Series {
	t.Helper()
	suffix := uuid.NewString()

	series := entity.Series{GameID: f.game.ID, Code: "series-" + suffix, Name: "Catalog Test Series " + suffix}
	if err := db.Create(&series).Error; err != nil {
		t.Fatalf("creating series fixture: %v", err)
	}

	return series
}

func (f catalogFixture) newExpansionSet(t *testing.T, db *gorm.DB, seriesID *uuid.UUID, releaseDate *time.Time) entity.ExpansionSet {
	t.Helper()
	suffix := uuid.NewString()

	set := entity.ExpansionSet{
		GameID:      f.game.ID,
		Code:        "set-" + suffix,
		Name:        "Catalog Test Set " + suffix,
		LocaleID:    f.locale.ID,
		SeriesID:    seriesID,
		ReleaseDate: releaseDate,
	}
	if err := db.Create(&set).Error; err != nil {
		t.Fatalf("creating expansion set fixture: %v", err)
	}

	return set
}

func (f catalogFixture) newCard(t *testing.T, db *gorm.DB, setID uuid.UUID, mutate func(*entity.Card)) entity.Card {
	t.Helper()
	suffix := uuid.NewString()

	card := entity.Card{
		ExpansionSetID: setID,
		LocalID:        "001",
		Name:           "Catalog Test Card " + suffix,
		Category:       "Pokémon",
		Tags:           datatypes.JSONSlice[string]{},
		RarityID:       f.rarity.ID,
		Attributes:     datatypes.JSONMap{},
	}
	if mutate != nil {
		mutate(&card)
	}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("creating card fixture: %v", err)
	}

	return card
}

func TestCatalogRepository_ListSeries(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	series := fixture.newSeries(t, db)

	repo := NewCatalogRepository(db)
	all, err := repo.ListSeries(context.Background())
	if err != nil {
		t.Fatalf("ListSeries: %v", err)
	}

	found := false
	for _, s := range all {
		if s.ID == series.ID {
			found = true
			if s.Code != series.Code || s.Name != series.Name {
				t.Fatalf("ListSeries returned %+v, want Code=%q Name=%q", s, series.Code, series.Name)
			}
		}
	}
	if !found {
		t.Fatalf("ListSeries did not return the fixture series %s", series.ID)
	}
}

func TestCatalogRepository_ListExpansionSets(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	series := fixture.newSeries(t, db)
	otherSeries := fixture.newSeries(t, db)

	later := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	earlier := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	setNoDate := fixture.newExpansionSet(t, db, &series.ID, nil)
	setLater := fixture.newExpansionSet(t, db, &series.ID, &later)
	setEarlier := fixture.newExpansionSet(t, db, &series.ID, &earlier)
	// Belongs to a different Series - must not be returned when we ask for
	// series.ID only.
	fixture.newExpansionSet(t, db, &otherSeries.ID, nil)

	repo := NewCatalogRepository(db)
	sets, err := repo.ListExpansionSets(context.Background(), []uuid.UUID{series.ID})
	if err != nil {
		t.Fatalf("ListExpansionSets: %v", err)
	}
	if len(sets) != 3 {
		t.Fatalf("expected 3 expansion sets for series %s, got %d: %+v", series.ID, len(sets), sets)
	}

	// Ordered by release date ascending, with a nil release date sorting
	// last.
	wantOrder := []uuid.UUID{setEarlier.ID, setLater.ID, setNoDate.ID}
	for i, want := range wantOrder {
		if sets[i].ID != want {
			t.Fatalf("expected sets[%d].ID = %s, got %s (full order: %v)", i, want, sets[i].ID, idsOf(sets))
		}
	}
}

func TestCatalogRepository_ListUngroupedExpansionSets(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	series := fixture.newSeries(t, db)

	later := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	earlier := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	ungroupedNoDate := fixture.newExpansionSet(t, db, nil, nil)
	ungroupedLater := fixture.newExpansionSet(t, db, nil, &later)
	ungroupedEarlier := fixture.newExpansionSet(t, db, nil, &earlier)
	// Belongs to a Series - must not be returned.
	grouped := fixture.newExpansionSet(t, db, &series.ID, nil)

	repo := NewCatalogRepository(db)
	all, err := repo.ListUngroupedExpansionSets(context.Background())
	if err != nil {
		t.Fatalf("ListUngroupedExpansionSets: %v", err)
	}

	// This table is shared with other tests and never truncated (see
	// catalogFixture's doc comment), so filter the results down to just
	// this test's own fixture rows before asserting on order.
	want := []uuid.UUID{ungroupedEarlier.ID, ungroupedLater.ID, ungroupedNoDate.ID}
	wantSet := make(map[uuid.UUID]bool, len(want))
	for _, id := range want {
		wantSet[id] = true
	}

	var got []uuid.UUID
	for _, s := range all {
		if s.SeriesID != nil {
			if s.ID == grouped.ID {
				t.Fatalf("expected ListUngroupedExpansionSets to exclude the grouped fixture set %s", grouped.ID)
			}
			continue
		}
		if wantSet[s.ID] {
			got = append(got, s.ID)
		}
	}

	if len(got) != len(want) {
		t.Fatalf("expected to find all 3 fixture sets among ungrouped results, found %d: %v", len(got), got)
	}
	for i, id := range want {
		if got[i] != id {
			t.Fatalf("expected order %v, got %v", want, got)
		}
	}
}

func TestCatalogRepository_ListExpansionSets_EmptyIDs(t *testing.T) {
	db := testDB(t)
	repo := NewCatalogRepository(db)

	sets, err := repo.ListExpansionSets(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListExpansionSets: %v", err)
	}
	if len(sets) != 0 {
		t.Fatalf("expected no expansion sets for an empty seriesIDs, got %d", len(sets))
	}
}

func idsOf(sets []entity.ExpansionSet) []uuid.UUID {
	ids := make([]uuid.UUID, len(sets))
	for i, s := range sets {
		ids[i] = s.ID
	}
	return ids
}

func TestCatalogRepository_ListRarities(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)

	repo := NewCatalogRepository(db)
	rarities, err := repo.ListRarities(context.Background())
	if err != nil {
		t.Fatalf("ListRarities: %v", err)
	}

	found := false
	for _, r := range rarities {
		if r.ID == fixture.rarity.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("ListRarities did not return the fixture rarity %s", fixture.rarity.ID)
	}
}

func TestCatalogRepository_ListDistinctCategories(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	series := fixture.newSeries(t, db)
	set := fixture.newExpansionSet(t, db, &series.ID, nil)

	uniqueCategory := "CatalogTestCategory-" + uuid.NewString()
	fixture.newCard(t, db, set.ID, func(c *entity.Card) { c.Category = uniqueCategory })

	repo := NewCatalogRepository(db)
	categories, err := repo.ListDistinctCategories(context.Background())
	if err != nil {
		t.Fatalf("ListDistinctCategories: %v", err)
	}

	if !containsString(categories, uniqueCategory) {
		t.Fatalf("expected ListDistinctCategories to include %q, got %v", uniqueCategory, categories)
	}
}

func TestCatalogRepository_ListDistinctTags(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	series := fixture.newSeries(t, db)
	set := fixture.newExpansionSet(t, db, &series.ID, nil)

	uniqueTag := "catalog-test-tag-" + uuid.NewString()
	fixture.newCard(t, db, set.ID, func(c *entity.Card) {
		c.Tags = datatypes.JSONSlice[string]{uniqueTag, "Basic"}
	})

	repo := NewCatalogRepository(db)
	tags, err := repo.ListDistinctTags(context.Background())
	if err != nil {
		t.Fatalf("ListDistinctTags: %v", err)
	}

	if !containsString(tags, uniqueTag) {
		t.Fatalf("expected ListDistinctTags to include %q, got %v", uniqueTag, tags)
	}
}

// TestCatalogRepository_SearchCards_StablePaginationAcrossTiedOrderKeys is a
// regression test for SearchCards's ORDER BY: two Expansion Sets sharing the
// same release date (or both nil) and cards across them sharing local_ids
// used to give Postgres no unique tiebreaker, so the same LIMIT/OFFSET query
// run twice (as pagination does) could return a card twice or skip it
// entirely. With expansion_sets.id and cards.id added as final tiebreakers,
// paginating through every page with a small limit must return every seeded
// card exactly once.
func TestCatalogRepository_SearchCards_StablePaginationAcrossTiedOrderKeys(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	series := fixture.newSeries(t, db)

	// Both sets share the same (nil) release date, so the old ORDER BY had
	// no way to break the tie between them.
	setA := fixture.newExpansionSet(t, db, &series.ID, nil)
	setB := fixture.newExpansionSet(t, db, &series.ID, nil)

	var want []uuid.UUID
	for i := range 5 {
		// Both sets use the same local_id/name at each i - colliding across
		// sets (the old ORDER BY's remaining keys couldn't break this tie
		// either) while staying unique within each set, which
		// idx_cards_expansion_set_id_local_id requires.
		localID := fmt.Sprintf("%03d", i)
		cardA := fixture.newCard(t, db, setA.ID, func(c *entity.Card) {
			c.LocalID = localID
			c.Name = "Tied Card " + localID
		})
		cardB := fixture.newCard(t, db, setB.ID, func(c *entity.Card) {
			c.LocalID = localID
			c.Name = "Tied Card " + localID
		})
		want = append(want, cardA.ID, cardB.ID)
	}

	repo := NewCatalogRepository(db)
	ctx := context.Background()

	const pageSize = 3
	seen := make(map[uuid.UUID]int)
	offset := 0
	for {
		page, total, err := repo.SearchCards(ctx, CardFilter{ExpansionSetID: uuid.Nil, RarityID: fixture.rarity.ID, Limit: pageSize, Offset: offset})
		if err != nil {
			t.Fatalf("SearchCards at offset %d: %v", offset, err)
		}
		if int(total) != len(want) {
			t.Fatalf("expected total = %d seeded cards, got %d", len(want), total)
		}
		for _, r := range page {
			seen[r.ID]++
		}
		offset += pageSize
		if offset >= int(total) {
			break
		}
	}

	for _, id := range want {
		if seen[id] != 1 {
			t.Fatalf("expected card %s to appear exactly once across all pages, appeared %d times", id, seen[id])
		}
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func TestCatalogRepository_SearchCards(t *testing.T) {
	db := testDB(t)
	fixture := newCatalogFixture(t, db)
	series := fixture.newSeries(t, db)
	set := fixture.newExpansionSet(t, db, &series.ID, nil)
	otherSet := fixture.newExpansionSet(t, db, &series.ID, nil)

	pikachu := fixture.newCard(t, db, set.ID, func(c *entity.Card) {
		c.LocalID = "001"
		c.Name = "Pikachu"
		c.Category = "Pokémon"
		c.Tags = datatypes.JSONSlice[string]{"Basic"}
	})
	raichu := fixture.newCard(t, db, set.ID, func(c *entity.Card) {
		c.LocalID = "002"
		c.Name = "Raichu"
		c.Category = "Pokémon"
		c.Tags = datatypes.JSONSlice[string]{"Stage 1"}
	})
	// A different Expansion Set - must be excluded when filtering by set.ID.
	fixture.newCard(t, db, otherSet.ID, func(c *entity.Card) {
		c.LocalID = "001"
		c.Name = "Pikachu"
	})

	repo := NewCatalogRepository(db)
	ctx := context.Background()

	t.Run("filters by expansion set and orders by local_id", func(t *testing.T) {
		results, total, err := repo.SearchCards(ctx, CardFilter{ExpansionSetID: set.ID, Limit: 10})
		if err != nil {
			t.Fatalf("SearchCards: %v", err)
		}
		if total != 2 {
			t.Fatalf("expected total 2, got %d", total)
		}
		if len(results) != 2 || results[0].ID != pikachu.ID || results[1].ID != raichu.ID {
			t.Fatalf("expected [pikachu, raichu] in local_id order, got %+v", results)
		}
		if results[0].ExpansionSetCode != set.Code || results[0].RarityCode != fixture.rarity.Code {
			t.Fatalf("expected joined expansion set/rarity data, got %+v", results[0])
		}
	})

	t.Run("filters by name, case-insensitive substring", func(t *testing.T) {
		results, total, err := repo.SearchCards(ctx, CardFilter{ExpansionSetID: set.ID, Name: "pika", Limit: 10})
		if err != nil {
			t.Fatalf("SearchCards: %v", err)
		}
		if total != 1 || len(results) != 1 || results[0].ID != pikachu.ID {
			t.Fatalf("expected only pikachu, got total=%d results=%+v", total, results)
		}
	})

	t.Run("filters by local id", func(t *testing.T) {
		results, total, err := repo.SearchCards(ctx, CardFilter{ExpansionSetID: set.ID, LocalID: "002", Limit: 10})
		if err != nil {
			t.Fatalf("SearchCards: %v", err)
		}
		if total != 1 || len(results) != 1 || results[0].ID != raichu.ID {
			t.Fatalf("expected only raichu, got total=%d results=%+v", total, results)
		}
	})

	t.Run("filters by rarity id", func(t *testing.T) {
		results, total, err := repo.SearchCards(ctx, CardFilter{ExpansionSetID: set.ID, RarityID: fixture.rarity.ID, Limit: 10})
		if err != nil {
			t.Fatalf("SearchCards: %v", err)
		}
		if total != 2 || len(results) != 2 {
			t.Fatalf("expected both cards to match the shared rarity, got total=%d results=%+v", total, results)
		}
	})

	t.Run("filters by tag", func(t *testing.T) {
		results, total, err := repo.SearchCards(ctx, CardFilter{ExpansionSetID: set.ID, Tag: "Stage 1", Limit: 10})
		if err != nil {
			t.Fatalf("SearchCards: %v", err)
		}
		if total != 1 || len(results) != 1 || results[0].ID != raichu.ID {
			t.Fatalf("expected only raichu, got total=%d results=%+v", total, results)
		}
	})

	t.Run("paginates with limit and offset", func(t *testing.T) {
		page1, total, err := repo.SearchCards(ctx, CardFilter{ExpansionSetID: set.ID, Limit: 1, Offset: 0})
		if err != nil {
			t.Fatalf("SearchCards page1: %v", err)
		}
		if total != 2 || len(page1) != 1 || page1[0].ID != pikachu.ID {
			t.Fatalf("expected page1 = [pikachu], total=2, got total=%d page1=%+v", total, page1)
		}

		page2, total, err := repo.SearchCards(ctx, CardFilter{ExpansionSetID: set.ID, Limit: 1, Offset: 1})
		if err != nil {
			t.Fatalf("SearchCards page2: %v", err)
		}
		if total != 2 || len(page2) != 1 || page2[0].ID != raichu.ID {
			t.Fatalf("expected page2 = [raichu], total=2, got total=%d page2=%+v", total, page2)
		}
	})

	t.Run("no filters other than expansion set still scopes results", func(t *testing.T) {
		results, total, err := repo.SearchCards(ctx, CardFilter{ExpansionSetID: uuid.Nil, LocalID: "001", Name: pikachu.Name, Limit: 10})
		if err != nil {
			t.Fatalf("SearchCards: %v", err)
		}
		if total < 1 {
			t.Fatalf("expected at least one match across all expansion sets, got total=%d results=%+v", total, results)
		}
	})
}
