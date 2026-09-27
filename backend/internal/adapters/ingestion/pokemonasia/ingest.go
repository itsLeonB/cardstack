package pokemonasia

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

const (
	// gameSlug/gameName identify the single Game row this ingester
	// find-or-creates on every run — the same row ticket 03's TCGDex
	// ingester created (see the plan's "Upsert strategy" decision).
	gameSlug = "pokemon-tcg"
	gameName = "Pokémon TCG"

	// localeCode is fixed: this ingester only ever scrapes the Indonesian
	// locale of the site (ticket scope).
	localeCode = "id"

	// maxConcurrency bounds concurrent card-detail fetches. See client.go's
	// requestInterval for the shared rate limit all requests go through
	// regardless of this cap.
	maxConcurrency = 5
)

// targetSeries is the closed set of Series (the site's own span.series
// label text) this ingester covers — see the ticket's "every Indonesian
// print edition across all four Series" scope.
var targetSeries = map[string]bool{
	"Evolusi Mega":     true,
	"Scarlet & Violet": true,
	"Pedang & Perisai": true,
	"Matahari & Bulan": true,
}

// Summary reports row counts from a completed ingestion run.
type Summary struct {
	Series   int
	Sets     int
	Rarities int
	Cards    int
	Failures []IngestFailure
}

// IngestFailure records one item (a card, or a listing/results page) that
// failed after client.get exhausted its own retries. Run collects these
// instead of aborting so one bad card doesn't cost hours of otherwise-good
// progress; the recorded ExpansionCode/CardID is enough to target a later,
// independent retry of just this item instead of rerunning the whole set.
type IngestFailure struct {
	ExpansionCode string
	CardID        string // empty for a listing/results-page failure
	Stage         string
	Err           error
}

func (f IngestFailure) Error() string {
	if f.CardID != "" {
		return fmt.Sprintf("set %s: %s (card %s): %v", f.ExpansionCode, f.Stage, f.CardID, f.Err)
	}
	return fmt.Sprintf("set %s: %s: %v", f.ExpansionCode, f.Stage, f.Err)
}

// Ingester ingests Pokémon Asia catalog data into the games/series/
// expansion_sets/rarities/cards tables. It constructs its own
// crud.Repository instances directly from a *gorm.DB rather than going
// through provider/repository_provider.go, since nothing else needs these
// repositories yet (mirrors ticket 03's tcgdex ingester).
type Ingester struct {
	games    crud.Repository[entity.Game]
	locales  crud.Repository[entity.Locale]
	series   crud.Repository[entity.Series]
	sets     crud.Repository[entity.ExpansionSet]
	rarities crud.Repository[entity.Rarity]
	cards    crud.Repository[entity.Card]
	client   *client

	// gameID is set once at the start of Run and read (never written)
	// concurrently afterward, so no locking is needed for it.
	gameID uuid.UUID

	// rarityMu guards rarityCache: a (GameID, code)->ID cache for
	// resolveRarityID, looked up concurrently by every card in a set's
	// errgroup. Keyed on GameID too, not just code, to match Rarity's own
	// natural key (game_id, code) — even though gameID is fixed for this
	// Ingester's whole lifetime today, a bare code key would silently
	// collide across games if that ever changed.
	rarityMu    sync.Mutex
	rarityCache map[rarityCacheKey]uuid.UUID

	// failuresMu guards failures, appended to concurrently from ingestSet's
	// errgroup (one goroutine per card) as well as from Run's own sequential
	// loop.
	failuresMu sync.Mutex
	failures   []IngestFailure
}

// recordFailure logs and stores a non-fatal, retries-exhausted failure so
// Run can report it in Summary.Failures instead of aborting the whole
// ingestion over it.
func (in *Ingester) recordFailure(f IngestFailure) {
	logger.Warn(f.Error())
	in.failuresMu.Lock()
	in.failures = append(in.failures, f)
	in.failuresMu.Unlock()
}

type rarityCacheKey struct {
	gameID uuid.UUID
	code   string
}

// NewIngester builds an Ingester backed by the given *gorm.DB.
func NewIngester(db *gorm.DB) *Ingester {
	return &Ingester{
		games:    crud.NewRepository[entity.Game](db),
		locales:  crud.NewRepository[entity.Locale](db),
		series:   crud.NewRepository[entity.Series](db),
		sets:     crud.NewRepository[entity.ExpansionSet](db),
		rarities: crud.NewRepository[entity.Rarity](db),
		cards:    crud.NewRepository[entity.Card](db),
		client:   newClient(),
	}
}

