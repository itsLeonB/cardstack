package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/repository"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	domainservice "github.com/itsLeonB/cardstack/backend/internal/domain/service"
	crud "github.com/itsLeonB/go-crud"
)

// fakeCatalogRepository is an in-memory stand-in for catalogRepository, so
// catalogService's mapping/pagination logic can be unit tested without a
// real Postgres.
type fakeCatalogRepository struct {
	series            []entity.Series
	setsBySeries      map[uuid.UUID][]entity.ExpansionSet
	ungrouped         []entity.ExpansionSet
	rarities          []entity.Rarity
	categories        []string
	tags              []string
	searchResults     []repository.CardResult
	searchTotal       int64
	searchErr         error
	lastSearchFilter  repository.CardFilter
	lastListedSeries  []uuid.UUID
	listExpansionErr  error
	listUngroupedErr  error
	listSeriesErr     error
	listRaritiesErr   error
	listCategoriesErr error
	listTagsErr       error
}

func (f *fakeCatalogRepository) ListSeries(ctx context.Context) ([]entity.Series, error) {
	return f.series, f.listSeriesErr
}

func (f *fakeCatalogRepository) ListExpansionSets(ctx context.Context, seriesIDs []uuid.UUID) ([]entity.ExpansionSet, error) {
	f.lastListedSeries = seriesIDs
	if f.listExpansionErr != nil {
		return nil, f.listExpansionErr
	}

	var sets []entity.ExpansionSet
	for _, id := range seriesIDs {
		sets = append(sets, f.setsBySeries[id]...)
	}
	return sets, nil
}

func (f *fakeCatalogRepository) ListUngroupedExpansionSets(ctx context.Context) ([]entity.ExpansionSet, error) {
	return f.ungrouped, f.listUngroupedErr
}

func (f *fakeCatalogRepository) ListRarities(ctx context.Context) ([]entity.Rarity, error) {
	return f.rarities, f.listRaritiesErr
}

func (f *fakeCatalogRepository) ListDistinctCategories(ctx context.Context) ([]string, error) {
	return f.categories, f.listCategoriesErr
}

func (f *fakeCatalogRepository) ListDistinctTags(ctx context.Context) ([]string, error) {
	return f.tags, f.listTagsErr
}

func (f *fakeCatalogRepository) SearchCards(ctx context.Context, filter repository.CardFilter) ([]repository.CardResult, int64, error) {
	f.lastSearchFilter = filter
	if f.searchErr != nil {
		return nil, 0, f.searchErr
	}
	return f.searchResults, f.searchTotal, nil
}

func TestCatalogService_ListSeries_NestsExpansionSets(t *testing.T) {
	seriesID := uuid.New()
	otherSeriesID := uuid.New()
	setID := uuid.New()
	ungroupedSetID := uuid.New()

	repo := &fakeCatalogRepository{
		series: []entity.Series{
			{BaseEntity: baseEntity(seriesID), Code: "sv", Name: "Scarlet & Violet"},
			{BaseEntity: baseEntity(otherSeriesID), Code: "empty", Name: "No Sets Yet"},
		},
		setsBySeries: map[uuid.UUID][]entity.ExpansionSet{
			seriesID: {
				{BaseEntity: baseEntity(setID), SeriesID: &seriesID, Code: "sv1", Name: "Scarlet ex"},
			},
		},
		ungrouped: []entity.ExpansionSet{
			{BaseEntity: baseEntity(ungroupedSetID), Code: "promo", Name: "Promo Set"},
		},
	}
	svc := NewCatalogService(repo)

	got, err := svc.ListSeries(context.Background())
	if err != nil {
		t.Fatalf("ListSeries: %v", err)
	}
	if len(got.Series) != 2 {
		t.Fatalf("expected 2 series, got %d: %+v", len(got.Series), got.Series)
	}
	if got.Series[0].ID != seriesID || len(got.Series[0].ExpansionSets) != 1 || got.Series[0].ExpansionSets[0].ID != setID {
		t.Fatalf("expected series[0] to nest its expansion set, got %+v", got.Series[0])
	}
	if got.Series[1].ID != otherSeriesID || got.Series[1].ExpansionSets != nil {
		t.Fatalf("expected series[1] to have no expansion sets, got %+v", got.Series[1])
	}
	if len(repo.lastListedSeries) != 2 {
		t.Fatalf("expected ListExpansionSets to be called with both series IDs, got %v", repo.lastListedSeries)
	}
	if len(got.UngroupedExpansionSets) != 1 || got.UngroupedExpansionSets[0].ID != ungroupedSetID {
		t.Fatalf("expected the series-less expansion set to be surfaced separately, got %+v", got.UngroupedExpansionSets)
	}
}

