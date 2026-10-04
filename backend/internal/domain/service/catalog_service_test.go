package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/apperr"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testImageBase = "https://img.example.test"

var testImages = mapper.NewImageHost(testImageBase)

func TestCatalogService_ListSeries_NestsExpansionSets(t *testing.T) {
	seriesID := uuid.New()
	otherSeriesID := uuid.New()
	setID := uuid.New()
	ungroupedSetID := uuid.New()
	ctx := context.Background()

	series := []entity.Series{
		{BaseEntity: baseEntity(seriesID), Code: "sv", Name: "Scarlet & Violet"},
		{BaseEntity: baseEntity(otherSeriesID), Code: "empty", Name: "No Sets Yet"},
	}
	sets := []entity.ExpansionSet{
		{BaseEntity: baseEntity(setID), SeriesID: &seriesID, Code: "sv1", Name: "Scarlet ex"},
	}
	ungrouped := []entity.ExpansionSet{
		{BaseEntity: baseEntity(ungroupedSetID), Code: "promo", Name: "Promo Set"},
	}

	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().ListSeries(ctx).Return(series, nil).Once()
	repo.EXPECT().ListExpansionSets(ctx, []uuid.UUID{seriesID, otherSeriesID}).Return(sets, nil).Once()
	repo.EXPECT().ListUngroupedExpansionSets(ctx).Return(ungrouped, nil).Once()
	svc := NewCatalogService(repo, testImages)

	got, err := svc.ListSeries(ctx)
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
	if len(got.UngroupedExpansionSets) != 1 || got.UngroupedExpansionSets[0].ID != ungroupedSetID {
		t.Fatalf("expected the series-less expansion set to be surfaced separately, got %+v", got.UngroupedExpansionSets)
	}
}

func TestCatalogService_ListSeries_PropagatesRepositoryError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("boom")

	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().ListSeries(ctx).Return(nil, wantErr).Once()
	svc := NewCatalogService(repo, testImages)

	_, err := svc.ListSeries(ctx)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestCatalogService_ListSeries_PropagatesUngroupedExpansionSetsError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("boom")

	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().ListSeries(ctx).Return(nil, nil).Once()
	repo.EXPECT().ListExpansionSets(ctx, []uuid.UUID{}).Return(nil, nil).Once()
	repo.EXPECT().ListUngroupedExpansionSets(ctx).Return(nil, wantErr).Once()
	svc := NewCatalogService(repo, testImages)

	_, err := svc.ListSeries(ctx)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestCatalogService_ListRarities(t *testing.T) {
	ctx := context.Background()
	rarityID := uuid.New()

	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().ListRarities(ctx).Return([]entity.Rarity{{BaseEntity: baseEntity(rarityID), Code: "SR", Name: "Super Rare"}}, nil).Once()
	svc := NewCatalogService(repo, testImages)

	got, err := svc.ListRarities(ctx)
	if err != nil {
		t.Fatalf("ListRarities: %v", err)
	}
	want := []dto.RaritySummary{{ID: rarityID, Code: "SR", Name: "Super Rare"}}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("expected %+v, got %+v", want, got)
	}
}

func TestCatalogService_ListCategoriesAndTags(t *testing.T) {
	ctx := context.Background()

	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().ListDistinctCategories(ctx).Return([]string{"Pokémon", "Trainer"}, nil).Once()
	repo.EXPECT().ListDistinctTags(ctx).Return([]string{"Basic", "Stage 1"}, nil).Once()
	svc := NewCatalogService(repo, testImages)

	categories, err := svc.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(categories) != 2 || categories[0] != "Pokémon" {
		t.Fatalf("expected repo's categories passed through, got %v", categories)
	}

	tags, err := svc.ListTags(ctx)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 2 || tags[1] != "Stage 1" {
		t.Fatalf("expected repo's tags passed through, got %v", tags)
	}
}