// Run ingests every Expansion Set under every target Series (or, if
// seriesFilter is non-empty, just that one Series — a debug aid; the ticket
// scope is fixed to all four), upserting Game/Series/ExpansionSet/Rarity/
// Card rows. Re-running with the same arguments is idempotent: existing
// rows are updated in place rather than duplicated.
func (in *Ingester) Run(ctx context.Context, seriesFilter string) (Summary, error) {
	logger.Infof("starting ingestion (series filter: %q)", seriesFilter)

	game, err := in.upsertGame(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("upserting game: %w", err)
	}
	in.gameID = game.ID

	locale, err := in.upsertLocale(ctx, localeCode)
	if err != nil {
		return Summary{}, fmt.Errorf("upserting locale: %w", err)
	}

	listings, err := in.enumerateExpansions(ctx, seriesFilter)
	if err != nil {
		return Summary{}, fmt.Errorf("enumerating expansions: %w", err)
	}
	logger.Infof("enumerated %d expansion set(s) to ingest", len(listings))

	var summary Summary
	seriesCache := map[string]entity.Series{}
	for i, listing := range listings {
		if ctx.Err() != nil {
			return summary, ctx.Err()
		}

		seriesRow, ok := seriesCache[listing.Series]
		if !ok {
			seriesRow, err = in.upsertSeries(ctx, game.ID, listing.Series)
			if err != nil {
				in.recordFailure(IngestFailure{ExpansionCode: listing.Code, Stage: "upserting series " + listing.Series, Err: err})
				continue
			}
			seriesCache[listing.Series] = seriesRow
			summary.Series++
		}

		set, err := in.upsertExpansionSet(ctx, game.ID, locale.ID, seriesRow.ID, listing)
		if err != nil {
			in.recordFailure(IngestFailure{ExpansionCode: listing.Code, Stage: "upserting expansion set", Err: err})
			continue
		}
		summary.Sets++

		logger.Infof("[%d/%d] ingesting set %s (%s, series %s)...", i+1, len(listings), listing.Code, listing.Name, listing.Series)
		n, err := in.ingestSet(ctx, set.ID, listing.Code)
		if err != nil {
			// ingestSet only returns an error for a caller-driven context
			// cancellation (see its doc comment) - every other failure is
			// recorded and ingestion of the set continues.
			return summary, fmt.Errorf("ingesting set %s: %w", listing.Code, err)
		}
		summary.Cards += n
		logger.Infof("[%d/%d] finished set %s: %d card(s) ingested", i+1, len(listings), listing.Code, n)
	}

	in.rarityMu.Lock()
	summary.Rarities = len(in.rarityCache)
	in.rarityMu.Unlock()

	in.failuresMu.Lock()
	summary.Failures = append([]IngestFailure(nil), in.failures...)
	in.failuresMu.Unlock()

	logger.Infof(
		"ingestion finished: %d series, %d sets, %d rarities, %d cards, %d failure(s)",
		summary.Series, summary.Sets, summary.Rarities, summary.Cards, len(summary.Failures),
	)

	return summary, nil
}

// enumerateExpansions paginates GET /card-search/?pageNo=N until a page
// returns no listings, keeping only listings under a target Series
// (optionally narrowed further to a single seriesFilter).
func (in *Ingester) enumerateExpansions(ctx context.Context, seriesFilter string) ([]expansionListing, error) {
	var out []expansionListing
	for pageNo := 1; ; pageNo++ {
		doc, _, err := in.client.expansionListPage(ctx, pageNo)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			// Retries are already exhausted at this point (client.get's own
			// job) - a page we can't reach means we can't discover what's
			// beyond it, so stop paginating but keep whatever was already
			// found rather than losing it.
			in.recordFailure(IngestFailure{Stage: fmt.Sprintf("fetching expansion list page %d", pageNo), Err: err})
			break
		}

		page := parseExpansionListings(doc)
		if len(page) == 0 {
			break
		}

		for _, listing := range page {
			if !targetSeries[listing.Series] {
				continue
			}
			if seriesFilter != "" && listing.Series != seriesFilter {
				continue
			}
			out = append(out, listing)
		}
	}
	return out, nil
}