func TestCatalogService_ListSeries_PropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeCatalogRepository{listSeriesErr: wantErr}
	svc := NewCatalogService(repo)

	_, err := svc.ListSeries(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestCatalogService_ListSeries_PropagatesUngroupedExpansionSetsError(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeCatalogRepository{listUngroupedErr: wantErr}
	svc := NewCatalogService(repo)

	_, err := svc.ListSeries(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestCatalogService_ListRarities(t *testing.T) {
	rarityID := uuid.New()
	repo := &fakeCatalogRepository{
		rarities: []entity.Rarity{{BaseEntity: baseEntity(rarityID), Code: "SR", Name: "Super Rare"}},
	}
	svc := NewCatalogService(repo)

	got, err := svc.ListRarities(context.Background())
	if err != nil {
		t.Fatalf("ListRarities: %v", err)
	}
	want := []domainservice.RaritySummary{{ID: rarityID, Code: "SR", Name: "Super Rare"}}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("expected %+v, got %+v", want, got)
	}
}

func TestCatalogService_ListCategoriesAndTags(t *testing.T) {
	repo := &fakeCatalogRepository{
		categories: []string{"Pokémon", "Trainer"},
		tags:       []string{"Basic", "Stage 1"},
	}
	svc := NewCatalogService(repo)

	categories, err := svc.ListCategories(context.Background())
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(categories) != 2 || categories[0] != "Pokémon" {
		t.Fatalf("expected repo's categories passed through, got %v", categories)
	}

	tags, err := svc.ListTags(context.Background())
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 2 || tags[1] != "Stage 1" {
		t.Fatalf("expected repo's tags passed through, got %v", tags)
	}
}

func TestCatalogService_SearchCards_MapsResultsAndNormalizesPagination(t *testing.T) {
	cardID := uuid.New()
	setID := uuid.New()
	rarityID := uuid.New()

	repo := &fakeCatalogRepository{
		searchResults: []repository.CardResult{
			{
				ID:               cardID,
				LocalID:          "001",
				Name:             "Pikachu",
				Category:         "Pokémon",
				Illustrator:      "Someone",
				Tags:             []string{"Basic"},
				ImageURL:         "https://example.com/pikachu.png",
				RarityID:         rarityID,
				RarityCode:       "C",
				RarityName:       "Common",
				ExpansionSetID:   setID,
				ExpansionSetCode: "sv1",
				ExpansionSetName: "Scarlet ex",
			},
		},
		searchTotal: 1,
	}
	svc := NewCatalogService(repo)

	// Page/limit both unset (zero value) - should be normalized to page 1,
	// the default limit.
	got, err := svc.SearchCards(context.Background(), domainservice.CardFilter{Name: "pika"})
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}

	if got.Page != 1 || got.Limit != defaultCardSearchLimit || got.Total != 1 {
		t.Fatalf("expected normalized Page=1 Limit=%d Total=1, got %+v", defaultCardSearchLimit, got)
	}
	if len(got.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(got.Cards))
	}

	want := domainservice.CardSummary{
		ID: cardID,
		ExpansionSet: domainservice.ExpansionSetSummary{
			ID:   setID,
			Code: "sv1",
			Name: "Scarlet ex",
		},
		LocalID:  "001",
		Name:     "Pikachu",
		Category: "Pokémon",
		Tags:     []string{"Basic"},
		Rarity: domainservice.RaritySummary{
			ID:   rarityID,
			Code: "C",
			Name: "Common",
		},
		Illustrator: "Someone",
		ImageURL:    "https://example.com/pikachu.png",
	}
	if got.Cards[0].ID != want.ID ||
		got.Cards[0].ExpansionSet != want.ExpansionSet ||
		got.Cards[0].LocalID != want.LocalID ||
		got.Cards[0].Rarity != want.Rarity {
		t.Fatalf("expected mapped card %+v, got %+v", want, got.Cards[0])
	}

	if repo.lastSearchFilter.Name != "pika" || repo.lastSearchFilter.Limit != defaultCardSearchLimit || repo.lastSearchFilter.Offset != 0 {
		t.Fatalf("expected repo filter Name=pika Limit=%d Offset=0, got %+v", defaultCardSearchLimit, repo.lastSearchFilter)
	}
}

func TestCatalogService_SearchCards_ClampsLimitAndComputesOffset(t *testing.T) {
	repo := &fakeCatalogRepository{}
	svc := NewCatalogService(repo)

	_, err := svc.SearchCards(context.Background(), domainservice.CardFilter{Page: 3, Limit: 1000})
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}

	if repo.lastSearchFilter.Limit != maxCardSearchLimit {
		t.Fatalf("expected limit clamped to %d, got %d", maxCardSearchLimit, repo.lastSearchFilter.Limit)
	}
	if repo.lastSearchFilter.Offset != 2*maxCardSearchLimit {
		t.Fatalf("expected offset %d for page 3, got %d", 2*maxCardSearchLimit, repo.lastSearchFilter.Offset)
	}
}

func TestCatalogService_SearchCards_NegativePageDefaultsToOne(t *testing.T) {
	repo := &fakeCatalogRepository{}
	svc := NewCatalogService(repo)

	_, err := svc.SearchCards(context.Background(), domainservice.CardFilter{Page: -5})
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}

	if repo.lastSearchFilter.Offset != 0 {
		t.Fatalf("expected offset 0 for a negative page, got %d", repo.lastSearchFilter.Offset)
	}
}

func TestCatalogService_SearchCards_PropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeCatalogRepository{searchErr: wantErr}
	svc := NewCatalogService(repo)

	_, err := svc.SearchCards(context.Background(), domainservice.CardFilter{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func baseEntity(id uuid.UUID) crud.BaseEntity {
	return crud.BaseEntity{ID: id}
}
