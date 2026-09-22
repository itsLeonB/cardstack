package tcgdex

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

const (
	// gameSlug/gameName identify the single Game row this ingester
	// find-or-creates on every run. Ticket 04's MA ingestion finds this
	// same row by slug rather than creating a duplicate.
	gameSlug = "pokemon-tcg"
	gameName = "Pokémon TCG"

	// maxConcurrency bounds concurrent HTTP requests during ingestion. A
	// full SV/id pull is several thousand requests; errgroup.SetLimit
	// replaces a hand-rolled worker pool.
	maxConcurrency = 12
)

// nameLocales are the additional locales fetched for each card's display
// name, beyond the primary locale (whose full detail is read once). A
// locale that 404s for a given card is skipped, not an error — some
// locales legitimately have no data for a given card (e.g. `en` for a
// SV/Indonesian-only print run).
var nameLocales = []string{"ja", "zh-tw", "th"}

// Summary reports row counts from a completed ingestion run. Mismatches
// lists sets where TCGDex's own cardCount.official didn't match the number
// of cards actually ingested — an advisory sanity signal, not a failure
// (a card can legitimately fail to decode into a row for reasons TCGDex's
// own count doesn't reflect).
type Summary struct {
	Sets       int
	Cards      int
	Variants   int
	Mismatches []string
}

// Ingester ingests TCGDex catalog data into the games/expansion_sets/cards/
// card_variants tables. It constructs its own crud.Repository instances
// directly from a *gorm.DB rather than going through
// provider/repository_provider.go, since nothing else needs these
// repositories yet.
type Ingester struct {
	games    crud.Repository[entity.Game]
	sets     crud.Repository[entity.ExpansionSet]
	cards    crud.Repository[entity.Card]
	variants crud.Repository[entity.CardVariant]
	client   *client
}

// NewIngester builds an Ingester backed by the given *gorm.DB.
func NewIngester(db *gorm.DB) *Ingester {
	return &Ingester{
		games:    crud.NewRepository[entity.Game](db),
		sets:     crud.NewRepository[entity.ExpansionSet](db),
		cards:    crud.NewRepository[entity.Card](db),
		variants: crud.NewRepository[entity.CardVariant](db),
		client:   newClient(),
	}
}

// Run ingests every set in the given series (e.g. "sv") for the given
// locale (e.g. "id"), upserting Game/ExpansionSet/Card/CardVariant rows.
// Re-running with the same arguments is idempotent: existing rows are
// updated in place rather than duplicated.
func (in *Ingester) Run(ctx context.Context, locale, seriesID string) (Summary, error) {
	game, err := in.upsertGame(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("upserting game: %w", err)
	}

	series, err := in.client.getSeries(ctx, locale, seriesID)
	if err != nil {
		return Summary{}, fmt.Errorf("fetching series %s (%s): %w", seriesID, locale, err)
	}

	var summary Summary
	for _, setRef := range series.Sets {
		result, err := in.ingestSet(ctx, game.ID, locale, setRef)
		if err != nil {
			return summary, fmt.Errorf("ingesting set %s: %w", setRef.ID, err)
		}
		summary.Sets++
		summary.Cards += result.cards
		summary.Variants += result.variants
		if result.official != 0 && result.cards != result.official {
			summary.Mismatches = append(summary.Mismatches, fmt.Sprintf(
				"%s: ingested %d cards, TCGDex reports %d official", setRef.ID, result.cards, result.official,
			))
		}
	}

	return summary, nil
}

// setResult is ingestSet's internal return shape: row counts plus TCGDex's
// own official card count for the caller to compare against (see
// Summary.Mismatches).
type setResult struct {
	cards, variants, official int
}

