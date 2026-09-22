package pokemonasia

import (
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func mustParseFixture(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)
	return doc
}

const expansionListFixture = `
<html><body>
<ul class="expansionList">
  <li class="expansion">
    <a class="expansionLink" href="/id/card-search/list/?expansionCodes=MA6">
      <div class="rightColumn">
        <div class="seriesBlock"><span class="series">Evolusi Mega</span></div>
        <div class="titleBlock">
          <h3 class="expansionTitle">Booster Pack "30th CELEBRATION"</h3>
          <label class="releaseDateLabel">Tanggal Penjualan</label>
          <time class="relaseDate" datetime="09-16-2026"><span>09-16-2026</span></time>
        </div>
      </div>
    </a>
  </li>
  <li class="expansion">
    <a class="expansionLink" href="/id/card-search/list/?expansionCodes=TW1">
      <div class="rightColumn">
        <div class="seriesBlock"><span class="series">Taiwan Only</span></div>
        <div class="titleBlock">
          <h3 class="expansionTitle">Out of scope set</h3>
          <time class="relaseDate" datetime="01-01-2020"><span>01-01-2020</span></time>
        </div>
      </div>
    </a>
  </li>
  <li class="expansion">
    <a class="expansionLink" href="/id/card-search/list/?expansionCodes=NODATE">
      <div class="rightColumn">
        <div class="seriesBlock"><span class="series">Evolusi Mega</span></div>
        <div class="titleBlock">
          <h3 class="expansionTitle">Missing date, must be skipped</h3>
        </div>
      </div>
    </a>
  </li>
</ul>
</body></html>`

func TestParseExpansionListings(t *testing.T) {
	doc := mustParseFixture(t, expansionListFixture)

	got := parseExpansionListings(doc)

	require.Len(t, got, 2, "the listing with no parseable release date must be skipped")
	assert.Equal(t, expansionListing{
		Series:      "Evolusi Mega",
		Code:        "MA6",
		Name:        `Booster Pack "30th CELEBRATION"`,
		ReleaseDate: time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC),
	}, got[0])
	assert.Equal(t, "TW1", got[1].Code, "out-of-scope Series listings are still parsed; filtering happens in ingest.go")
}

const resultsPageFixture = `
<html><body>
<p class="resultNumber">2</p>
<ul class="list">
  <li class="card">
    <a href="/id/card-search/detail/16488/">
      <div class="imageContainer loading"><img class="lazy" data-original="https://asia.pokemon-card.com/id/card-img/id00016488.png"></div>
    </a>
  </li>
  <li class="card">
    <a href="/id/card-search/detail/16489/"><div class="imageContainer"></div></a>
  </li>
</ul>
</body></html>`

func TestParseResultCardIDs(t *testing.T) {
	doc := mustParseFixture(t, resultsPageFixture)

	got := parseResultCardIDs(doc)

	assert.Equal(t, []string{"16488", "16489"}, got)
}

func TestParseResultCardIDs_EmptyPage(t *testing.T) {
	doc := mustParseFixture(t, `<html><body><ul class="list"></ul></body></html>`)

	got := parseResultCardIDs(doc)

	assert.Nil(t, got)
}

const pokemonDetailFixture = `
<html><body>
<h1 class="pageHeader cardDetail"><span class="evolveMarker">Stage 2</span> Mega Venusaur ex</h1>
<div class="skillInformation"><h3 class="commonHeader">Serangan</h3></div>
<section class="expansionColumn">
  <span class="alpha">I</span>
  <span class="collectorNumber">001/126</span>
</section>
<div class="illustrator"><a href="/id/card-search/list/?illustratorName=HYOGONOSUKE">HYOGONOSUKE</a></div>
</body></html>`

const trainerDetailFixture = `
<html><body>
<h1 class="pageHeader cardDetail">Ultra Ball</h1>
<div class="skillInformation"><h3 class="commonHeader">Item</h3></div>
<section class="expansionColumn">
  <span class="alpha">C</span>
  <span class="collectorNumber">150/126</span>
</section>
<div class="illustrator"><a href="/id/card-search/list/?illustratorName=someone">someone</a></div>
</body></html>`

const energyDetailFixture = `
<html><body>
<h1 class="pageHeader cardDetail">Energi Air Dasar</h1>
<div class="skillInformation"><h3 class="commonHeader">Energi Dasar</h3></div>
<section class="expansionColumn">
  <span class="alpha">C</span>
  <span class="collectorNumber">PROMO-A</span>
</section>
</body></html>`

