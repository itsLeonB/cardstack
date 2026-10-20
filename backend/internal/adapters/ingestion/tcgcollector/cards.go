package tcgcollector

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"gorm.io/datatypes"
)

const (
	categoryPokemon = "Pokémon"
	categoryTrainer = "Trainer"
	categoryEnergi  = "Energi"

	// defaultNumberWidth pads new local IDs in a set that has no numeric ID yet.
	defaultNumberWidth = 3
	// cardBatch is how many card pages pass between progress lines.
	cardBatch = 25

	// tanpaTanda is the Rarity named by the card page's "—", the same name the
	// admin default uses (service.DefaultRarityName).
	tanpaTanda = "Tanpa Tanda"
	noMark     = "—"

	victorySymbol = "Victory Symbol"
)

// tagByType maps a card page's type labels to Card tags. Labels not listed
// here are dropped.
var tagByType = map[string]string{
	"Item":           "Item",
	"Supporter":      "Supporter",
	"Stadium":        "Stadium",
	"Pokémon Tool":   "Pokémon Tool",
	"Basic Energy":   "Energi Dasar",
	"Special Energy": "Energi Spesial",
}

// SetCards is what the card ingestion did for one listed set. Added holds
// what a dry run would add, or what an apply did add.
type SetCards struct {
	Code string
	Name string
	// Published is whether the set was published before this run. A card
	// added to it shows at once.
	Published    bool
	Added        []string
	Skipped      []string
	Failed       []string
	Held         int
	WithoutImage int
}

// cardListing is one card on a set's card list page.
type cardListing struct {
	TCGCollectorID string
	Name           string
	// Number is as printed: "119/126", "001/SM-P" or "No. 201".
	Number   string
	PageURL  string
	ImageURL string
}

// cardPage is what a card's own page adds to its list entry.
type cardPage struct {
	Category       string
	Tags           []string
	Rarity         string
	Illustrator    string
	RegulationMark string
	Raw            string
}

// rarityLookup finds the Game's Rarities by code, and the Tanpa Tanda one for
// a card page's "—".
type rarityLookup struct {
	codes      map[string]uuid.UUID
	tanpaTanda uuid.UUID
}

// parseCardList reads a set's card list page. A printed number keeps its first
// card, so a variant listed under the same number is dropped by the caller.
func parseCardList(page []byte) ([]cardListing, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(page))
	if err != nil {
		return nil, ungerr.Wrap(err, "parsing card list page")
	}

	var cards []cardListing
	var parseErr error
	doc.Find("div.card-image-grid-item").EachWithBreak(func(_ int, item *goquery.Selection) bool {
		card, err := parseCardItem(item)
		if err != nil {
			parseErr = err
			return false
		}
		cards = append(cards, card)
		return true
	})
	if parseErr != nil {
		return nil, parseErr
	}
	if len(cards) == 0 {
		return nil, ungerr.Unknownf("no cards found on the card list page")
	}

	return cards, nil
}

func parseCardItem(item *goquery.Selection) (cardListing, error) {
	id, ok := item.Attr("data-card-id")
	if !ok || id == "" {
		return cardListing{}, ungerr.Unknownf("card entry has no data-card-id")
	}

	link := item.Find("a.card-image-grid-item-link").First()
	href, ok := link.Attr("href")
	if !ok || href == "" {
		return cardListing{}, ungerr.Unknownf("card %s has no page link", id)
	}

	number := strings.TrimSpace(item.Find("div.card-image-grid-item-info-overlay-number").First().Text())
	if number == "" {
		return cardListing{}, ungerr.Unknownf("card %s has no number", id)
	}

	return cardListing{
		TCGCollectorID: id,
		Name:           cardName(link.AttrOr("title", "")),
		Number:         number,
		PageURL:        tcgCollectorOrigin + href,
		ImageURL:       largestSrcset(item.Find("img.card-image-grid-item-image").AttrOr("srcset", "")),
	}, nil
}

// cardName drops the set and number a card's title ends with:
// "Oddish (Shiny Treasure ex 001/190)" is "Oddish".
func cardName(title string) string {
	if i := strings.LastIndex(title, " ("); i >= 0 {
		return title[:i]
	}
	return title
}

// cardNumber reduces a printed number to its integer: "119/126" and "001/SM-P"
// give 119 and 1, "No. 201" gives 201. ok is false when no digits remain.
func cardNumber(printed string) (int, bool) {
	head, _, _ := strings.Cut(printed, "/")
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, head)
	return numericID(digits)
}