// ingestSet upserts one ExpansionSet row and every Card/CardVariant row
// under it, fetching and upserting cards with bounded concurrency.
func (in *Ingester) ingestSet(ctx context.Context, gameID uuid.UUID, locale string, setRef seriesSetRef) (setResult, error) {
	expansionSet, err := in.upsertExpansionSet(ctx, gameID, locale, setRef)
	if err != nil {
		return setResult{}, fmt.Errorf("upserting expansion set: %w", err)
	}

	set, err := in.client.getSet(ctx, locale, setRef.ID)
	if err != nil {
		return setResult{}, fmt.Errorf("fetching set detail: %w", err)
	}

	var cardTotal, variantTotal atomic.Int64

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(maxConcurrency)

	for _, cardRef := range set.Cards {
		group.Go(func() error {
			n, err := in.ingestCard(groupCtx, expansionSet.ID, locale, cardRef)
			if err != nil {
				return fmt.Errorf("ingesting card %s: %w", cardRef.ID, err)
			}
			cardTotal.Add(1)
			variantTotal.Add(int64(n))
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return setResult{}, err
	}

	return setResult{
		cards:    int(cardTotal.Load()),
		variants: int(variantTotal.Load()),
		official: set.CardCount.Official,
	}, nil
}

// ingestCard fetches one card's full detail (primary locale) plus its
// display name in each of nameLocales, upserts the Card row, and upserts
// one CardVariant row per true finish flag. It returns the number of
// variant rows upserted.
func (in *Ingester) ingestCard(ctx context.Context, expansionSetID uuid.UUID, locale string, cardRef setCardRef) (int, error) {
	detail, found, err := in.client.getCard(ctx, locale, cardRef.ID)
	if err != nil {
		return 0, fmt.Errorf("fetching primary (%s) detail: %w", locale, err)
	}
	if !found {
		return 0, fmt.Errorf("card missing from its own set's primary locale (%s)", locale)
	}

	names := map[string]string{locale: detail.Name}
	for _, nameLocale := range nameLocales {
		nameResp, found, err := in.client.getCard(ctx, nameLocale, cardRef.ID)
		if err != nil {
			return 0, fmt.Errorf("fetching %s name: %w", nameLocale, err)
		}
		if found {
			names[nameLocale] = nameResp.Name
		}
	}

	card, err := in.upsertCard(ctx, mapCard(detail, expansionSetID, names))
	if err != nil {
		return 0, fmt.Errorf("upserting card: %w", err)
	}

	variantCount := 0
	for _, variant := range mapVariants(detail.Variants) {
		if err := in.upsertVariant(ctx, card.ID, variant.Finish); err != nil {
			return 0, fmt.Errorf("upserting variant %s: %w", variant.Finish, err)
		}
		variantCount++
	}

	return variantCount, nil
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

func (in *Ingester) upsertExpansionSet(ctx context.Context, gameID uuid.UUID, locale string, setRef seriesSetRef) (entity.ExpansionSet, error) {
	existing, err := in.sets.FindFirst(ctx, crud.Specification[entity.ExpansionSet]{
		Model: entity.ExpansionSet{GameID: gameID, Code: setRef.ID},
	})
	if err != nil {
		return entity.ExpansionSet{}, err
	}
	if existing.IsZero() {
		return in.sets.Insert(ctx, entity.ExpansionSet{
			GameID: gameID,
			Code:   setRef.ID,
			Name:   setRef.Name,
			Locale: locale,
		})
	}
	if existing.Name != setRef.Name || existing.Locale != locale {
		existing.Name = setRef.Name
		existing.Locale = locale
		return in.sets.Update(ctx, existing)
	}
	return existing, nil
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
	existing.Names = card.Names
	existing.Rarity = card.Rarity
	existing.ImageURL = card.ImageURL
	existing.Attributes = card.Attributes
	return in.cards.Update(ctx, existing)
}

// upsertVariant find-or-creates a CardVariant row. Unlike Card/ExpansionSet
// there is nothing to update on a re-run: (CardID, Finish) is the whole
// identity, so existence alone means the row is up to date.
func (in *Ingester) upsertVariant(ctx context.Context, cardID uuid.UUID, finish string) error {
	existing, err := in.variants.FindFirst(ctx, crud.Specification[entity.CardVariant]{
		Model: entity.CardVariant{CardID: cardID, Finish: finish},
	})
	if err != nil {
		return err
	}
	if !existing.IsZero() {
		return nil
	}
	_, err = in.variants.Insert(ctx, entity.CardVariant{CardID: cardID, Finish: finish})
	return err
}
