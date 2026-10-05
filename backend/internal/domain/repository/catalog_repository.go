package repository

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/ezutil/v2"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// CatalogRepository is the persistence access the catalog domain needs
// (ticket 05): browsing Series/Expansion Sets/Rarities/categories/tags and
// searching Cards. It covers the full read surface catalogRepository (this
// package's GORM-backed implementation) implements, not narrowed to any one
// caller's needs, so a service depending on it can be tested against a
// mockery-generated mock (see internal/mocks) instead of
// a real Postgres.
//
// Interface and implementation live together in this one file, under
// internal/domain/repository rather than internal/adapters/repository: per
// PR review, adapters is for interchangeable infrastructure (messaging,
// email, cache, object storage), not domain logic and the database access
// behind it, which is unlikely to change and belongs in domain.
type CatalogRepository interface {
	// ListSeries returns every Series, ordered by name.
	ListSeries(ctx context.Context) ([]entity.Series, error)
	// ListExpansionSets returns every Expansion Set whose SeriesID is in
	// seriesIDs, ordered by release date (sets with an unknown release date
	// sort last) then name.
	ListExpansionSets(ctx context.Context, seriesIDs []uuid.UUID) ([]entity.ExpansionSet, error)
	// ListUngroupedExpansionSets returns every Expansion Set whose SeriesID
	// is nil, ordered the same way ListExpansionSets orders each Series's
	// sets.
	ListUngroupedExpansionSets(ctx context.Context) ([]entity.ExpansionSet, error)
	// ListRarities returns every Rarity across all Games, ordered by name.
	ListRarities(ctx context.Context) ([]entity.Rarity, error)
	// ListDistinctCategories returns the distinct Card.Category values
	// actually in use, ordered alphabetically.
	ListDistinctCategories(ctx context.Context) ([]string, error)
	// ListDistinctTags returns the distinct values found across every
	// Card's Tags array, ordered alphabetically.
	ListDistinctTags(ctx context.Context) ([]string, error)
	// SearchCards returns the Cards matching filter (joined with their
	// Rarity and Expansion Set), limited/offset per filter, plus the total
	// number of Cards matching filter before that pagination.
	SearchCards(ctx context.Context, filter CardFilter) ([]CardResult, int64, error)
	// ListCardFacets returns, for each facet, the options available among
	// the Cards matching every filter except that facet's own (OR within a
	// facet, AND across), always including that facet's selected values.
	ListCardFacets(ctx context.Context, filter CardFilter) (CardFacets, error)
}

// catalogRepository is CatalogRepository's GORM-backed implementation. It
// embeds a crud.Repository only for GetGormInstance, which every method uses
// to get the database (the transaction carried by ctx, if any). The entity
// type is arbitrary: no method uses the base's CRUD operations.
type catalogRepository struct {
	crud.Repository[entity.Card]
}

// NewCatalogRepository builds a CatalogRepository over base's database.
func NewCatalogRepository(base crud.Repository[entity.Card]) CatalogRepository {
	return &catalogRepository{Repository: base}
}

// ListSeries returns every Series, most recently released first. A Series has
// no release date of its own: it is ordered by the earliest release date among
// its own Expansion Sets, derived at query time. A Series with no known date
// (all its sets undated, or no sets) sorts last; ties break by name.
func (r *catalogRepository) ListSeries(ctx context.Context) ([]entity.Series, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var series []entity.Series
	err = db.
		Select("series.*").
		Joins("LEFT JOIN (SELECT series_id, MIN(release_date) AS release_date FROM expansion_sets GROUP BY series_id) AS set_dates ON set_dates.series_id = series.id").
		Order("set_dates.release_date DESC NULLS LAST, series.name ASC").
		Find(&series).
		Error
	return series, err
}

// ListExpansionSets returns every Expansion Set whose SeriesID is in
// seriesIDs, ordered by release date, most recent first (sets with an unknown
// release date sort last), then name. An empty seriesIDs returns no rows
// rather than every Expansion Set, since the only caller (ListSeries's
// nesting) always passes the Series it actually found.
func (r *catalogRepository) ListExpansionSets(ctx context.Context, seriesIDs []uuid.UUID) ([]entity.ExpansionSet, error) {
	if len(seriesIDs) == 0 {
		return nil, nil
	}

	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var sets []entity.ExpansionSet
	err = db.
		Where("series_id IN ?", seriesIDs).
		Order("release_date DESC NULLS LAST, name ASC").
		Find(&sets).
		Error
	return sets, err
}