// textCardID returns a printed number whose text before the "/" has no digits,
// such as "R/RGB", whole and trimmed as its local ID. The Asia ingester cuts it
// at the "/" instead; see ADR-0021.
func textCardID(printed string) (string, bool) {
	head, _, found := strings.Cut(printed, "/")
	if !found || strings.TrimSpace(head) == "" || strings.ContainsAny(head, "0123456789") {
		return "", false
	}
	return strings.TrimSpace(printed), true
}

// numericID reads a local ID as an integer. ok is false for an empty ID or one
// with anything but digits, such as "SM-P".
func numericID(id string) (int, bool) {
	if id == "" || strings.Trim(id, "0123456789") != "" {
		return 0, false
	}
	n, err := strconv.Atoi(id)
	return n, err == nil
}

// numberWidth is the zero-padded width new local IDs take in a set: the
// longest numeric ID it holds, or defaultNumberWidth when it holds none.
func numberWidth(cards []entity.Card) int {
	width := 0
	for _, card := range cards {
		if _, ok := numericID(card.LocalID); ok && len(card.LocalID) > width {
			width = len(card.LocalID)
		}
	}
	if width == 0 {
		return defaultNumberWidth
	}
	return width
}

// isBasicEnergy matches by name, as the Asia ingester's Basic Energy (type code
// FIR) cannot be matched by number.
func isBasicEnergy(name string) bool {
	return strings.HasPrefix(name, "Basic ") && strings.HasSuffix(name, " Energy")
}

// parseCardPage reads a card's own page: rarity, illustrator, regulation mark,
// category, tags and the <main> HTML the card keeps as raw.
func parseCardPage(page []byte) (cardPage, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(page))
	if err != nil {
		return cardPage{}, ungerr.Wrap(err, "parsing card page")
	}

	rarity := footerText(doc, "Rarity")
	if rarity == "" {
		return cardPage{}, ungerr.Unknownf("card page has no rarity")
	}
	code := noMark
	if rarity != noMark {
		code = rarityCode(rarity)
	}

	illustrator := footerText(doc, "Illustrators")
	if illustrator == noMark {
		illustrator = ""
	}
	regulationMark := footerText(doc, "Regulation mark")
	if regulationMark == noMark {
		regulationMark = ""
	}

	var types []string
	doc.Find("#card-type-containers .card-type-container").Each(func(_ int, s *goquery.Selection) {
		types = append(types, strings.Trim(strings.TrimSpace(s.Text()), ", "))
	})
	category, tags, err := categoryAndTags(doc, types)
	if err != nil {
		return cardPage{}, err
	}

	main := doc.Find("main").First()
	if main.Length() == 0 {
		return cardPage{}, ungerr.Unknownf("card page has no main element")
	}
	raw, err := rawMain(main)
	if err != nil {
		return cardPage{}, err
	}

	return cardPage{
		Category:       category,
		Tags:           tags,
		Rarity:         code,
		Illustrator:    illustrator,
		RegulationMark: regulationMark,
		Raw:            raw,
	}, nil
}

// footerText is the text of the footer row titled title, with its whitespace
// collapsed. It is empty when the page has no such row.
func footerText(doc *goquery.Document, title string) string {
	var text string
	doc.Find("div.card-info-footer-item").EachWithBreak(func(_ int, item *goquery.Selection) bool {
		if strings.TrimSpace(item.Find("div.card-info-footer-item-title").First().Text()) != title {
			return true
		}
		text = strings.Join(strings.Fields(item.Find(".card-info-footer-item-text").First().Text()), " ")
		return false
	})
	return text
}

// rarityCode reads "Double Rare (RR)" as RR. Text with no parentheses is
// returned whole.
func rarityCode(text string) string {
	open := strings.LastIndex(text, "(")
	if open < 0 || !strings.HasSuffix(text, ")") {
		return text
	}
	return text[open+1 : len(text)-1]
}

// categoryAndTags reads a card's category from its HP and type labels. A card
// that shows HP is a Pokémon; otherwise it is an Energi when a type label ends
// in "Energy", and a Trainer when not.
func categoryAndTags(doc *goquery.Document, types []string) (string, []string, error) {
	if len(types) == 0 {
		return "", nil, ungerr.Unknownf("card page has no type label")
	}

	tags := []string{}
	energy := false
	for _, label := range types {
		if tag, ok := tagByType[label]; ok {
			tags = append(tags, tag)
		}
		energy = energy || strings.HasSuffix(label, "Energy")
	}

	switch {
	case doc.Find("#card-hit-points-container").Length() > 0:
		return categoryPokemon, tags, nil
	case energy:
		return categoryEnergi, tags, nil
	default:
		return categoryTrainer, tags, nil
	}
}

