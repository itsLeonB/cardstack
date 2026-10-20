package tcgcollector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixtureBytes(t *testing.T, name string) []byte {
	t.Helper()

	page, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return page
}

func TestParseCardList_ReadsIDNameNumberAndLargestImage(t *testing.T) {
	cards, err := parseCardList(fixtureBytes(t, "card-list-shiny-treasure.html"))
	require.NoError(t, err)
	require.Len(t, cards, 3)

	assert.Equal(t, cardListing{
		TCGCollectorID: "71277",
		Name:           "Oddish",
		Number:         "001/190",
		PageURL:        "https://www.tcgcollector.com/cards/71277/oddish-shiny-treasure-ex-001-190",
		ImageURL:       "https://static.tcgcollector.com/content/images/70/50/45/7050451f2b790a0e571d4f2b049a58d765d43a90ff8c08d3bfe8c3f3d2df25c7.webp",
	}, cards[0])
}

func TestParseCardList_FailsOnAPageWithNoCards(t *testing.T) {
	_, err := parseCardList([]byte("<html><body></body></html>"))
	assert.ErrorContains(t, err, "no cards found")
}

func TestParseCardPage_ReadsPokemon(t *testing.T) {
	page, err := parseCardPage(fixtureBytes(t, "card-pokemon-mega-venusaur.html"))
	require.NoError(t, err)

	assert.Equal(t, categoryPokemon, page.Category)
	assert.Empty(t, page.Tags)
	assert.Equal(t, "RR", page.Rarity)
	assert.Equal(t, "5ban Graphics", page.Illustrator)
	assert.Equal(t, "I", page.RegulationMark)
}

func TestParseCardPage_PromoWithNoRarityMarkIsPokemon(t *testing.T) {
	page, err := parseCardPage(fixtureBytes(t, "card-pokemon-detective-pikachu.html"))
	require.NoError(t, err)

	assert.Equal(t, categoryPokemon, page.Category)
	assert.Equal(t, noMark, page.Rarity, `"—" stays a code for the caller to map to Tanpa Tanda`)
	assert.Equal(t, "J", page.RegulationMark)
}

func TestParseCardPage_ReadsTrainerWithItsTags(t *testing.T) {
	page, err := parseCardPage(fixtureBytes(t, "card-trainer-justified-gloves.html"))
	require.NoError(t, err)

	assert.Equal(t, categoryTrainer, page.Category)
	assert.Equal(t, []string{"Item", "Pokémon Tool"}, page.Tags)
	assert.Equal(t, "U", page.Rarity)
}

func TestParseCardPage_ReadsSupporter(t *testing.T) {
	page, err := parseCardPage(fixtureBytes(t, "card-trainer-caitlin.html"))
	require.NoError(t, err)

	assert.Equal(t, categoryTrainer, page.Category)
	assert.Equal(t, []string{"Supporter"}, page.Tags)
}

func TestParseCardPage_ReadsBasicEnergyAsEnergiDasar(t *testing.T) {
	page, err := parseCardPage(fixtureBytes(t, "card-energy-basic-fire.html"))
	require.NoError(t, err)

	assert.Equal(t, categoryEnergi, page.Category)
	assert.Equal(t, []string{"Energi Dasar"}, page.Tags)
	assert.Equal(t, noMark, page.Rarity)
}

func TestParseCardPage_ReadsSpecialEnergyAndDropsOtherLabels(t *testing.T) {
	page, err := parseCardPage(fixtureBytes(t, "card-energy-spiral.html"))
	require.NoError(t, err)

	assert.Equal(t, categoryEnergi, page.Category, "a label ending in Energy makes an Energi")
	assert.Equal(t, []string{"Energi Spesial"}, page.Tags, "Rapid Strike Energy has no tag")
}

