package tcgcollector

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"gorm.io/gorm"
)

// gameSlug is the Game the Pokémon Asia ingester writes, which the TCG
// Collector rows share.
const gameSlug = "pokemon-tcg"

// UnpublishedSetLister lists a Game's Expansion Sets that are not published.
type UnpublishedSetLister interface {
	ListUnpublished(ctx context.Context, gameID uuid.UUID) ([]entity.ExpansionSet, error)
}

// noUnpublished stands in until the published column exists, so no set is
// unpublished yet.
type noUnpublished struct{}

func (noUnpublished) ListUnpublished(context.Context, uuid.UUID) ([]entity.ExpansionSet, error) {
	return nil, nil
}

// Ingester classifies TCG Collector's sets list against the catalog. It
// writes nothing yet.
type Ingester struct {
	games       crud.Repository[entity.Game]
	sets        crud.Repository[entity.ExpansionSet]
	cards       repository.CardRepository
	pages       PageSource
	unpublished UnpublishedSetLister
	gameSlug    string
}

func NewIngester(db *gorm.DB, pages PageSource) *Ingester {
	return &Ingester{
		games:       crud.NewRepository[entity.Game](db),
		sets:        crud.NewRepository[entity.ExpansionSet](db),
		cards:       repository.NewCardRepository(crud.NewRepository[entity.Card](db)),
		pages:       pages,
		unpublished: noUnpublished{},
		gameSlug:    gameSlug,
	}
}

// Options narrows a run.
type Options struct {
	// SetCode limits the report to one listed set, by its catalog code. ""
	// means every set.
	SetCode string
}

// Report is what a run found, grouped the way the operator reads it.
type Report struct {
	// Matched are the listed sets the catalog already holds.
	Matched []SetListing
	// New are the listed sets the catalog lacks.
	New []SetListing
	// Unpublished are the catalog's unpublished sets, with what they need checked.
	Unpublished []UnpublishedSet
}

// UnpublishedSet is one unpublished Expansion Set in the catalog.
type UnpublishedSet struct {
	Code              string
	Name              string
	CardCount         int
	CardsWithoutImage int
	SeriesUnknown     bool
}

// Run reads the sets list and classifies every listed set against the catalog.
func (in *Ingester) Run(ctx context.Context, opts Options) (Report, error) {
	page, err := in.pages.Fetch(ctx, setsPageURL)
	if err != nil {
		return Report{}, err
	}

	listings, err := parseSets(page)
	if err != nil {
		return Report{}, err
	}

	if opts.SetCode != "" {
		// The flag takes either code, so SMP and SM-P both select the set.
		listings = filterByCode(listings, normalizeSetCode(opts.SetCode))
		if len(listings) == 0 {
			return Report{}, ungerr.Unknownf("set %s is not on the sets page", opts.SetCode)
		}
	}

	game, err := in.games.FindFirst(ctx, crud.Specification[entity.Game]{
		Model: entity.Game{Slug: in.gameSlug},
	})
	if err != nil {
		return Report{}, err
	}

	// A missing Game means no Expansion Set exists yet. The game checks below
	// matter: a zero GameID is dropped from a query's WHERE clause, which would
	// list the sets of every game.
	codes := map[string]bool{}
	if !game.IsZero() {
		existing, err := in.sets.FindAll(ctx, crud.Specification[entity.ExpansionSet]{
			Model: entity.ExpansionSet{GameID: game.ID},
		})
		if err != nil {
			return Report{}, err
		}
		for _, set := range existing {
			codes[set.Code] = true
		}
	}

	var report Report
	for i, listing := range listings {
		if codes[listing.Code] {
			report.Matched = append(report.Matched, listing)
			logger.Infof("[%d/%d] %s matched", i+1, len(listings), listing.Code)
			continue
		}
		report.New = append(report.New, listing)
		logger.Infof("[%d/%d] %s new", i+1, len(listings), listing.Code)
	}

	if !game.IsZero() {
		unpublished, err := in.unpublished.ListUnpublished(ctx, game.ID)
		if err != nil {
			return Report{}, err
		}
		counts, err := in.cards.CountsByExpansionSet(ctx, game.ID)
		if err != nil {
			return Report{}, err
		}
		for _, set := range unpublished {
			count := counts[set.ID]
			report.Unpublished = append(report.Unpublished, UnpublishedSet{
				Code:              set.Code,
				Name:              set.Name,
				CardCount:         count.Total,
				CardsWithoutImage: count.WithoutImage,
				SeriesUnknown:     set.SeriesID == nil,
			})
		}
	}

	logger.Infof(
		"checked %d sets: %d matched, %d new, %d unpublished",
		len(listings), len(report.Matched), len(report.New), len(report.Unpublished),
	)
	return report, nil
}

func filterByCode(listings []SetListing, code string) []SetListing {
	var out []SetListing
	for _, listing := range listings {
		if listing.Code == code {
			out = append(out, listing)
		}
	}
	return out
}

// Render writes the report as plain text, one section per group.
func (r Report) Render(w io.Writer) error {
	var b strings.Builder

	fmt.Fprintf(&b, "Matched sets already in the catalog: %d\n", len(r.Matched))
	for _, set := range r.Matched {
		fmt.Fprintf(&b, "  %-10s %s\n", set.Code, set.Name)
	}

	fmt.Fprintf(&b, "\nNew sets, created unpublished with -apply: %d\n", len(r.New))
	for _, set := range r.New {
		series := set.Series
		if series == "" {
			series = fmt.Sprintf("UNKNOWN (heading %q)", set.SeriesHeading)
		}
		fmt.Fprintf(&b, "  %-10s %s | series: %s | cards: %d\n", set.Code, set.Name, series, set.CardCount)
	}

	fmt.Fprintf(&b, "\nUnpublished sets in the catalog: %d\n", len(r.Unpublished))
	for _, set := range r.Unpublished {
		fmt.Fprintf(&b, "  %-10s %s | cards: %d | without image: %d", set.Code, set.Name, set.CardCount, set.CardsWithoutImage)
		if set.SeriesUnknown {
			b.WriteString(" | series: UNKNOWN")
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "    UPDATE expansion_sets SET published = true WHERE code = '%s';\n", set.Code)
	}

	_, err := io.WriteString(w, b.String())
	if err != nil {
		return ungerr.Wrap(err, "writing report")
	}
	return nil
}
