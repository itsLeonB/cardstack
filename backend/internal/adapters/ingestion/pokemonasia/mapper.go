package pokemonasia

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"gorm.io/datatypes"
)

// The three category values the site's own card-type structure maps onto
// (docs/adr/0007's closed vocabulary), stored verbatim as the site's label
// text.
const (
	categoryPokemon = "Pokémon"
	categoryTrainer = "Trainer"
	categoryEnergi  = "Energi"
)

// releaseDateLayout matches the site's <time datetime="MM-DD-YYYY"> format.
const releaseDateLayout = "01-02-2006"

var detailIDPattern = regexp.MustCompile(`/card-search/detail/(\d+)/`)

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// specialMarkers lists known Pokémon-card name-suffix markers, longest/most
// specific first so a compound marker (e.g. "V-UNION") is matched before a
// shorter one it contains (e.g. "V") produces a false positive.
//
// ponytail: this is a name-suffix heuristic against a small known
// vocabulary, not a structured field (the site doesn't expose one — see the
// plan's "Card Tag" decision). Upgrade to per-marker query-bucketing if a
// filter on this field is ever needed.
var specialMarkers = []string{
	"Evolusi Mega ex",
	"V-UNION",
	"VMAX",
	"VSTAR",
	"Bercahaya",
	"GX",
	"EX",
	"ex",
	"V",
}

// parseExpansionListings extracts every Series/Expansion Set/release-date
// tuple from one page of GET /card-search/?pageNo=N (ul.expansionList). A
// listing with no expansionCodes query param or an unparseable release date
// is skipped rather than returned half-populated.
func parseExpansionListings(doc *goquery.Document) []expansionListing {
	var out []expansionListing
	doc.Find("ul.expansionList li.expansion").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Find("a.expansionLink").Attr("href")
		code := hrefQueryParam(href, "expansionCodes")
		if code == "" {
			return
		}

		dateStr, _ := s.Find("time.relaseDate").Attr("datetime")
		releaseDate, err := time.Parse(releaseDateLayout, strings.TrimSpace(dateStr))
		if err != nil {
			return
		}

		out = append(out, expansionListing{
			Series:      strings.TrimSpace(s.Find("span.series").Text()),
			Code:        code,
			Name:        strings.TrimSpace(s.Find("h3.expansionTitle").Text()),
			ReleaseDate: releaseDate,
		})
	})
	return out
}

func hrefQueryParam(href, key string) string {
	i := strings.IndexByte(href, '?')
	if i < 0 {
		return ""
	}
	values, err := url.ParseQuery(href[i+1:])
	if err != nil {
		return ""
	}
	return values.Get(key)
}

// parseResultCardIDs extracts every detail-page numeric id linked from one
// page of GET /card-search/list/?... (ul.list > li.card > a).
func parseResultCardIDs(doc *goquery.Document) []string {
	var ids []string
	doc.Find("ul.list li.card a").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		m := detailIDPattern.FindStringSubmatch(href)
		if m == nil {
			return
		}
		ids = append(ids, m[1])
	})
	return ids
}

// detectCategory implements the plan's category rule: an h1.pageHeader with
// a span.evolveMarker is a Pokémon card; otherwise the h3.commonHeader text
// (the card's own structural type label) is Energi (Energi Dasar/Spesial)
// or Trainer, and that same text is also the card's Tag. Pokémon cards get
// no Tag — see the plan's "Card Tag" decision.
func detectCategory(doc *goquery.Document) (category, tag string) {
	if doc.Find("h1.pageHeader span.evolveMarker").Length() > 0 {
		return categoryPokemon, ""
	}
	header := strings.TrimSpace(doc.Find("h3.commonHeader").First().Text())
	switch header {
	case "Energi Dasar", "Energi Spesial":
		return categoryEnergi, header
	default:
		return categoryTrainer, header
	}
}

// detectSpecialMarkers matches the known Pokémon special-marker vocabulary
// against the end of a card's display name (e.g. "Mega Venusaur ex" -> "ex")
// — see specialMarkers' doc comment.
func detectSpecialMarkers(name string) []string {
	for _, marker := range specialMarkers {
		if strings.HasSuffix(name, " "+marker) {
			return []string{marker}
		}
	}
	return nil
}

