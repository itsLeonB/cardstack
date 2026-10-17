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

// localeCode is the Indonesian locale TCG Collector's sets are written in, the
// same locale the Pokémon Asia ingester writes.
const localeCode = "id"

// Ingester compares TCG Collector's sets list with the catalog, and with
// Options.Apply creates the sets the catalog lacks and fills the missing logos.
type Ingester struct {
	games      crud.Repository[entity.Game]
	sets       repository.ExpansionSetRepository
	cards      repository.CardRepository
	locales    crud.Repository[entity.Locale]
	series     crud.Repository[entity.Series]
	transactor crud.Transactor
	rarities   crud.Repository[entity.Rarity]
	pages      PageSource
	gameSlug   string
}

func NewIngester(db *gorm.DB, pages PageSource) *Ingester {
	return &Ingester{
		games:      crud.NewRepository[entity.Game](db),
		sets:       repository.NewExpansionSetRepository(crud.NewRepository[entity.ExpansionSet](db)),
		cards:      repository.NewCardRepository(crud.NewRepository[entity.Card](db)),
		locales:    crud.NewRepository[entity.Locale](db),
		series:     crud.NewRepository[entity.Series](db),
		rarities:   crud.NewRepository[entity.Rarity](db),
		transactor: crud.NewTransactor(db),
		pages:      pages,
		gameSlug:   gameSlug,
	}
}

// loadSets reads the Game's Expansion Sets by code.
func (in *Ingester) loadSets(ctx context.Context, gameID uuid.UUID) (map[string]entity.ExpansionSet, error) {
	sets, err := in.sets.FindAll(ctx, crud.Specification[entity.ExpansionSet]{
		Model: entity.ExpansionSet{GameID: gameID},
	})
	if err != nil {
		return nil, err
	}

	byCode := make(map[string]entity.ExpansionSet, len(sets))
	for _, set := range sets {
		byCode[set.Code] = set
	}
	return byCode, nil
}

// Options narrows a run.
type Options struct {
	// SetCode limits the run to one listed set, by its catalog code. "" means
	// every set.
	SetCode string
	// Apply creates the missing sets and fills the missing logos. Without it the
	// run only reports.
	Apply bool
	// Refresh refetches the pages this run reads instead of using the page
	// cache. With SetCode that is the sets page and that set's pages.
	Refresh bool
}

// Report is what a run found, grouped the way the operator reads it.
type Report struct {
	// Matched are the listed sets the catalog already holds.
	Matched []SetListing
	// New are the listed sets the catalog lacked before this run.
	New []SetListing
	// Unpublished are the catalog's unpublished sets, with what they need checked.
	// Sets this run created are among them.
	Unpublished []UnpublishedSet
	// Failed are the sets a -apply write could not finish, with the cause. The
	// other sets are still written.
	Failed []string
	// Cards are the card ingestion's results for each listed set in scope.
	Cards []SetCards
}

// UnpublishedSet is one unpublished Expansion Set in the catalog.
type UnpublishedSet struct {
	Code              string
	Name              string
	CardCount         int
	CardsWithoutImage int
	SeriesUnknown     bool
	// Created marks a set this run created, so the operator knows it is new.
	Created bool
}

// Run reads the sets list and classifies every listed set against the catalog.
// With opts.Apply it writes the missing sets and logos first, so the
// unpublished list it reports includes the sets it just created.
func (in *Ingester) Run(ctx context.Context, opts Options) (Report, error) {
	if opts.Refresh {
		ctx = withRefresh(ctx)
	}

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
	if opts.Apply && game.IsZero() {
		return Report{}, ungerr.Unknownf("game %s is not in the catalog: run ingest-pokemon-asia first", in.gameSlug)
	}

	// A missing Game means no Expansion Set exists yet. The game checks below
	// matter: a zero GameID is dropped from a query's WHERE clause, which would
	// list the sets of every game.
	existing := map[string]entity.ExpansionSet{}
	if !game.IsZero() {
		existing, err = in.loadSets(ctx, game.ID)
		if err != nil {
			return Report{}, err
		}
	}

	var report Report
	for i, listing := range listings {
		if _, ok := existing[listing.Code]; ok {
			report.Matched = append(report.Matched, listing)
			logger.Infof("[%d/%d] %s matched", i+1, len(listings), listing.Code)
			continue
		}
		report.New = append(report.New, listing)
		logger.Infof("[%d/%d] %s new", i+1, len(listings), listing.Code)
	}

	var created map[string]bool
	if opts.Apply {
		created, report.Failed, err = in.apply(ctx, game.ID, listings, existing)
		if err != nil {
			return Report{}, err
		}
	}

	if !game.IsZero() {
		// Cards go in before the unpublished counts, which must include them.
		report.Cards, err = in.cardReports(ctx, opts, game.ID, listings, created)
		if err != nil {
			return Report{}, err
		}

		unpublished, err := in.sets.ListUnpublished(ctx, game.ID)
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
				Created:           created[set.Code],
			})
		}
	}

	logger.Infof(
		"checked %d sets: %d matched, %d new, %d unpublished",
		len(listings), len(report.Matched), len(report.New), len(report.Unpublished),
	)
	return report, nil
}

