package repository

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
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
// holds the bare *gorm.DB rather than going through
// crud.Repository/GetGormInstance's transaction lookup: every method here is
// a plain read that never needs to participate in a write transaction.
type catalogRepository struct {
	db *gorm.DB
}

// NewCatalogRepository builds a CatalogRepository over db.
func NewCatalogRepository(db *gorm.DB) CatalogRepository {
	return &catalogRepository{db: db}
}

// ListSeries returns every Series, ordered by name.
func (r *catalogRepository) ListSeries(ctx context.Context) ([]entity.Series, error) {
	var series []entity.Series
	err := r.db.WithContext(ctx).Order("name ASC").Find(&series).Error
	return series, err
}

// ListExpansionSets returns every Expansion Set whose SeriesID is in
// seriesIDs, ordered by release date (sets with an unknown release date
// sort last) then name. An empty seriesIDs returns no rows rather than
// every Expansion Set, since the only caller (ListSeries's nesting) always
// passes the Series it actually found.
func (r *catalogRepository) ListExpansionSets(ctx context.Context, seriesIDs []uuid.UUID) ([]entity.ExpansionSet, error) {
	if len(seriesIDs) == 0 {
		return nil, nil
	}

	var sets []entity.ExpansionSet
	err := r.db.WithContext(ctx).
		Where("series_id IN ?", seriesIDs).
		Order("release_date ASC NULLS LAST, name ASC").
		Find(&sets).
		Error
	return sets, err
}

// ListUngroupedExpansionSets returns every Expansion Set whose SeriesID is
// nil, ordered the same way ListExpansionSets orders each Series's sets: by
// release date (sets with an unknown release date sort last) then name. A
// series-less Expansion Set is a legitimate domain state (see CONTEXT.md's
// Series entry), not an edge case to special-case away - this is how it's
// surfaced through GET /catalog/series alongside the grouped Series.
func (r *catalogRepository) ListUngroupedExpansionSets(ctx context.Context) ([]entity.ExpansionSet, error) {
	var sets []entity.ExpansionSet
	err := r.db.WithContext(ctx).
		Where("series_id IS NULL").
		Order("release_date ASC NULLS LAST, name ASC").
		Find(&sets).
		Error
	return sets, err
}

// ListRarities returns every Rarity across all Games, ordered by name.
func (r *catalogRepository) ListRarities(ctx context.Context) ([]entity.Rarity, error) {
	var rarities []entity.Rarity
	err := r.db.WithContext(ctx).Order("name ASC").Find(&rarities).Error
	return rarities, err
}

// ListDistinctCategories returns the distinct Card.Category values actually
// in use, ordered alphabetically.
func (r *catalogRepository) ListDistinctCategories(ctx context.Context) ([]string, error) {
	var categories []string
	err := r.db.WithContext(ctx).
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
	var tags []string
	err := r.db.WithContext(ctx).
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
	Name            string
	ExpansionSetIDs []uuid.UUID
	LocalID         string
	RarityIDs       []uuid.UUID
	Categories      []string
	Tags            []string
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
	ImageURL                string
	RarityID                uuid.UUID
	RarityCode              string
	RarityName              string
	ExpansionSetID          uuid.UUID
	ExpansionSetCode        string
	ExpansionSetName        string
	ExpansionSetReleaseDate *time.Time
}

const cardResultColumns = `cards.id AS id,
	cards.local_id AS local_id,
	cards.name AS name,
	cards.category AS category,
	cards.illustrator AS illustrator,
	cards.tags AS tags,
	cards.image_url AS image_url,
	cards.rarity_id AS rarity_id,
	rarities.code AS rarity_code,
	rarities.name AS rarity_name,
	cards.expansion_set_id AS expansion_set_id,
	expansion_sets.code AS expansion_set_code,
	expansion_sets.name AS expansion_set_name,
	expansion_sets.release_date AS expansion_set_release_date`

// likeEscaper escapes ILIKE's own wildcard characters (%, _) and its default
// escape character (\) in a single pass, so a literal % or _ in user input
// can't widen a search beyond what the user typed (e.g. searching "A_B"
// matching "AxB" too, since _ means "any one character" unless escaped).
var likeEscaper = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`)

// applyCardFilters adds filter's non-empty facets as WHERE conditions to
// query, which must be scoped to a query over the cards table (the
// conditions only touch cards columns). Passing a narrower query (e.g. one
// limited to a Collection's Cards) scopes the conditions, and so the facets,
// to that base set.
func applyCardFilters(query *gorm.DB, filter CardFilter) *gorm.DB {
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
		conds := make([]string, len(filter.Tags))
		args := make([]any, len(filter.Tags))
		for i, tag := range filter.Tags {
			conds[i] = "cards.tags @> jsonb_build_array(?::text)"
			args[i] = tag
		}
		query = query.Where("("+strings.Join(conds, " OR ")+")", args...)
	}

	return query
}

// SearchCards returns the Cards matching filter (joined with their Rarity
// and Expansion Set), limited/offset per filter, plus the total number of
// Cards matching filter before that pagination.
func (r *catalogRepository) SearchCards(ctx context.Context, filter CardFilter) ([]CardResult, int64, error) {
	base := r.db.WithContext(ctx).
		Table("cards").
		Joins("JOIN rarities ON rarities.id = cards.rarity_id").
		Joins("JOIN expansion_sets ON expansion_sets.id = cards.expansion_set_id")

	base = applyCardFilters(base, filter)

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var results []CardResult
	err := base.Session(&gorm.Session{}).
		Select(cardResultColumns).
		Order("expansion_sets.release_date ASC NULLS LAST, expansion_sets.id ASC, cards.local_id ASC, cards.name ASC, cards.id ASC").
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
	// without is where a narrower base set (ticket 22's Collection) would
	// plug in: swap the starting query for one limited to those Cards.
	without := func(mutate func(*CardFilter)) *gorm.DB {
		f := filter
		mutate(&f)
		return applyCardFilters(r.db.WithContext(ctx).Table("cards"), f)
	}

	var facets CardFacets

	setIDs, err := pluckDistinct[uuid.UUID](without(func(f *CardFilter) { f.ExpansionSetIDs = nil }), "cards.expansion_set_id")
	if err != nil {
		return CardFacets{}, err
	}
	var sets []entity.ExpansionSet
	if ids := unionIDs(setIDs, filter.ExpansionSetIDs); len(ids) > 0 {
		if err := r.db.WithContext(ctx).Where("id IN ?", ids).
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
		if err := r.db.WithContext(ctx).Where("id IN ?", ids).Order("name ASC").Find(&rarities).Error; err != nil {
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
	options := make([]StringFacetOption, len(all))
	for i, v := range all {
		options[i] = StringFacetOption{Value: v, Available: slices.Contains(available, v)}
	}
	return options
}