// splitCollectorNumber extracts the numeric portion of a collector number
// (e.g. "001/126" -> "001"). A collector number with no "/" is returned
// unchanged (a non-numeric collector number is possible in general, though
// not expected within the four in-scope Series — see the plan).
func splitCollectorNumber(s string) string {
	if i := strings.IndexByte(s, '/'); i >= 0 {
		return s[:i]
	}
	return s
}

// cardImageURL builds a card's image URL from its detail-page id (not its
// LocalID/collector number), per the site's fixed 8-digit zero-padded
// filename convention.
func cardImageURL(id string) string {
	n, err := strconv.Atoi(id)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("https://asia.pokemon-card.com/id/card-img/id%08d.png", n)
}

// parseCardDetail parses one GET /card-search/detail/{id}/ page into a
// cardDetail. It is pure: no I/O, no DB. Regulation is left zero for the
// caller to fill in (see cardDetail's doc comment).
func parseCardDetail(doc *goquery.Document) cardDetail {
	h1 := doc.Find("h1.pageHeader").First()

	nameOnly := h1.Clone()
	nameOnly.Find("span.evolveMarker").Remove()
	name := strings.TrimSpace(nameOnly.Text())

	category, tag := detectCategory(doc)

	// ponytail: attributes here captures only what's cheaply available from
	// documented selectors (stage, special markers); full Pokémon battle-
	// stat extraction (HP/attacks/weakness/resistance/retreat) isn't in the
	// ticket's must-extract list and the plan doesn't capture its selectors.
	// Add it if a consumer needs it.
	attributes := map[string]any{}
	if stage := strings.TrimSpace(h1.Find("span.evolveMarker").First().Text()); stage != "" {
		attributes["stage"] = stage
	}
	if markers := detectSpecialMarkers(name); len(markers) > 0 {
		attributes["specialMarkers"] = markers
	}

	rarityCode := strings.TrimSpace(doc.Find("section.expansionColumn span.alpha").First().Text())
	collectorNumber := strings.TrimSpace(doc.Find("section.expansionColumn span.collectorNumber").First().Text())
	illustrator := strings.TrimSpace(doc.Find("div.illustrator a").First().Text())

	return cardDetail{
		Name:        name,
		Category:    category,
		Tag:         tag,
		RarityCode:  rarityCode,
		LocalID:     splitCollectorNumber(collectorNumber),
		Illustrator: illustrator,
		Attributes:  attributes,
	}
}

// slugifySeries locally derives a Series code from its display name (e.g.
// "Scarlet & Violet" -> "scarlet-violet") — per ADR-0008, the site doesn't
// provide one.
func slugifySeries(name string) string {
	slug := slugNonAlnum.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(slug, "-")
}

// mapCard converts a parsed cardDetail plus its already-resolved
// ExpansionSetID/RarityID/ImageURL/raw bytes into the entity.Card row to
// upsert. It is pure: no I/O, no DB. ID resolution (RarityID) and image URL
// construction happen in ingest.go, since they need the client/DB.
func mapCard(detail cardDetail, expansionSetID, rarityID uuid.UUID, imageURL string, raw []byte) entity.Card {
	// tags must marshal to a JSON array, never JSON null (json.Marshal(nil
	// slice) produces "null", which the NOT NULL tags column would happily
	// accept as a valid jsonb value but breaks the GIN index's array
	// assumption) — so a card with no tag gets an explicit empty slice.
	tags := datatypes.JSONSlice[string]{}
	if detail.Tag != "" {
		tags = append(tags, detail.Tag)
	}

	attributes := detail.Attributes
	if attributes == nil {
		attributes = map[string]any{}
	}
	if detail.Regulation != "" {
		attributes["regulation"] = detail.Regulation
	}

	return entity.Card{
		ExpansionSetID: expansionSetID,
		LocalID:        detail.LocalID,
		Name:           detail.Name,
		Category:       detail.Category,
		Illustrator:    detail.Illustrator,
		Tags:           tags,
		RarityID:       rarityID,
		ImageURL:       imageURL,
		Attributes:     datatypes.JSONMap(attributes),
		Raw:            string(raw),
	}
}
