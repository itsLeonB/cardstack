package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
)

// stubCatalogService is a hand-written service.CatalogService stub that
// records the CardFilter it's called with, so tests can assert on the query
// string -> filter translation without a real repository/DB.
type stubCatalogService struct {
	seriesResult   service.SeriesBrowseResult
	rarities       []service.RaritySummary
	categories     []string
	tags           []string
	searchCards    []service.CardSummary
	searchMeta     service.PaginationMeta
	searchErr      error
	lastFilter     service.CardFilter
	filterCaptured bool
}

func (s *stubCatalogService) ListSeries(context.Context) (service.SeriesBrowseResult, error) {
	return s.seriesResult, nil
}

func (s *stubCatalogService) ListRarities(context.Context) ([]service.RaritySummary, error) {
	return s.rarities, nil
}

func (s *stubCatalogService) ListCategories(context.Context) ([]string, error) {
	return s.categories, nil
}

func (s *stubCatalogService) ListTags(context.Context) ([]string, error) {
	return s.tags, nil
}

func (s *stubCatalogService) SearchCards(_ context.Context, filter service.CardFilter) ([]service.CardSummary, service.PaginationMeta, error) {
	s.lastFilter = filter
	s.filterCaptured = true
	if s.searchErr != nil {
		return nil, service.PaginationMeta{}, s.searchErr
	}
	return s.searchCards, s.searchMeta, nil
}

func newTestCatalogHandler(t *testing.T, stub *stubCatalogService) humatest.TestAPI {
	t.Helper()

	h := NewCatalogHandler(stub)
	_, api := humatest.New(t, httpapi.NewConfig())
	endpoint.RegisterAll(api, h.Routes())

	return api
}

func TestCatalogHandler_ListSeries(t *testing.T) {
	stub := &stubCatalogService{
		seriesResult: service.SeriesBrowseResult{
			Series:                 []service.SeriesSummary{{Code: "sv", Name: "Scarlet & Violet"}},
			UngroupedExpansionSets: []service.ExpansionSetSummary{{Code: "promo", Name: "Promo Set"}},
		},
	}
	api := newTestCatalogHandler(t, stub)

	resp := api.Get("/catalog/series")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body struct {
		Data service.SeriesBrowseResult `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshaling response body: %v", err)
	}
	if len(body.Data.Series) != 1 || body.Data.Series[0].Code != "sv" {
		t.Fatalf("expected series in response, got %+v", body.Data)
	}
	if len(body.Data.UngroupedExpansionSets) != 1 || body.Data.UngroupedExpansionSets[0].Code != "promo" {
		t.Fatalf("expected ungrouped expansion sets in response, got %+v", body.Data)
	}
}

func TestCatalogHandler_ListRarities(t *testing.T) {
	stub := &stubCatalogService{rarities: []service.RaritySummary{{Code: "SR", Name: "Super Rare"}}}
	api := newTestCatalogHandler(t, stub)

	resp := api.Get("/catalog/rarities")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCatalogHandler_ListCategories(t *testing.T) {
	stub := &stubCatalogService{categories: []string{"Pokémon", "Trainer"}}
	api := newTestCatalogHandler(t, stub)

	resp := api.Get("/catalog/categories")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	const want = `{"data":["Pokémon","Trainer"]}`
	if got := trimTrailingNewline(resp.Body.String()); got != want {
		t.Fatalf("expected body %q, got %q", want, got)
	}
}

func TestCatalogHandler_ListTags(t *testing.T) {
	stub := &stubCatalogService{tags: []string{"Basic"}}
	api := newTestCatalogHandler(t, stub)

	resp := api.Get("/catalog/tags")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCatalogHandler_SearchCards_DefaultsPageAndLimit(t *testing.T) {
	stub := &stubCatalogService{}
	api := newTestCatalogHandler(t, stub)

	resp := api.Get("/catalog/cards")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if !stub.filterCaptured {
		t.Fatal("expected SearchCards to be called")
	}
	if stub.lastFilter.Page != 1 || stub.lastFilter.Limit != 24 {
		t.Fatalf("expected default Page=1 Limit=24, got %+v", stub.lastFilter)
	}
}

func TestCatalogHandler_SearchCards_ParsesFilters(t *testing.T) {
	stub := &stubCatalogService{}
	api := newTestCatalogHandler(t, stub)

	expansionSetID := uuid.New()
	rarityID := uuid.New()

	resp := api.Get("/catalog/cards?name=pika&expansionSetId=" + expansionSetID.String() +
		"&localId=001&rarityId=" + rarityID.String() + "&category=Pok%C3%A9mon&tag=Basic&page=2&limit=10")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	want := service.CardFilter{
		Name:           "pika",
		ExpansionSetID: expansionSetID,
		LocalID:        "001",
		RarityID:       rarityID,
		Category:       "Pokémon",
		Tag:            "Basic",
		Page:           2,
		Limit:          10,
	}
	if stub.lastFilter != want {
		t.Fatalf("expected filter %+v, got %+v", want, stub.lastFilter)
	}
}

func TestCatalogHandler_SearchCards_InvalidExpansionSetID(t *testing.T) {
	stub := &stubCatalogService{}
	api := newTestCatalogHandler(t, stub)

	resp := api.Get("/catalog/cards?expansionSetId=not-a-uuid")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
	if stub.filterCaptured {
		t.Fatal("expected SearchCards not to be called for an invalid expansionSetId")
	}
}

func TestCatalogHandler_SearchCards_InvalidRarityID(t *testing.T) {
	stub := &stubCatalogService{}
	api := newTestCatalogHandler(t, stub)

	resp := api.Get("/catalog/cards?rarityId=not-a-uuid")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
	if stub.filterCaptured {
		t.Fatal("expected SearchCards not to be called for an invalid rarityId")
	}
}

func TestCatalogHandler_SearchCards_ReturnsResult(t *testing.T) {
	cardID := uuid.New()
	stub := &stubCatalogService{
		searchCards: []service.CardSummary{{ID: cardID, Name: "Pikachu"}},
		searchMeta:  service.PaginationMeta{Total: 1, Page: 1, Limit: 24},
	}
	api := newTestCatalogHandler(t, stub)

	resp := api.Get("/catalog/cards")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body struct {
		Data []service.CardSummary  `json:"data"`
		Meta service.PaginationMeta `json:"meta"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshaling response body: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].ID != cardID || body.Data[0].Name != "Pikachu" {
		t.Fatalf("expected data to hold the cards directly, got %+v", body.Data)
	}
	if body.Meta != (service.PaginationMeta{Total: 1, Page: 1, Limit: 24}) {
		t.Fatalf("expected meta %+v, got %+v", service.PaginationMeta{Total: 1, Page: 1, Limit: 24}, body.Meta)
	}
}

func trimTrailingNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