// ListUngroupedExpansionSets returns every Expansion Set whose SeriesID is
// nil, ordered the same way ListExpansionSets orders each Series's sets: by
// release date, most recent first (sets with an unknown release date sort
// last), then name. A series-less Expansion Set is a legitimate domain state
// (see GLOSSARY.md's Series entry), not an edge case to special-case away -
// this is how it's surfaced through GET /catalog/series alongside the grouped
// Series.
func (r *catalogRepository) ListUngroupedExpansionSets(ctx context.Context) ([]entity.ExpansionSet, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var sets []entity.ExpansionSet
	err = db.
		Where("series_id IS NULL").
		Order("release_date DESC NULLS LAST, name ASC").
		Find(&sets).
		Error
	return sets, err
}

// ListRarities returns every Rarity across all Games, ordered by name.
func (r *catalogRepository) ListRarities(ctx context.Context) ([]entity.Rarity, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var rarities []entity.Rarity
	err = db.Order("name ASC").Find(&rarities).Error
	return rarities, err
}

// ListDistinctCategories returns the distinct Card.Category values actually
// in use, ordered alphabetically.
func (r *catalogRepository) ListDistinctCategories(ctx context.Context) ([]string, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var categories []string
	err = db.
		Model(&entity.Card{}).
		Distinct("category").
		Order("category ASC").
		Pluck("category", &categories).
		Error
	return categories, err
}

// ListDistinctTags returns the distinct values found across every Card's
// Tags array, ordered alphabetically. Tags is a JSONB array (see
// entity.Card's doc comment), so this unnests it with Postgres's
// jsonb_array_elements_text rather than something Pluck/Distinct can
// express directly. The jsonb_typeof guard skips any row whose Tags isn't
// actually a JSON array (e.g. an unset Tags column stores JSON null) -
// jsonb_array_elements_text errors on a non-array/scalar value otherwise.
func (r *catalogRepository) ListDistinctTags(ctx context.Context) ([]string, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var tags []string
	err = db.
		Raw(`SELECT DISTINCT tag FROM cards, jsonb_array_elements_text(cards.tags) AS tag
			WHERE jsonb_typeof(cards.tags) = 'array'
			ORDER BY tag ASC`).
		Scan(&tags).
		Error
	return tags, err
}

// CardFilter narrows CatalogRepository.SearchCards and ListCardFacets. An
// empty/zero field means "don't filter on this facet". Values within one
// multi-value field combine with OR; fields combine with AND.
type CardFilter struct {
	// CollectionID, when set, restricts the base set to that Collection's
	// Cards (SearchCards also returns their quantity). The caller must have
	// checked ownership; uuid.Nil means the whole catalog.
	CollectionID uuid.UUID
	// ProfileID, when set (and CollectionID is not), restricts the base set to
	// the Cards the profile owns across all its Collections, with Quantity the
	// sum over them (Master Inventory). Only owned Cards appear (entry quantity is CHECK > 0).
	ProfileID       uuid.UUID
	Name            string
	ExpansionSetIDs []uuid.UUID
	LocalID         string
	RarityIDs       []uuid.UUID
	Categories      []string
	Tags            []string
	CardIDs         []uuid.UUID
	Limit           int
	Offset          int
}

// CardFacets is ListCardFacets's result. Each option set already includes
// the facet's selected values, flagged Available=false when the other
// filters leave no matching Card for them.
type CardFacets struct {
	ExpansionSets []ExpansionSetFacetOption
	Rarities      []RarityFacetOption
	Categories    []StringFacetOption
	Tags          []StringFacetOption
}

type ExpansionSetFacetOption struct {
	entity.ExpansionSet
	Available bool
}

type RarityFacetOption struct {
	entity.Rarity
	Available bool
}

type StringFacetOption struct {
	Value     string
	Available bool
}

// CardResult is one row of a SearchCards result: a Card joined with its
// Rarity and Expansion Set, already resolved to the readable fields a
// catalog search result needs (see docs/adr/0009 on why Rarity in
// particular can't be shown as a bare code).
type CardResult struct {
	ID                      uuid.UUID
	LocalID                 string
	Name                    string
	Category                string
	Illustrator             string
	Tags                    datatypes.JSONSlice[string]
	ImageKey                string
	RarityID                uuid.UUID
	RarityCode              string
	RarityName              string
	ExpansionSetID          uuid.UUID
	ExpansionSetCode        string
	ExpansionSetName        string
	ExpansionSetReleaseDate *time.Time
	ExpansionSetImageKey    string
	// Quantity is set only when searching within a Collection or a profile's Master Inventory.
	Quantity int
}