func TestParseCardDetail(t *testing.T) {
	tests := []struct {
		name string
		html string
		want cardDetail
	}{
		{
			name: "pokemon card with evolve marker and ex suffix",
			html: pokemonDetailFixture,
			want: cardDetail{
				Name:        "Mega Venusaur ex",
				Category:    categoryPokemon,
				Tag:         "",
				RarityCode:  "I",
				LocalID:     "001",
				Illustrator: "HYOGONOSUKE",
				Attributes: map[string]any{
					"stage":          "Stage 2",
					"specialMarkers": []string{"ex"},
				},
			},
		},
		{
			name: "trainer card",
			html: trainerDetailFixture,
			want: cardDetail{
				Name:        "Ultra Ball",
				Category:    categoryTrainer,
				Tag:         "Item",
				RarityCode:  "C",
				LocalID:     "150",
				Illustrator: "someone",
				Attributes:  map[string]any{},
			},
		},
		{
			name: "energy card with non-numeric collector number",
			html: energyDetailFixture,
			want: cardDetail{
				Name:        "Energi Air Dasar",
				Category:    categoryEnergi,
				Tag:         "Energi Dasar",
				RarityCode:  "C",
				LocalID:     "PROMO-A",
				Illustrator: "",
				Attributes:  map[string]any{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := mustParseFixture(t, tt.html)
			got := parseCardDetail(doc)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectCategory(t *testing.T) {
	tests := []struct {
		name         string
		html         string
		wantCategory string
		wantTag      string
	}{
		{"pokemon", pokemonDetailFixture, categoryPokemon, ""},
		{"trainer", trainerDetailFixture, categoryTrainer, "Item"},
		{"energy", energyDetailFixture, categoryEnergi, "Energi Dasar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := mustParseFixture(t, tt.html)
			category, tag := detectCategory(doc)
			assert.Equal(t, tt.wantCategory, category)
			assert.Equal(t, tt.wantTag, tag)
		})
	}
}

func TestDetectSpecialMarkers(t *testing.T) {
	tests := []struct {
		name string
		card string
		want []string
	}{
		{"ex suffix", "Mega Venusaur ex", []string{"ex"}},
		{"V suffix", "Pikachu V", []string{"V"}},
		{"V-UNION not shadowed by V", "Kyogre & Groudon V-UNION", []string{"V-UNION"}},
		{"VMAX not shadowed by V", "Charizard VMAX", []string{"VMAX"}},
		{"no marker", "Ultra Ball", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, detectSpecialMarkers(tt.card))
		})
	}
}

func TestSplitCollectorNumber(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"delimited", "001/126", "001"},
		{"non-delimited", "PROMO-A", "PROMO-A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, splitCollectorNumber(tt.in))
		})
	}
}

func TestCardImageURL(t *testing.T) {
	assert.Equal(t, "https://asia.pokemon-card.com/id/card-img/id00016488.png", cardImageURL("16488"))
	assert.Equal(t, "", cardImageURL("not-a-number"))
}

func TestSlugifySeries(t *testing.T) {
	assert.Equal(t, "scarlet-violet", slugifySeries("Scarlet & Violet"))
	assert.Equal(t, "evolusi-mega", slugifySeries("Evolusi Mega"))
}

func TestMapCard(t *testing.T) {
	expansionSetID := uuid.New()
	rarityID := uuid.New()
	raw := []byte(`<html>stub</html>`)

	tests := []struct {
		name   string
		detail cardDetail
		want   entity.Card
	}{
		{
			name: "pokemon card with regulation and attributes",
			detail: cardDetail{
				Name:        "Mega Venusaur ex",
				Category:    categoryPokemon,
				Tag:         "",
				RarityCode:  "I",
				LocalID:     "001",
				Illustrator: "HYOGONOSUKE",
				Regulation:  "Standar",
				Attributes:  map[string]any{"stage": "Stage 2"},
			},
			want: entity.Card{
				ExpansionSetID: expansionSetID,
				LocalID:        "001",
				Name:           "Mega Venusaur ex",
				Category:       categoryPokemon,
				Illustrator:    "HYOGONOSUKE",
				Tags:           datatypes.JSONSlice[string]{},
				RarityID:       rarityID,
				ImageURL:       "https://asia.pokemon-card.com/id/card-img/id00016488.png",
				Attributes:     datatypes.JSONMap{"stage": "Stage 2", "regulation": "Standar"},
				Raw:            string(raw),
			},
		},
		{
			name: "trainer card with a tag",
			detail: cardDetail{
				Name:        "Ultra Ball",
				Category:    categoryTrainer,
				Tag:         "Item",
				RarityCode:  "C",
				LocalID:     "150",
				Illustrator: "someone",
				Regulation:  "Luas",
				Attributes:  map[string]any{},
			},
			want: entity.Card{
				ExpansionSetID: expansionSetID,
				LocalID:        "150",
				Name:           "Ultra Ball",
				Category:       categoryTrainer,
				Illustrator:    "someone",
				Tags:           datatypes.JSONSlice[string]{"Item"},
				RarityID:       rarityID,
				ImageURL:       "https://asia.pokemon-card.com/id/card-img/id00016488.png",
				Attributes:     datatypes.JSONMap{"regulation": "Luas"},
				Raw:            string(raw),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapCard(tt.detail, expansionSetID, rarityID, "https://asia.pokemon-card.com/id/card-img/id00016488.png", raw)
			assert.Equal(t, tt.want, got)
		})
	}
}
