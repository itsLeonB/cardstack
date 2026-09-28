package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
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

// CardFilter narrows CatalogRepository.SearchCards. The zero value of each
// field means "don't filter on this facet" (mirrors service.CardFilter,
// which this sits behind).
type CardFilter struct {
	Name           string
	ExpansionSetID uuid.UUID
	LocalID        string
	RarityID       uuid.UUID
	Category       string
	Tag            string
	Limit          int
	Offset         int
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

// applyCardFilters adds filter's non-zero facets as WHERE conditions to
// query, which must already be scoped to the joined cards/rarities/
// expansion_sets query SearchCards builds.
func applyCardFilters(query *gorm.DB, filter CardFilter) *gorm.DB {
	if filter.Name != "" {
		query = query.Where("cards.name ILIKE ?", "%"+likeEscaper.Replace(filter.Name)+"%")
	}
	if filter.ExpansionSetID != uuid.Nil {
		query = query.Where("cards.expansion_set_id = ?", filter.ExpansionSetID)
	}
	if filter.LocalID != "" {
		query = query.Where("cards.local_id = ?", filter.LocalID)
	}
	if filter.RarityID != uuid.Nil {
		query = query.Where("cards.rarity_id = ?", filter.RarityID)
	}
	if filter.Category != "" {
		query = query.Where("cards.category = ?", filter.Category)
	}
	if filter.Tag != "" {
		// cards.tags is JSONB (see entity.Card's doc comment); @> is
		// Postgres's jsonb containment operator, true when the left side
		// contains every element of the right side. json.Marshal of a
		// []string literal can't fail, so its error is ignored rather than
		// threaded through every caller for a case that can't happen.
		tagJSON, _ := json.Marshal([]string{filter.Tag})
		query = query.Where("cards.tags @> ?::jsonb", string(tagJSON))
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
