package pokemonasia

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
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

// regulationLabels maps the site's own regulation query-partition (1, 2, 3)
// to its own label text, captured into cards.attributes.regulation — see
// the plan's "Enumerate cards per set as 3 regulation-partitioned passes"
// decision.
var regulationLabels = map[int]string{
	1: "Standar",
	2: "Luas",
	3: "Lainnya",
}

// Summary reports row counts from a completed ingestion run.
type Summary struct {
	Series   int
	Sets     int
	Rarities int
	Cards    int
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

	var summary Summary
	seriesCache := map[string]entity.Series{}
	for _, listing := range listings {
		seriesRow, ok := seriesCache[listing.Series]
		if !ok {
			seriesRow, err = in.upsertSeries(ctx, game.ID, listing.Series)
			if err != nil {
				return summary, fmt.Errorf("upserting series %s: %w", listing.Series, err)
			}
			seriesCache[listing.Series] = seriesRow
			summary.Series++
		}

		set, err := in.upsertExpansionSet(ctx, game.ID, locale.ID, seriesRow.ID, listing)
		if err != nil {
			return summary, fmt.Errorf("upserting expansion set %s: %w", listing.Code, err)
		}
		summary.Sets++

		n, err := in.ingestSet(ctx, set.ID, listing.Code)
		if err != nil {
			return summary, fmt.Errorf("ingesting set %s: %w", listing.Code, err)
		}
		summary.Cards += n
	}

	in.rarityMu.Lock()
	summary.Rarities = len(in.rarityCache)
	in.rarityMu.Unlock()

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
			return nil, fmt.Errorf("fetching expansion list page %d: %w", pageNo, err)
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

// ingestSet enumerates every card id under one Expansion Set across the 3
// regulation-partitioned passes, then fetches and upserts each card's
// detail with bounded concurrency. It returns the number of cards ingested.
func (in *Ingester) ingestSet(ctx context.Context, expansionSetID uuid.UUID, expansionCode string) (int, error) {
	type job struct {
		id         string
		regulation string
	}

	// seen dedupes ids across regulation passes: the site's regulation query
	// param doesn't reliably partition results (confirmed live: bucket 1 and
	// bucket 2 can both return the same full card list for a set), so
	// without this a card gets refetched/re-upserted once per bucket it
	// appears in, and whichever bucket's write lands last would win
	// attributes.regulation nondeterministically. Keeping the
	// lowest-numbered (most specific/default) bucket a card is seen in is
	// as good a guess as the site gives us.
	seen := map[string]bool{}
	var jobs []job
	for regulation := 1; regulation <= 3; regulation++ {
		label := regulationLabels[regulation]
		for pageNo := 1; ; pageNo++ {
			doc, _, err := in.client.resultsPage(ctx, expansionCode, regulation, pageNo)
			if err != nil {
				return 0, fmt.Errorf("fetching results page %d (regulation %d): %w", pageNo, regulation, err)
			}

			ids := parseResultCardIDs(doc)
			if len(ids) == 0 {
				break
			}
			for _, id := range ids {
				if seen[id] {
					continue
				}
				seen[id] = true
				jobs = append(jobs, job{id: id, regulation: label})
			}
		}
	}

	var cardCount atomic.Int64
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(maxConcurrency)
	for _, j := range jobs {
		group.Go(func() error {
			if err := in.ingestCard(groupCtx, expansionSetID, j.id, j.regulation); err != nil {
				return fmt.Errorf("ingesting card %s: %w", j.id, err)
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

// ingestCard fetches, parses, and upserts one card's detail page.
func (in *Ingester) ingestCard(ctx context.Context, expansionSetID uuid.UUID, id, regulation string) error {
	doc, raw, err := in.client.cardDetail(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching card detail: %w", err)
	}

	detail := parseCardDetail(doc)
	detail.Regulation = regulation

	rarityID, err := in.resolveRarityID(ctx, detail.RarityCode)
	if err != nil {
		return fmt.Errorf("resolving rarity %s: %w", detail.RarityCode, err)
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