// ingestSet enumerates every card id under one Expansion Set with a single
// regulation=all pass, resolves each card's rarity code via a sweep of the
// results-list rarity filter, then fetches and upserts each card's detail
// with bounded concurrency. It returns the number of cards ingested.
//
// A page or card that fails after client.get exhausts its own retries is
// recorded via in.recordFailure and skipped rather than aborting the set;
// ingestSet only returns a non-nil error when ctx itself has been canceled
// by the caller (e.g. Ctrl+C), since retrying or recording more failures at
// that point would be pointless.
func (in *Ingester) ingestSet(ctx context.Context, expansionSetID uuid.UUID, expansionCode string) (int, error) {
	ids, rarityOptions, err := in.enumerateSetCardIDs(ctx, expansionCode)
	if err != nil {
		return 0, err
	}
	logger.Infof("set %s: %d card(s) to fetch", expansionCode, len(ids))

	rarityByCard, err := in.sweepRarities(ctx, expansionCode, rarityOptions)
	if err != nil {
		return 0, err
	}

	var cardCount atomic.Int64
	var processed atomic.Int64
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(maxConcurrency)
	for _, id := range ids {
		group.Go(func() error {
			defer func() {
				if done := processed.Add(1); done%20 == 0 || int(done) == len(ids) {
					logger.Infof("set %s: processed %d/%d card(s)", expansionCode, done, len(ids))
				}
			}()

			rarityCode, ok := rarityByCard[id]
			if !ok {
				// A card enumerated via regulation=all but not returned by
				// any known rarity[] filter id - log and skip it rather
				// than blocking the rest of the set (see the ticket's
				// "card whose rarity can't be resolved" criterion).
				in.recordFailure(IngestFailure{
					ExpansionCode: expansionCode,
					CardID:        id,
					Stage:         "resolving rarity",
					Err:           fmt.Errorf("card not found under any known rarity filter id"),
				})
				return nil
			}

			if err := in.ingestCard(groupCtx, expansionSetID, id, rarityCode); err != nil {
				if groupCtx.Err() != nil {
					return fmt.Errorf("ingesting card %s: %w", id, err)
				}
				in.recordFailure(IngestFailure{ExpansionCode: expansionCode, CardID: id, Stage: "ingesting card", Err: err})
				return nil
			}
			cardCount.Add(1)
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return 0, err
	}

	return int(cardCount.Load()), nil
}

// enumerateSetCardIDs paginates GET /card-search/list/?...&regulation=all
// for one Expansion Set until a page returns no card ids, returning every
// id found. It also returns the rarity[] filter's id->code options, parsed
// from the first page fetched (the filter widget is present on every
// results-list response regardless of how many cards match - see
// mapper.go's parseRarityFilterOptions).
func (in *Ingester) enumerateSetCardIDs(ctx context.Context, expansionCode string) ([]string, map[string]string, error) {
	var ids []string
	var rarityOptions map[string]string
	for pageNo := 1; ; pageNo++ {
		doc, _, err := in.client.resultsPage(ctx, expansionCode, pageNo)
		if err != nil {
			if ctx.Err() != nil {
				return nil, nil, ctx.Err()
			}
			in.recordFailure(IngestFailure{
				ExpansionCode: expansionCode,
				Stage:         fmt.Sprintf("fetching results page %d", pageNo),
				Err:           err,
			})
			break
		}

		if pageNo == 1 {
			rarityOptions = parseRarityFilterOptions(doc)
		}

		page := parseResultCardIDs(doc)
		if len(page) == 0 {
			break
		}
		ids = append(ids, page...)
	}
	return ids, rarityOptions, nil
}

// sweepRarities resolves every enumerated card's rarity code for one
// Expansion Set. It queries the results-list endpoint once per known
// rarity[] filter id (most return zero cards for a given set) and
// paginates each, attributing that id's code to every card id it returns.
// See ADR-0010: this replaces the mis-scraped detail-page field as the
// authoritative source of Card Rarity.
func (in *Ingester) sweepRarities(ctx context.Context, expansionCode string, rarityOptions map[string]string) (map[string]string, error) {
	byCard := map[string]string{}
	for rarityFilterID, code := range rarityOptions {
		for pageNo := 1; ; pageNo++ {
			doc, _, err := in.client.rarityResultsPage(ctx, expansionCode, rarityFilterID, pageNo)
			if err != nil {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				in.recordFailure(IngestFailure{
					ExpansionCode: expansionCode,
					Stage:         fmt.Sprintf("fetching rarity %s results page %d", code, pageNo),
					Err:           err,
				})
				break
			}

			page := parseResultCardIDs(doc)
			if len(page) == 0 {
				break
			}
			for _, id := range page {
				byCard[id] = code
			}
		}
	}
	return byCard, nil
}

// ingestCard fetches, parses, and upserts one card's detail page.
// rarityCode is the card's already-resolved print rarity (from
// sweepRarities), independent of anything the detail page itself exposes.
func (in *Ingester) ingestCard(ctx context.Context, expansionSetID uuid.UUID, id, rarityCode string) error {
	doc, raw, err := in.client.cardDetail(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching card detail: %w", err)
	}

	detail := parseCardDetail(doc)

	// An empty LocalID is a zero value to gorm's Where(struct), which drops
	// zero-value fields from the WHERE clause instead of matching them
	// literally - an empty LocalID here would broaden the lookup to any
	// existing row for the set and silently corrupt or overwrite it.
	if detail.LocalID == "" {
		return fmt.Errorf("card %s: missing local id", id)
	}

	rarityID, err := in.resolveRarityID(ctx, rarityCode)
	if err != nil {
		return fmt.Errorf("resolving rarity %s: %w", rarityCode, err)
	}

	card := mapCard(detail, expansionSetID, rarityID, cardImageURL(id), raw)
	if _, err := in.upsertCard(ctx, card); err != nil {
		return fmt.Errorf("upserting card: %w", err)
	}
	return nil
}

func (in *Ingester) upsertGame(ctx context.Context) (entity.Game, error) {
	game, err := in.games.FindFirst(ctx, crud.Specification[entity.Game]{
		Model: entity.Game{Slug: gameSlug},
	})
	if err != nil {
		return entity.Game{}, err
	}
	if !game.IsZero() {
		return game, nil
	}
	return in.games.Insert(ctx, entity.Game{Slug: gameSlug, Name: gameName})
}

// upsertLocale find-or-creates a Locale row for the given code. Called once
// serially before any concurrent work, so no caching/locking is needed.
func (in *Ingester) upsertLocale(ctx context.Context, code string) (entity.Locale, error) {
	locale, err := in.locales.FindFirst(ctx, crud.Specification[entity.Locale]{
		Model: entity.Locale{Code: code},
	})
	if err != nil {
		return entity.Locale{}, err
	}
	if !locale.IsZero() {
		return locale, nil
	}
	return in.locales.Insert(ctx, entity.Locale{Code: code})
}

func (in *Ingester) upsertSeries(ctx context.Context, gameID uuid.UUID, name string) (entity.Series, error) {
	code := slugifySeries(name)

	existing, err := in.series.FindFirst(ctx, crud.Specification[entity.Series]{
		Model: entity.Series{GameID: gameID, Code: code},
	})
	if err != nil {
		return entity.Series{}, err
	}
	if existing.IsZero() {
		return in.series.Insert(ctx, entity.Series{GameID: gameID, Code: code, Name: name})
	}
	if existing.Name != name {
		existing.Name = name
		return in.series.Update(ctx, existing)
	}
	return existing, nil
}

func (in *Ingester) upsertExpansionSet(
	ctx context.Context, gameID, localeID, seriesID uuid.UUID, listing expansionListing,
) (entity.ExpansionSet, error) {
	existing, err := in.sets.FindFirst(ctx, crud.Specification[entity.ExpansionSet]{
		Model: entity.ExpansionSet{GameID: gameID, Code: listing.Code},
	})
	if err != nil {
		return entity.ExpansionSet{}, err
	}

	releaseDate := listing.ReleaseDate
	if existing.IsZero() {
		return in.sets.Insert(ctx, entity.ExpansionSet{
			GameID:      gameID,
			Code:        listing.Code,
			Name:        listing.Name,
			LocaleID:    localeID,
			SeriesID:    &seriesID,
			ReleaseDate: &releaseDate,
		})
	}

	existing.Name = listing.Name
	existing.LocaleID = localeID
	existing.SeriesID = &seriesID
	existing.ReleaseDate = &releaseDate
	return in.sets.Update(ctx, existing)
}

func (in *Ingester) upsertCard(ctx context.Context, card entity.Card) (entity.Card, error) {
	existing, err := in.cards.FindFirst(ctx, crud.Specification[entity.Card]{
		Model: entity.Card{ExpansionSetID: card.ExpansionSetID, LocalID: card.LocalID},
	})
	if err != nil {
		return entity.Card{}, err
	}
	if existing.IsZero() {
		return in.cards.Insert(ctx, card)
	}

	existing.Name = card.Name
	existing.Category = card.Category
	existing.Illustrator = card.Illustrator
	existing.Tags = card.Tags
	existing.RarityID = card.RarityID
	existing.ImageURL = card.ImageURL
	existing.Attributes = card.Attributes
	existing.Raw = card.Raw
	return in.cards.Update(ctx, existing)
}

// resolveRarityID find-or-creates a Rarity row for the given code and
// caches the result. Guarded by rarityMu since it's called concurrently by
// every card in a set's errgroup. Name defaults to the raw code: the source
// page exposes no readable rarity name (see docs/adr/0009) — a human
// curates a nicer Name later without needing a new ingestion field.
func (in *Ingester) resolveRarityID(ctx context.Context, code string) (uuid.UUID, error) {
	in.rarityMu.Lock()
	defer in.rarityMu.Unlock()

	key := rarityCacheKey{gameID: in.gameID, code: code}
	if id, ok := in.rarityCache[key]; ok {
		return id, nil
	}

	rarity, err := in.rarities.FindFirst(ctx, crud.Specification[entity.Rarity]{
		Model: entity.Rarity{GameID: in.gameID, Code: code},
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	if rarity.IsZero() {
		rarity, err = in.rarities.Insert(ctx, entity.Rarity{GameID: in.gameID, Code: code, Name: code})
		if err != nil {
			return uuid.UUID{}, err
		}
	}

	if in.rarityCache == nil {
		in.rarityCache = map[rarityCacheKey]uuid.UUID{}
	}
	in.rarityCache[key] = rarity.ID
	return rarity.ID, nil
}

// staleRegulationMarkCodes is the closed set of single-letter Regulation
// Mark codes (see CONTEXT.md) the pre-fix ingester mistakenly wrote into
// rarities.code instead of a real Kelangkaan rarity code (ticket 11 /
// ADR-0010). Real rarity codes are a different, longer vocabulary, except
// where they coincidentally collide with a Regulation Mark letter (e.g.
// "C", "A" are both real rarity codes and real Regulation Mark letters) -
// CleanupStaleRarities only deletes a colliding code's row once it has zero
// referencing Cards, never unconditionally.
var staleRegulationMarkCodes = []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}

// CleanupResult reports what CleanupStaleRarities did (or found) for one
// candidate code.
type CleanupResult struct {
	Code    string
	Deleted bool
	// CardCount is >0 when the row was left in place because Cards still
	// reference it (Deleted is always false in that case).
	CardCount int
}

// CleanupStaleRarities is a one-time, opt-in fix for ticket 11: earlier runs
// of this ingester (before the rarity/Regulation-Mark bug fix) populated
// `rarities` rows with Regulation Mark letters instead of real print-rarity
// codes. It deletes only rows whose code is in staleRegulationMarkCodes AND
// have zero referencing Cards. A matching row that's still referenced is
// reported, not deleted - that means a corrected re-ingestion of whatever
// set(s) reference it hasn't fully run yet, and deleting it now would
// dangle those Cards' rarity_id.
func (in *Ingester) CleanupStaleRarities(ctx context.Context) ([]CleanupResult, error) {
	return in.cleanupRaritiesByCode(ctx, staleRegulationMarkCodes)
}

// cleanupRaritiesByCode implements CleanupStaleRarities against an
// arbitrary candidate code list. Split out so tests can exercise the
// delete-vs-report logic against codes they control, independent of
// whatever real rarity data already exists under staleRegulationMarkCodes
// in this package's shared, non-truncated test DB (see testdb_test.go).
func (in *Ingester) cleanupRaritiesByCode(ctx context.Context, codes []string) ([]CleanupResult, error) {
	game, err := in.upsertGame(ctx)
	if err != nil {
		return nil, fmt.Errorf("upserting game: %w", err)
	}

	var results []CleanupResult
	for _, code := range codes {
		rarity, err := in.rarities.FindFirst(ctx, crud.Specification[entity.Rarity]{
			Model: entity.Rarity{GameID: game.ID, Code: code},
		})
		if err != nil {
			return results, fmt.Errorf("looking up rarity %s: %w", code, err)
		}
		if rarity.IsZero() {
			continue
		}

		referencingCards, err := in.cards.FindAll(ctx, crud.Specification[entity.Card]{
			Model: entity.Card{RarityID: rarity.ID},
		})
		if err != nil {
			return results, fmt.Errorf("checking references to rarity %s: %w", code, err)
		}
		if len(referencingCards) > 0 {
			logger.Warnf("rarity %q (id %s) still referenced by %d card(s) - not deleted", code, rarity.ID, len(referencingCards))
			results = append(results, CleanupResult{Code: code, CardCount: len(referencingCards)})
			continue
		}

		if err := in.rarities.Delete(ctx, rarity); err != nil {
			return results, fmt.Errorf("deleting stale rarity %s: %w", code, err)
		}
		logger.Infof("deleted stale rarity %q (id %s)", code, rarity.ID)
		results = append(results, CleanupResult{Code: code, Deleted: true})
	}

	return results, nil
}