const cardResultColumns = `cards.id AS id,
	cards.local_id AS local_id,
	cards.name AS name,
	cards.category AS category,
	cards.illustrator AS illustrator,
	cards.tags AS tags,
	cards.image_key AS image_key,
	cards.rarity_id AS rarity_id,
	rarities.code AS rarity_code,
	rarities.name AS rarity_name,
	cards.expansion_set_id AS expansion_set_id,
	expansion_sets.code AS expansion_set_code,
	expansion_sets.name AS expansion_set_name,
	expansion_sets.release_date AS expansion_set_release_date,
	expansion_sets.image_key AS expansion_set_image_key`

// likeEscaper escapes ILIKE's own wildcard characters (%, _) and its default
// escape character (\) in a single pass, so a literal % or _ in user input
// can't widen a search beyond what the user typed (e.g. searching "A_B"
// matching "AxB" too, since _ means "any one character" unless escaped).
var likeEscaper = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`)

// cardsBase is the query over cards that filters apply to: all Cards, or only
// the Collection's when filter.CollectionID is set.
func cardsBase(db *gorm.DB, filter CardFilter) *gorm.DB {
	q := db.Table("cards")
	if filter.CollectionID != uuid.Nil {
		q = q.Joins("JOIN inventory_entries ON inventory_entries.card_id = cards.id AND inventory_entries.collection_id = ?", filter.CollectionID)
	} else if filter.ProfileID != uuid.Nil {
		q = q.Joins(`JOIN (SELECT ie.card_id, SUM(ie.quantity) AS quantity
			FROM inventory_entries ie JOIN collections c ON c.id = ie.collection_id
			WHERE c.profile_id = ? GROUP BY ie.card_id) AS master_inventory
			ON master_inventory.card_id = cards.id`, filter.ProfileID)
	}
	return q
}

// applyCardFilters adds filter's non-empty facets as WHERE conditions to
// query, which must be scoped to a query over the cards table (the
// conditions only touch cards columns). Passing a narrower query (e.g. one
// limited to a Collection's Cards) scopes the conditions, and so the facets,
// to that base set.
func applyCardFilters(query *gorm.DB, filter CardFilter) *gorm.DB {
	if len(filter.CardIDs) > 0 {
		query = query.Where("cards.id IN ?", filter.CardIDs)
	}
	if filter.Name != "" {
		query = query.Where("cards.name ILIKE ?", "%"+likeEscaper.Replace(filter.Name)+"%")
	}
	if len(filter.ExpansionSetIDs) > 0 {
		query = query.Where("cards.expansion_set_id IN ?", filter.ExpansionSetIDs)
	}
	if filter.LocalID != "" {
		query = query.Where("cards.local_id = ?", filter.LocalID)
	}
	if len(filter.RarityIDs) > 0 {
		query = query.Where("cards.rarity_id IN ?", filter.RarityIDs)
	}
	if len(filter.Categories) > 0 {
		query = query.Where("cards.category IN ?", filter.Categories)
	}
	if len(filter.Tags) > 0 {
		// cards.tags is JSONB (see entity.Card's doc comment); @> is
		// Postgres's jsonb containment operator. One containment test per
		// tag, OR-ed.
		conds := ezutil.MapSlice(filter.Tags, func(string) string { return "cards.tags @> jsonb_build_array(?::text)" })
		args := ezutil.MapSlice(filter.Tags, func(tag string) any { return tag })
		query = query.Where("("+strings.Join(conds, " OR ")+")", args...)
	}

	return query
}

// SearchCards returns the Cards matching filter (joined with their Rarity
// and Expansion Set), limited/offset per filter, plus the total number of
// Cards matching filter before that pagination.
func (r *catalogRepository) SearchCards(ctx context.Context, filter CardFilter) ([]CardResult, int64, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, 0, err
	}

	base := cardsBase(db, filter).
		Joins("JOIN rarities ON rarities.id = cards.rarity_id").
		Joins("JOIN expansion_sets ON expansion_sets.id = cards.expansion_set_id")

	base = applyCardFilters(base, filter)

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	columns := cardResultColumns
	if filter.CollectionID != uuid.Nil {
		columns += ", inventory_entries.quantity AS quantity"
	} else if filter.ProfileID != uuid.Nil {
		columns += ", master_inventory.quantity AS quantity"
	}

	var results []CardResult
	err = base.Session(&gorm.Session{}).
		Select(columns).
		Order("expansion_sets.release_date DESC NULLS LAST, expansion_sets.id ASC, cards.local_id ASC, cards.name ASC, cards.id ASC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&results).
		Error

	return results, total, err
}

// ListCardFacets returns each facet's options: the values present on Cards
// matching every other facet's filter (its own selection excluded), merged
// with its selected values so a selection never disappears.
func (r *catalogRepository) ListCardFacets(ctx context.Context, filter CardFilter) (CardFacets, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return CardFacets{}, err
	}

	without := func(mutate func(*CardFilter)) *gorm.DB {
		f := filter
		mutate(&f)
		return applyCardFilters(cardsBase(db, f), f)
	}

	var facets CardFacets

	setIDs, err := pluckDistinct[uuid.UUID](without(func(f *CardFilter) { f.ExpansionSetIDs = nil }), "cards.expansion_set_id")
	if err != nil {
		return CardFacets{}, err
	}
	var sets []entity.ExpansionSet
	if ids := unionIDs(setIDs, filter.ExpansionSetIDs); len(ids) > 0 {
		if err := db.Where("id IN ?", ids).
			Order("release_date ASC NULLS LAST, name ASC").Find(&sets).Error; err != nil {
			return CardFacets{}, ungerr.Wrap(err, "listing expansion set facet options")
		}
	}
	for _, s := range sets {
		facets.ExpansionSets = append(facets.ExpansionSets, ExpansionSetFacetOption{ExpansionSet: s, Available: slices.Contains(setIDs, s.ID)})
	}

	rarityIDs, err := pluckDistinct[uuid.UUID](without(func(f *CardFilter) { f.RarityIDs = nil }), "cards.rarity_id")
	if err != nil {
		return CardFacets{}, err
	}
	var rarities []entity.Rarity
	if ids := unionIDs(rarityIDs, filter.RarityIDs); len(ids) > 0 {
		if err := db.Where("id IN ?", ids).Order("name ASC").Find(&rarities).Error; err != nil {
			return CardFacets{}, ungerr.Wrap(err, "listing rarity facet options")
		}
	}
	for _, ra := range rarities {
		facets.Rarities = append(facets.Rarities, RarityFacetOption{Rarity: ra, Available: slices.Contains(rarityIDs, ra.ID)})
	}

	categories, err := pluckDistinct[string](without(func(f *CardFilter) { f.Categories = nil }), "cards.category")
	if err != nil {
		return CardFacets{}, err
	}
	facets.Categories = stringOptions(categories, filter.Categories)

	// Unnest the JSONB tags array; the jsonb_typeof guard skips rows whose
	// tags isn't an array (jsonb_array_elements_text errors on those).
	tags, err := pluckDistinct[string](
		without(func(f *CardFilter) { f.Tags = nil }).
			Joins("CROSS JOIN LATERAL jsonb_array_elements_text(cards.tags) AS tag").
			Where("jsonb_typeof(cards.tags) = 'array'"),
		"tag")
	if err != nil {
		return CardFacets{}, err
	}
	facets.Tags = stringOptions(tags, filter.Tags)

	return facets, nil
}

func pluckDistinct[T any](query *gorm.DB, column string) ([]T, error) {
	var values []T
	if err := query.Distinct().Pluck(column, &values).Error; err != nil {
		return nil, ungerr.Wrap(err, "plucking distinct facet values")
	}
	return values, nil
}

func unionIDs(available, selected []uuid.UUID) []uuid.UUID {
	all := slices.Concat(available, selected)
	slices.SortFunc(all, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	return slices.Compact(all)
}

// stringOptions merges available with selected (flagged unavailable when
// absent from available), sorted alphabetically.
func stringOptions(available, selected []string) []StringFacetOption {
	all := slices.Compact(slices.Sorted(slices.Values(slices.Concat(available, selected))))
	return ezutil.MapSlice(all, func(v string) StringFacetOption {
		return StringFacetOption{Value: v, Available: slices.Contains(available, v)}
	})
}
