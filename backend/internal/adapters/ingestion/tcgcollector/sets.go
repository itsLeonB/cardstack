package tcgcollector

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/itsLeonB/ungerr"
)

// setsPageURL is TCG Collector's Indonesian sets list. DirSource finds a saved
// copy of it by this URL's path.
const setsPageURL = "https://www.tcgcollector.com/sets/id"

// releaseDateLayout matches the list's "Sep 16, 2026" release dates.
const releaseDateLayout = "Jan 2, 2006"

// seriesByHeading maps the four Series headings on the sets list to the
// Series names the Pokémon Asia ingester writes. A heading missing here
// leaves the set's Series empty and is reported.
var seriesByHeading = map[string]string{
	"Mega Evolution Series":   "Evolusi Mega",
	"Scarlet & Violet Series": "Scarlet & Violet",
	"Sword & Shield Series":   "Pedang & Perisai",
	"Sun & Moon Series":       "Matahari & Bulan",
}

// codeAliases maps TCG Collector set codes that differ from the catalog's
// codes, so a set the catalog already holds is matched rather than created
// again.
var codeAliases = map[string]string{
	"MP":    "M-P",
	"SVP":   "SV-P",
	"SMP":   "SM-P",
	"SP":    "S-P",
	"AS4Di": "AS4D",
	"AS3Di": "AS3D",
	"AS2Di": "AS2D",
}

// SetListing is one Expansion Set on TCG Collector's sets list.
type SetListing struct {
	TCGCollectorID string
	// Code is the catalog code (aliases applied); TCGCode is as printed.
	Code    string
	TCGCode string
	Name    string
	// ReleaseDate is the list's date, the same one the set page shows.
	ReleaseDate time.Time
	// CardCount is the total in the progress label ("0/207" gives 207).
	CardCount int
	// LogoURL is the widest srcset entry, "" when the set has no logo.
	LogoURL string
	// SeriesHeading is the heading the set sits under, verbatim. Series is the
	// catalog name for it, "" when the heading is not in seriesByHeading.
	SeriesHeading string
	Series        string
}

// parseSets reads every set under the sets list's Series headings, in page
// order. A page with no sets is an error, so a challenge page or a wrong file
// fails loudly instead of reporting every set as new.
func parseSets(page []byte) ([]SetListing, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(page))
	if err != nil {
		return nil, ungerr.Wrap(err, "parsing sets page")
	}

	var sets []SetListing
	doc.Find("div.set-logo-grid").EachWithBreak(func(_ int, group *goquery.Selection) bool {
		heading := strings.TrimSpace(group.Find("h2.set-logo-grid-title").First().Text())
		group.Find("div.set-logo-grid-item").EachWithBreak(func(_ int, item *goquery.Selection) bool {
			listing, itemErr := parseSetItem(item)
			if itemErr != nil {
				err = itemErr
				return false
			}
			listing.SeriesHeading = heading
			listing.Series = seriesByHeading[heading]
			sets = append(sets, listing)
			return true
		})
		return err == nil
	})
	if err != nil {
		return nil, err
	}
	if len(sets) == 0 {
		return nil, ungerr.Unknownf("no sets found on the sets page")
	}

	return sets, nil
}

// parseSetItem reads one set's entry. Its code and release date are required,
// because a set without them cannot be matched or created.
func parseSetItem(item *goquery.Selection) (SetListing, error) {
	id, ok := item.Attr("data-set-id")
	if !ok || id == "" {
		return SetListing{}, ungerr.Unknownf("set entry has no data-set-id")
	}

	tcgCode := strings.TrimSpace(item.Find("span.set-logo-grid-item-code").First().Text())
	if tcgCode == "" {
		return SetListing{}, ungerr.Unknownf("set %s has no code", id)
	}

	released := strings.TrimSpace(item.Find("div.set-logo-grid-item-release-date").First().Text())
	releaseDate, err := time.Parse(releaseDateLayout, released)
	if err != nil {
		return SetListing{}, ungerr.Wrapf(err, "parsing release date of set %s", tcgCode)
	}

	_, total, _ := strings.Cut(strings.TrimSpace(item.Find("div.progress-label").First().Text()), "/")
	cardCount, err := strconv.Atoi(total)
	if err != nil {
		return SetListing{}, ungerr.Wrapf(err, "parsing card count of set %s", tcgCode)
	}

	return SetListing{
		TCGCollectorID: id,
		Code:           normalizeSetCode(tcgCode),
		TCGCode:        tcgCode,
		Name:           strings.TrimSpace(item.Find("a.set-logo-grid-item-name").First().Text()),
		ReleaseDate:    releaseDate,
		CardCount:      cardCount,
		LogoURL:        largestSrcset(item.Find("img.set-logo-grid-item-logo").AttrOr("srcset", "")),
	}, nil
}

// normalizeSetCode returns the catalog code for a TCG Collector set code.
func normalizeSetCode(code string) string {
	if alias, ok := codeAliases[code]; ok {
		return alias
	}
	return code
}

// largestSrcset returns the URL of the widest entry in a srcset such as
// "a.webp 136w, b.webp 272w", or "" when there is none.
func largestSrcset(srcset string) string {
	best, bestWidth := "", -1
	for entry := range strings.SplitSeq(srcset, ",") {
		fields := strings.Fields(entry)
		if len(fields) != 2 {
			continue
		}
		width, err := strconv.Atoi(strings.TrimSuffix(fields[1], "w"))
		if err != nil || width <= bestWidth {
			continue
		}
		best, bestWidth = fields[0], width
	}
	return best
}