func TestCatalogService_SearchCards_MapsResultsAndNormalizesPagination(t *testing.T) {
	ctx := context.Background()
	cardID := uuid.New()
	setID := uuid.New()
	rarityID := uuid.New()

	searchResults := []repository.CardResult{
		{
			ID:                   cardID,
			LocalID:              "001",
			Name:                 "Pikachu",
			Category:             "Pokémon",
			Illustrator:          "Someone",
			Tags:                 []string{"Basic"},
			ImageKey:             "cards/pikachu",
			RarityID:             rarityID,
			RarityCode:           "C",
			RarityName:           "Common",
			ExpansionSetID:       setID,
			ExpansionSetCode:     "sv1",
			ExpansionSetName:     "Scarlet ex",
			ExpansionSetImageKey: "expansion-sets/sv1",
		},
	}

	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().
		SearchCards(ctx, repository.CardFilter{Name: "pika", Limit: defaultCardSearchLimit, Offset: 0}).
		Return(searchResults, 1, nil).
		Once()
	svc := NewCatalogService(repo, testImages)

	// Page/limit both unset (zero value) - should be normalized to page 1,
	// the default limit.
	cards, meta, err := svc.SearchCards(ctx, dto.CardFilter{Name: "pika"})
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}

	if meta.Page != 1 || meta.Limit != defaultCardSearchLimit || meta.Total != 1 {
		t.Fatalf("expected normalized Page=1 Limit=%d Total=1, got %+v", defaultCardSearchLimit, meta)
	}
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}

	want := dto.CardSummary{
		ID: cardID,
		ExpansionSet: dto.ExpansionSetSummary{
			ID:       setID,
			Code:     "sv1",
			Name:     "Scarlet ex",
			ImageURL: testImageBase + "/expansion-sets/sv1",
		},
		LocalID:  "001",
		Name:     "Pikachu",
		Category: "Pokémon",
		Tags:     []string{"Basic"},
		Rarity: dto.RaritySummary{
			ID:   rarityID,
			Code: "C",
			Name: "Common",
		},
		Illustrator: "Someone",
		ImageURL:    testImageBase + "/cards/pikachu",
	}
	if cards[0].ID != want.ID ||
		cards[0].ExpansionSet != want.ExpansionSet ||
		cards[0].LocalID != want.LocalID ||
		cards[0].ImageURL != want.ImageURL ||
		cards[0].Rarity != want.Rarity {
		t.Fatalf("expected mapped card %+v, got %+v", want, cards[0])
	}
}

func TestCatalogService_SearchCards_ClampsLimitAndComputesOffset(t *testing.T) {
	ctx := context.Background()

	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().
		SearchCards(ctx, repository.CardFilter{Limit: maxCardSearchLimit, Offset: 2 * maxCardSearchLimit}).
		Return(nil, 0, nil).
		Once()
	svc := NewCatalogService(repo, testImages)

	_, _, err := svc.SearchCards(ctx, dto.CardFilter{Page: 3, Limit: 1000})
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}
}

func TestCatalogService_SearchCards_NegativePageDefaultsToOne(t *testing.T) {
	ctx := context.Background()

	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().
		SearchCards(ctx, repository.CardFilter{Limit: defaultCardSearchLimit, Offset: 0}).
		Return(nil, 0, nil).
		Once()
	svc := NewCatalogService(repo, testImages)

	_, _, err := svc.SearchCards(ctx, dto.CardFilter{Page: -5})
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}
}