// rawMain is the page's <main> as the card stores it: scripts, styles and SVG
// removed, and runs of whitespace collapsed.
func rawMain(main *goquery.Selection) (string, error) {
	main.Find("script, style, svg").Remove()
	html, err := goquery.OuterHtml(main)
	if err != nil {
		return "", ungerr.Wrap(err, "rendering card page")
	}
	return strings.Join(strings.Fields(html), " "), nil
}

// loadRarities reads the Game's Rarities by code.
func (in *Ingester) loadRarities(ctx context.Context, gameID uuid.UUID) (rarityLookup, error) {
	rows, err := in.rarities.FindAll(ctx, crud.Specification[entity.Rarity]{
		Model: entity.Rarity{GameID: gameID},
	})
	if err != nil {
		return rarityLookup{}, err
	}

	lookup := rarityLookup{codes: map[string]uuid.UUID{}}
	for _, rarity := range rows {
		lookup.codes[rarity.Code] = rarity.ID
		if rarity.Name == tanpaTanda {
			lookup.tanpaTanda = rarity.ID
		}
	}
	return lookup, nil
}

// resolve returns the Rarity of a card page's code, or an error for one that
// is not in the lookup. Creating it is rarityFor's job.
func (r rarityLookup) resolve(code string) (uuid.UUID, error) {
	if code == noMark {
		if r.tanpaTanda == uuid.Nil {
			return uuid.Nil, ungerr.Unknownf("rarity %q is not in rarities", tanpaTanda)
		}
		return r.tanpaTanda, nil
	}

	id, ok := r.codes[code]
	if !ok {
		return uuid.Nil, ungerr.Unknownf("rarity %s is not in rarities", code)
	}
	return id, nil
}

// rarityFor returns the Rarity of a card page's code, creating it on an apply
// run when the Game has none, named by its code as the Asia ingester does. A
// dry run creates nothing and returns uuid.Nil for a new code: that Card is
// only reported, never inserted. The lookup's map is shared by every set of
// the run, so a code is created once.
func (in *Ingester) rarityFor(ctx context.Context, run setRun, code string) (uuid.UUID, error) {
	if _, known := run.rarities.codes[code]; known || code == noMark {
		return run.rarities.resolve(code)
	}
	if !run.apply {
		return uuid.Nil, nil
	}

	created, err := in.rarities.Insert(ctx, entity.Rarity{GameID: run.gameID, Code: code, Name: code})
	if err != nil {
		return uuid.Nil, err
	}
	run.rarities.codes[code] = created.ID
	return created.ID, nil
}

// setRun is what the sets of one run share: whether to write, the Game and
// Locale a new set is created under, and the lookups its Cards and Series
// resolve against.
type setRun struct {
	apply     bool
	gameID    uuid.UUID
	localeID  uuid.UUID
	rarities  rarityLookup
	seriesIDs map[string]*uuid.UUID
}