// apply creates each listed set the catalog lacks, unpublished, and fills the
// logo of each matched set that has none. One failed set is logged and
// collected while the others are still written, so the error is only for a
// failure that stops every set.
func (in *Ingester) apply(
	ctx context.Context, gameID uuid.UUID, listings []SetListing, existing map[string]entity.ExpansionSet,
) (map[string]bool, []string, error) {
	locale, err := in.upsertLocale(ctx)
	if err != nil {
		return nil, nil, err
	}

	created := map[string]bool{}
	var failed []string
	logos := 0
	seriesIDs := map[string]*uuid.UUID{}
	for i, listing := range listings {
		progress := fmt.Sprintf("[%d/%d]", i+1, len(listings))
		set, matched := existing[listing.Code]

		if !matched {
			if err := in.createSet(ctx, gameID, locale.ID, seriesIDs, listing); err != nil {
				failure := fmt.Sprintf("%s: %v", listing.Code, err)
				logger.Errorf("%s failed %s", progress, failure)
				failed = append(failed, failure)
				continue
			}
			created[listing.Code] = true
			logger.Infof("%s created %s", progress, listing.Code)
			continue
		}

		if set.SourceLogoURL != "" || listing.LogoURL == "" {
			logger.Infof("%s %s up to date", progress, listing.Code)
			continue
		}
		if err := in.sets.FillSourceLogoURL(ctx, set.ID, listing.LogoURL); err != nil {
			failure := fmt.Sprintf("%s: %v", listing.Code, err)
			logger.Errorf("%s failed %s", progress, failure)
			failed = append(failed, failure)
			continue
		}
		logos++
		logger.Infof("%s %s logo filled", progress, listing.Code)
	}

	logger.Infof("applied: %d created, %d logos filled, %d failed", len(created), logos, len(failed))
	return created, failed, nil
}

// createSet inserts a listed set as unpublished. Its Series is the catalog
// Series of the same name, or none when the catalog has no such Series.
func (in *Ingester) createSet(
	ctx context.Context, gameID, localeID uuid.UUID, seriesIDs map[string]*uuid.UUID, listing SetListing,
) error {
	seriesID, err := in.seriesID(ctx, gameID, listing.Series, seriesIDs)
	if err != nil {
		return err
	}

	releaseDate := listing.ReleaseDate
	return in.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		set, err := in.sets.Insert(ctx, entity.ExpansionSet{
			GameID:        gameID,
			Code:          listing.Code,
			Name:          listing.Name,
			LocaleID:      localeID,
			SeriesID:      seriesID,
			ReleaseDate:   &releaseDate,
			SourceLogoURL: listing.LogoURL,
			CreatedBy:     entity.SourceTCGCollector,
			UpdatedBy:     entity.SourceTCGCollector,
		})
		if err != nil {
			return err
		}

		// published's column default is true, and Create leaves a zero value to
		// that default, so the unpublished state is a second write in the same
		// transaction.
		set.Published = false
		_, err = in.sets.Update(ctx, set)
		return err
	})
}

// seriesID returns the ID of the catalog Series with the given name, or nil
// when the name is empty or the catalog has no such Series. A nil leaves the
// set's Series unknown, which the report shows.
func (in *Ingester) seriesID(
	ctx context.Context, gameID uuid.UUID, name string, seriesIDs map[string]*uuid.UUID,
) (*uuid.UUID, error) {
	if name == "" {
		return nil, nil
	}
	if id, ok := seriesIDs[name]; ok {
		return id, nil
	}

	series, err := in.series.FindFirst(ctx, crud.Specification[entity.Series]{
		Model: entity.Series{GameID: gameID, Name: name},
	})
	if err != nil {
		return nil, err
	}
	if series.IsZero() {
		logger.Warnf("series %q is not in the catalog: its sets stay without a Series", name)
		seriesIDs[name] = nil
		return nil, nil
	}
	seriesIDs[name] = &series.ID
	return &series.ID, nil
}

// upsertLocale find-or-creates the Indonesian Locale row.
func (in *Ingester) upsertLocale(ctx context.Context) (entity.Locale, error) {
	locale, err := in.locales.FindFirst(ctx, crud.Specification[entity.Locale]{
		Model: entity.Locale{Code: localeCode},
	})
	if err != nil {
		return entity.Locale{}, err
	}
	if !locale.IsZero() {
		return locale, nil
	}
	return in.locales.Insert(ctx, entity.Locale{Code: localeCode})
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
		if set.Created {
			b.WriteString(" | created this run")
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "    UPDATE expansion_sets SET published = true WHERE code = '%s';\n", set.Code)
	}

	fmt.Fprintf(&b, "\nCards in the listed sets: %d sets\n", len(r.Cards))
	for _, set := range r.Cards {
		published := 0
		if set.Published {
			published = len(set.Added)
		}
		fmt.Fprintf(
			&b, "  %-10s %s | added: %d | added to a published set: %d | held: %d | skipped: %d | failed: %d | added without image: %d\n",
			set.Code, set.Name, len(set.Added), published, set.Held, len(set.Skipped), len(set.Failed), set.WithoutImage,
		)
		for _, card := range set.Added {
			fmt.Fprintf(&b, "    + %s\n", card)
		}
		for _, skip := range set.Skipped {
			fmt.Fprintf(&b, "    skip %s\n", skip)
		}
		for _, failure := range set.Failed {
			fmt.Fprintf(&b, "    fail %s\n", failure)
		}
	}

	fmt.Fprintf(&b, "\nFailed to write: %d\n", len(r.Failed))
	for _, failure := range r.Failed {
		fmt.Fprintf(&b, "  %s\n", failure)
	}

	_, err := io.WriteString(w, b.String())
	if err != nil {
		return ungerr.Wrap(err, "writing report")
	}
	return nil
}