func TestCatalogService_SearchCards_PropagatesRepositoryError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("boom")

	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().
		SearchCards(ctx, repository.CardFilter{Limit: defaultCardSearchLimit, Offset: 0}).
		Return(nil, 0, wantErr).
		Once()
	svc := NewCatalogService(repo, testImages)

	_, _, err := svc.SearchCards(ctx, dto.CardFilter{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func baseEntity(id uuid.UUID) crud.BaseEntity {
	return crud.BaseEntity{ID: id}
}

func TestCatalogService_SearchCards_PassesMultiValueFilter(t *testing.T) {
	ctx := context.Background()
	setA, setB, rarityID := uuid.New(), uuid.New(), uuid.New()

	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().
		SearchCards(ctx, repository.CardFilter{
			ExpansionSetIDs: []uuid.UUID{setA, setB},
			RarityIDs:       []uuid.UUID{rarityID},
			Categories:      []string{"Pokémon", "Trainer"},
			Tags:            []string{"Basic", "ex"},
			Limit:           defaultCardSearchLimit,
		}).
		Return(nil, 0, nil).
		Once()
	svc := NewCatalogService(repo, testImages)

	_, _, err := svc.SearchCards(ctx, dto.CardFilter{
		ExpansionSetIDs: []uuid.UUID{setA, setB},
		RarityIDs:       []uuid.UUID{rarityID},
		Categories:      []string{"Pokémon", "Trainer"},
		Tags:            []string{"Basic", "ex"},
	})
	require.NoError(t, err)
}

func TestCatalogService_ListFacets_MapsOptionsAndAvailability(t *testing.T) {
	ctx := context.Background()
	setID, seriesID, rarityID := uuid.New(), uuid.New(), uuid.New()
	filter := dto.CardFilter{ExpansionSetIDs: []uuid.UUID{setID}, Tags: []string{"gone"}}

	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().
		ListCardFacets(ctx, repository.CardFilter{ExpansionSetIDs: []uuid.UUID{setID}, Tags: []string{"gone"}}).
		Return(repository.CardFacets{
			ExpansionSets: []repository.ExpansionSetFacetOption{{
				ExpansionSet: entity.ExpansionSet{BaseEntity: baseEntity(setID), Code: "s1", Name: "Set 1", SeriesID: &seriesID},
				Available:    true,
			}},
			Rarities:   []repository.RarityFacetOption{{Rarity: entity.Rarity{BaseEntity: baseEntity(rarityID), Code: "SR", Name: "Super Rare"}}},
			Categories: []repository.StringFacetOption{{Value: "Trainer", Available: true}},
			Tags:       []repository.StringFacetOption{{Value: "gone", Available: false}},
		}, nil).
		Once()
	svc := NewCatalogService(repo, testImages)

	got, err := svc.ListFacets(ctx, filter)
	require.NoError(t, err)

	assert.Equal(t, []dto.ExpansionSetFacetOption{{
		ExpansionSetSummary: dto.ExpansionSetSummary{ID: setID, Code: "s1", Name: "Set 1"},
		SeriesID:            &seriesID,
		Available:           true,
	}}, got.ExpansionSets)
	assert.Equal(t, []dto.RarityFacetOption{{RaritySummary: dto.RaritySummary{ID: rarityID, Code: "SR", Name: "Super Rare"}}}, got.Rarities)
	assert.Equal(t, []dto.StringFacetOption{{Value: "Trainer", Available: true}}, got.Categories)
	assert.Equal(t, []dto.StringFacetOption{{Value: "gone", Available: false}}, got.Tags)
}

func TestCatalogService_ListFacets_PropagatesRepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().ListCardFacets(ctx, repository.CardFilter{}).Return(repository.CardFacets{}, errors.New("boom")).Once()

	_, err := NewCatalogService(repo, testImages).ListFacets(ctx, dto.CardFilter{})
	assert.EqualError(t, err, "boom")
}

func TestCatalogService_ListFacets_ZeroMatchesYieldsEmptySlices(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().ListCardFacets(ctx, repository.CardFilter{}).Return(repository.CardFacets{}, nil).Once()

	got, err := NewCatalogService(repo, testImages).ListFacets(ctx, dto.CardFilter{})
	require.NoError(t, err)
	assert.NotNil(t, got.ExpansionSets)
	assert.NotNil(t, got.Rarities)
	assert.NotNil(t, got.Categories)
	assert.NotNil(t, got.Tags)
}

func TestCatalogService_GuestSearch_IsOnePageOfTheStandardSize(t *testing.T) {
	ctx := context.Background()

	for name, requested := range map[string]int{"omitted": 0, "above the standard page": 100, "the standard page": defaultCardSearchLimit} {
		t.Run(name, func(t *testing.T) {
			repo := mocks.NewMockCatalogRepository(t)
			repo.EXPECT().
				SearchCards(ctx, repository.CardFilter{Name: "pika", Limit: defaultCardSearchLimit, Offset: 0}).
				Return(nil, 80, nil).
				Once()

			_, meta, err := NewCatalogService(repo, testImages).SearchCards(ctx, dto.CardFilter{Name: "pika", Limit: requested, Guest: true})

			require.NoError(t, err)
			assert.Equal(t, dto.PaginationMeta{Total: 80, Page: 1, Limit: defaultCardSearchLimit}, meta, "the true total, not the capped one")
		})
	}
}

func TestCatalogService_GuestSearch_KeepsASmallerPageSizeAndTheOpenFilters(t *testing.T) {
	ctx := context.Background()
	setID := uuid.New()

	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().
		SearchCards(ctx, repository.CardFilter{Name: "pika", ExpansionSetIDs: []uuid.UUID{setID}, LocalID: "001", Limit: 5, Offset: 0}).
		Return(nil, 0, nil).
		Once()

	_, _, err := NewCatalogService(repo, testImages).SearchCards(ctx, dto.CardFilter{
		Name: "pika", ExpansionSetIDs: []uuid.UUID{setID}, LocalID: "001", Page: 1, Limit: 5, Guest: true,
	})

	require.NoError(t, err)
}

func TestCatalogService_GuestSearch_RefusesWhatIsLocked(t *testing.T) {
	ctx := context.Background()

	locked := map[string]dto.CardFilter{
		"a later page": {Page: 2},
		"a rarity":     {RarityIDs: []uuid.UUID{uuid.New()}},
		"a category":   {Categories: []string{"Trainer"}},
		"a tag":        {Tags: []string{"ex"}},
	}
	for name, filter := range locked {
		t.Run(name, func(t *testing.T) {
			filter.Guest = true
			repo := mocks.NewMockCatalogRepository(t) // no expectation: the repository is never reached

			_, _, err := NewCatalogService(repo, testImages).SearchCards(ctx, filter)

			require.Error(t, err)
			assert.Equal(t, apperr.CodeLoginRequired, apperr.CodeOf(err))
			appErr, ok := errors.AsType[ungerr.AppError](err)
			require.True(t, ok)
			assert.Equal(t, http.StatusUnauthorized, appErr.HttpStatus())
		})
	}
}

func TestCatalogService_SignedInSearch_IsNotLocked(t *testing.T) {
	ctx := context.Background()
	rarityID := uuid.New()

	repo := mocks.NewMockCatalogRepository(t)
	repo.EXPECT().
		SearchCards(ctx, repository.CardFilter{
			RarityIDs: []uuid.UUID{rarityID}, Categories: []string{"Trainer"}, Tags: []string{"ex"},
			Limit: maxCardSearchLimit, Offset: maxCardSearchLimit,
		}).
		Return(nil, 0, nil).
		Once()

	_, _, err := NewCatalogService(repo, testImages).SearchCards(ctx, dto.CardFilter{
		RarityIDs: []uuid.UUID{rarityID}, Categories: []string{"Trainer"}, Tags: []string{"ex"}, Page: 2, Limit: 1000,
	})

	require.NoError(t, err)
}

func TestCatalogService_ListFacets_RefusesAGuest(t *testing.T) {
	repo := mocks.NewMockCatalogRepository(t) // no expectation: the repository is never reached

	_, err := NewCatalogService(repo, testImages).ListFacets(context.Background(), dto.CardFilter{Guest: true})

	require.Error(t, err)
	assert.Equal(t, apperr.CodeLoginRequired, apperr.CodeOf(err))
}