// setCards reads one listed set's card list and works out which Cards the
// catalog lacks. set is the catalog row, valid only when matched. A set is new
// when it is not in the catalog or TCG Collector created it: its Basic Energy
// and Victory Symbol are ingested as they are, because a first run that
// stopped partway leaves such a set without them. With apply the Cards are
// written in one transaction, together with the set when it is created. A
// failed write returns its error and leaves that set's rows unwritten.
func (in *Ingester) setCards(
	ctx context.Context, run setRun, listing SetListing, set entity.ExpansionSet, matched bool,
) (SetCards, error) {
	newSet := !matched || set.CreatedBy == entity.SourceTCGCollector
	result := SetCards{Code: listing.Code, Name: listing.Name, Published: matched && set.Published}

	// A card list that cannot be read fails the set's Cards, not the set: a new
	// set is still created, as it is for any run.
	var listed []cardListing
	page, err := in.pages.Fetch(ctx, listing.CardListURL)
	if err == nil {
		listed, err = parseCardList(page)
	}
	if err != nil {
		result.Failed = append(result.Failed, fmt.Sprintf("card list: %v", err))
	}

	var existing []entity.Card
	if matched {
		existing, err = in.cards.FindAll(ctx, crud.Specification[entity.Card]{
			Model: entity.Card{ExpansionSetID: set.ID},
		})
		if err != nil {
			return SetCards{}, err
		}
	}

	held := map[int]bool{}
	localIDs := map[string]bool{}
	for _, card := range existing {
		localIDs[card.LocalID] = true
		if n, ok := numericID(card.LocalID); ok {
			held[n] = true
		}
	}
	width := numberWidth(existing)

	type wanted struct {
		listing cardListing
		localID string
	}
	var todo []wanted
	seenText := map[string]bool{}
	for _, card := range listed {
		number, ok := cardNumber(card.Number)
		text, isText := textCardID(card.Number)
		if !ok && !isText {
			result.Failed = append(result.Failed, fmt.Sprintf("%s: number %q has no digits", card.Name, card.Number))
			continue
		}
		// The skips come before the held check, so a Card skipped for its kind
		// counts as Skipped even when its number is held.
		if !newSet && isBasicEnergy(card.Name) {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s: basic energy in a set that exists", card.Name))
			continue
		}
		if !newSet && card.Name == victorySymbol && localIDs[listing.Code] {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s: the set already has a card with ID %s", card.Name, listing.Code))
			continue
		}
		// The integer held map cannot hold a text ID, so it is matched against the
		// set's local IDs instead.
		if isText {
			if seenText[text] {
				result.Failed = append(result.Failed, fmt.Sprintf("%s: number %q is listed more than once in this card list", card.Name, card.Number))
				continue
			}
			seenText[text] = true
			if localIDs[text] {
				result.Held++
				continue
			}
			todo = append(todo, wanted{listing: card, localID: text})
			continue
		}
		if held[number] {
			result.Held++
			continue
		}
		held[number] = true

		todo = append(todo, wanted{listing: card, localID: fmt.Sprintf("%0*d", width, number)})
	}

	type pendingCard struct {
		label string
		card  entity.Card
	}
	var pending []pendingCard
	for i, item := range todo {
		if i > 0 && i%cardBatch == 0 {
			logger.Infof("%s processed %d/%d cards", listing.Code, i, len(todo))
		}

		label := fmt.Sprintf("%s %s", item.localID, item.listing.Name)
		card, err := in.newCard(ctx, item.localID, item.listing, run)
		if err != nil {
			result.Failed = append(result.Failed, fmt.Sprintf("%s: %v", label, err))
			continue
		}
		pending = append(pending, pendingCard{label: label, card: card})
	}

	for _, p := range pending {
		result.Added = append(result.Added, p.label)
		if p.card.SourceImageURL == "" {
			result.WithoutImage++
		}
	}
	if !run.apply {
		return result, nil
	}

	err = in.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		setID := set.ID
		if !matched {
			created, err := in.insertSet(ctx, run, listing)
			if err != nil {
				return err
			}
			setID = created.ID
		}

		for _, p := range pending {
			p.card.ExpansionSetID = setID
			if _, err := in.cards.Insert(ctx, p.card); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		result.Added, result.WithoutImage = nil, 0
		return result, err
	}

	return result, nil
}

// newCard fetches a listed card's page and builds the Card to insert. Errors
// here are the card's own and only fail that card.
func (in *Ingester) newCard(
	ctx context.Context, localID string, listing cardListing, run setRun,
) (entity.Card, error) {
	page, err := in.pages.Fetch(ctx, listing.PageURL)
	if err != nil {
		return entity.Card{}, err
	}
	detail, err := parseCardPage(page)
	if err != nil {
		return entity.Card{}, err
	}
	rarityID, err := in.rarityFor(ctx, run, detail.Rarity)
	if err != nil {
		return entity.Card{}, err
	}

	attributes := datatypes.JSONMap{}
	if detail.RegulationMark != "" {
		attributes["regulationMark"] = detail.RegulationMark
	}

	return entity.Card{
		LocalID:        localID,
		Name:           listing.Name,
		Category:       detail.Category,
		Illustrator:    detail.Illustrator,
		Tags:           detail.Tags,
		RarityID:       rarityID,
		SourceImageURL: listing.ImageURL,
		Attributes:     attributes,
		Raw:            detail.Raw,
		CreatedBy:      entity.SourceTCGCollector,
		UpdatedBy:      entity.SourceTCGCollector,
	}, nil
}