func TestParseCardPage_KeepsRawAsStrippedMain(t *testing.T) {
	page, err := parseCardPage(fixtureBytes(t, "card-pokemon-mega-venusaur.html"))
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(page.Raw, "<main"), "raw starts with the main element")
	assert.NotContains(t, page.Raw, "<script")
	assert.NotContains(t, page.Raw, "<svg")
	assert.NotContains(t, page.Raw, "\n", "runs of whitespace are collapsed")
	assert.NotContains(t, page.Raw, "  ")
}

func TestCardNumber_ReducesPrintedNumbersToIntegers(t *testing.T) {
	cases := map[string]struct {
		printed string
		want    int
		ok      bool
	}{
		"slash and set total": {"119/126", 119, true},
		"zero padded":         {"001/190", 1, true},
		"promo code":          {"001/SM-P", 1, true},
		"number word":         {"No. 201", 201, true},
		"code with no digits": {"SM-P", 0, false},
		"text before slash":   {"R/RGB", 0, false},
		"empty":               {"", 0, false},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, ok := cardNumber(c.printed)
			assert.Equal(t, c.ok, ok)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestTextCardID_KeepsWholeANumberWhoseTextBeforeTheSlashHasNoDigits(t *testing.T) {
	cases := map[string]struct {
		printed string
		want    string
		ok      bool
	}{
		"letter and set code": {"R/RGB", "R/RGB", true},
		"trimmed":             {" G/RGB ", "G/RGB", true},
		"digits before slash": {"119/126", "", false},
		"number word":         {"No. 201", "", false},
		"no slash":            {"SM-P", "", false},
		"empty before slash":  {"/RGB", "", false},
		"empty":               {"", "", false},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, ok := textCardID(c.printed)
			assert.Equal(t, c.ok, ok)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestCardName_DropsTheSetAndNumberSuffix(t *testing.T) {
	assert.Equal(t, "Oddish", cardName("Oddish (Shiny Treasure ex 001/190)"))
	assert.Equal(t, "Pikachu (Pikachu VMAX)", cardName("Pikachu (Pikachu VMAX) (Set 001/190)"))
	assert.Equal(t, "Bare", cardName("Bare"))
}

func TestNumberWidth_IsTheLongestNumericID(t *testing.T) {
	cards := func(ids ...string) []entity.Card {
		var out []entity.Card
		for _, id := range ids {
			out = append(out, entity.Card{LocalID: id})
		}
		return out
	}

	assert.Equal(t, 3, numberWidth(nil), "a set with no numeric ID defaults to 3")
	assert.Equal(t, 3, numberWidth(cards("SM-P")), "non-numeric IDs are ignored")
	assert.Equal(t, 1, numberWidth(cards("1", "9")))
	assert.Equal(t, 3, numberWidth(cards("1", "002", "SV-P")))
}

func TestRarityCode_ReadsTheCodeInParentheses(t *testing.T) {
	assert.Equal(t, "RR", rarityCode("Double Rare (RR)"))
	assert.Equal(t, "U", rarityCode("Uncommon (U)"))
	assert.Equal(t, "Bare", rarityCode("Bare"))
}

func TestIsBasicEnergy_MatchesByName(t *testing.T) {
	assert.True(t, isBasicEnergy("Basic Fire Energy"))
	assert.False(t, isBasicEnergy("Spiral Energy"))
	assert.False(t, isBasicEnergy("Basic Trainer"))
}

func TestRarityLookup_ResolvesCodesAndTheTanpaTandaMark(t *testing.T) {
	doubleRare, tanpa := uuid.New(), uuid.New()
	lookup := rarityLookup{codes: map[string]uuid.UUID{"RR": doubleRare}, tanpaTanda: tanpa}

	got, err := lookup.resolve("RR")
	require.NoError(t, err)
	assert.Equal(t, doubleRare, got)

	got, err = lookup.resolve(noMark)
	require.NoError(t, err)
	assert.Equal(t, tanpa, got)

	_, err = lookup.resolve("SAR")
	assert.ErrorContains(t, err, "rarity SAR is not in rarities", "an unknown code is an error here, rarityFor creates it")
}
