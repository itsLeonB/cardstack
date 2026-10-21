package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedNameGroups embeds four name groups under model, all in the fixture's set:
//   - Tune Pikachu, three Cards: 001 and 002 at distance 0.04, 003 at 1.0 from both.
//   - Tune Able, two Cards with the same art: distance 0.
//   - Tune Energy, two Cards with different art: distance 1.
//   - Tune Solo, a single Card, which is no group.
//
// Local ids are unique in a set, so the groups take 001 to 008 between them.
func seedNameGroups(t *testing.T, db *gorm.DB, f fixture, model string) {
	t.Helper()
	pikachu1 := f.card(t, "Tune Pikachu", "001")
	pikachu2 := f.card(t, "Tune Pikachu", "002")
	pikachu3 := f.card(t, "Tune Pikachu", "003")
	embed(t, db, pikachu1.ID, model, unitVector(0))
	embed(t, db, pikachu2.ID, model, mixVector(map[int]float64{0: 0.96, 1: 0.28}))
	embed(t, db, pikachu3.ID, model, unitVector(2))

	able1 := f.card(t, "Tune Able", "004")
	able2 := f.card(t, "Tune Able", "005")
	embed(t, db, able1.ID, model, unitVector(3))
	embed(t, db, able2.ID, model, unitVector(3))

	energy1 := f.card(t, "Tune Energy", "006")
	energy2 := f.card(t, "Tune Energy", "007")
	embed(t, db, energy1.ID, model, unitVector(4))
	embed(t, db, energy2.ID, model, unitVector(5))

	solo := f.card(t, "Tune Solo", "008")
	embed(t, db, solo.ID, model, unitVector(6))
}

func TestReprints_LargestGroupsFirstWithPairwiseDistancesAndTheDelta(t *testing.T) {
	db := testDB(t)
	f := newFixture(t, db)
	model := "tune-" + uuid.NewString()
	seedNameGroups(t, db, f, model)
	h := newHarness(t, db, model, nil)
	code := h.run("reprints")

	require.Equal(t, 0, code, h.stderr.String())
	c := f.set.Code
	assert.Equal(t, []string{
		"MATCH_REPRINT_DELTA 0.050: a pair at or under this distance is the same artwork",
		"3 of 3 name group(s) with at least 2 embedded Cards, largest first",
		"tune pikachu (3 Cards)",
		c + " 001 " + c + " 002 0.040 within delta",
		c + " 001 " + c + " 003 1.000 beyond delta",
		c + " 002 " + c + " 003 1.000 beyond delta",
		"tune able (2 Cards)",
		c + " 004 " + c + " 005 0.000 within delta",
		"tune energy (2 Cards)",
		c + " 006 " + c + " 007 1.000 beyond delta",
	}, lines(h.stdout.String()))
}

func TestReprints_LimitAndMinSizeNarrowTheGroups(t *testing.T) {
	db := testDB(t)
	f := newFixture(t, db)
	model := "tune-" + uuid.NewString()
	seedNameGroups(t, db, f, model)
	c := f.set.Code
	pikachu := []string{
		"tune pikachu (3 Cards)",
		c + " 001 " + c + " 002 0.040 within delta",
		c + " 001 " + c + " 003 1.000 beyond delta",
		c + " 002 " + c + " 003 1.000 beyond delta",
	}

	h := newHarness(t, db, model, nil)
	require.Equal(t, 0, h.run("reprints", "--limit", "1"), h.stderr.String())
	assert.Equal(t, append([]string{
		"MATCH_REPRINT_DELTA 0.050: a pair at or under this distance is the same artwork",
		"1 of 3 name group(s) with at least 2 embedded Cards, largest first",
	}, pikachu...), lines(h.stdout.String()), "--limit keeps the largest group only")

	h = newHarness(t, db, model, nil)
	require.Equal(t, 0, h.run("reprints", "--min-size", "3"), h.stderr.String())
	assert.Equal(t, append([]string{
		"MATCH_REPRINT_DELTA 0.050: a pair at or under this distance is the same artwork",
		"1 of 1 name group(s) with at least 3 embedded Cards, largest first",
	}, pikachu...), lines(h.stdout.String()), "--min-size drops the groups under it")
}

func TestReprints_AGroupOverTheCapIsPairedForItsFirstCardsOnly(t *testing.T) {
	db := testDB(t)
	f := newFixture(t, db)
	model := "tune-" + uuid.NewString()
	for i := range 9 {
		card := f.card(t, "Tune Nine", fmt.Sprintf("%03d", i+1))
		embed(t, db, card.ID, model, unitVector(7))
	}
	h := newHarness(t, db, model, nil)

	code := h.run("reprints")

	require.Equal(t, 0, code, h.stderr.String())
	out := h.stdout.String()
	assert.Equal(t, 28, strings.Count(out, "within delta"), "8 Cards make 28 pairs")
	assert.Contains(t, out, "pairwise shown for the first 8 of 9 Cards (the cap is 8)")
	assert.NotContains(t, out, f.set.Code+" 009 "+f.set.Code+" 001", "the ninth Card is not paired")
}

func TestReprints_NoEmbeddingsForTheModelFailsWithoutOutput(t *testing.T) {
	db := testDB(t)
	model := "tune-" + uuid.NewString()
	h := newHarness(t, db, model, nil)

	code := h.run("reprints")

	assert.Equal(t, 1, code)
	assert.Contains(t, h.stderr.String(), `the catalog has no embeddings for model "`+model+`"`)
	assert.Empty(t, h.stdout.String())
}

func TestReprints_WritesNothing(t *testing.T) {
	db := testDB(t)
	f := newFixture(t, db)
	model := "tune-" + uuid.NewString()
	seedNameGroups(t, db, f, model)
	h := newHarness(t, db, model, nil)
	sets, cards, embeddings := writeCounts(t, db, f, model)

	require.Equal(t, 0, h.run("reprints"), h.stderr.String())

	gotSets, gotCards, gotEmbeddings := writeCounts(t, db, f, model)
	assert.Equal(t, sets, gotSets)
	assert.Equal(t, cards, gotCards)
	assert.Equal(t, embeddings, gotEmbeddings)
}

func TestReprints_BadFlagsAreUsageErrors(t *testing.T) {
	db := testDB(t)
	cases := map[string][]string{
		"zero limit":     {"reprints", "--limit", "0"},
		"min size one":   {"reprints", "--min-size", "1"},
		"stray argument": {"reprints", "extra"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t, db, "tune-"+uuid.NewString(), nil)

			code := h.run(args...)

			assert.Equal(t, 2, code, h.stderr.String())
			assert.Equal(t, 0, h.opens, "a bad flag opens nothing")
		})
	}
}
