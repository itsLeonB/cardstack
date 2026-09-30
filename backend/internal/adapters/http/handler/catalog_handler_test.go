package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	"github.com/stretchr/testify/mock"
)

func newTestCatalogHandler(t *testing.T) (*mocks.MockCatalogService, humatest.TestAPI) {
	t.Helper()

	svc := mocks.NewMockCatalogService(t)
	_, api := humatest.New(t, httpapi.NewConfig())
	endpoint.RegisterAll(api, NewCatalogHandler(svc).Routes())

	return svc, api
}

func TestCatalogHandler_ListSeries(t *testing.T) {
	svc, api := newTestCatalogHandler(t)
	svc.EXPECT().ListSeries(mock.Anything).Return(dto.SeriesBrowseResult{
		Series:                 []dto.SeriesSummary{{Code: "sv", Name: "Scarlet & Violet"}},
		UngroupedExpansionSets: []dto.ExpansionSetSummary{{Code: "promo", Name: "Promo Set"}},
	}, nil)

	resp := api.Get("/catalog/series")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body struct {
		Data dto.SeriesBrowseResult `json:"data"`
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
	svc, api := newTestCatalogHandler(t)
	svc.EXPECT().ListRarities(mock.Anything).Return([]dto.RaritySummary{{Code: "SR", Name: "Super Rare"}}, nil)

	resp := api.Get("/catalog/rarities")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCatalogHandler_ListCategories(t *testing.T) {
	svc, api := newTestCatalogHandler(t)
	svc.EXPECT().ListCategories(mock.Anything).Return([]string{"Pokémon", "Trainer"}, nil)

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
	svc, api := newTestCatalogHandler(t)
	svc.EXPECT().ListTags(mock.Anything).Return([]string{"Basic"}, nil)

	resp := api.Get("/catalog/tags")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCatalogHandler_SearchCards_DefaultsPageAndLimit(t *testing.T) {
	svc, api := newTestCatalogHandler(t)
	var got dto.CardFilter
	svc.EXPECT().SearchCards(mock.Anything, mock.Anything).
		Run(func(_ context.Context, filter dto.CardFilter) { got = filter }).
		Return(nil, dto.PaginationMeta{}, nil)

	resp := api.Get("/catalog/cards")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if got.Page != 1 || got.Limit != 24 {
		t.Fatalf("expected default Page=1 Limit=24, got %+v", got)
	}
}

func TestCatalogHandler_SearchCards_ParsesFilters(t *testing.T) {
	svc, api := newTestCatalogHandler(t)

	expansionSetID := uuid.New()
	rarityID := uuid.New()

	want := dto.CardFilter{
		Name:           "pika",
		ExpansionSetID: expansionSetID,
		LocalID:        "001",
		RarityID:       rarityID,
		Category:       "Pokémon",
		Tag:            "Basic",
		Page:           2,
		Limit:          10,
	}
	svc.EXPECT().SearchCards(mock.Anything, want).Return(nil, dto.PaginationMeta{}, nil)

	resp := api.Get("/catalog/cards?name=pika&expansionSetId=" + expansionSetID.String() +
		"&localId=001&rarityId=" + rarityID.String() + "&category=Pok%C3%A9mon&tag=Basic&page=2&limit=10")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCatalogHandler_SearchCards_InvalidExpansionSetID(t *testing.T) {
	_, api := newTestCatalogHandler(t)

	resp := api.Get("/catalog/cards?expansionSetId=not-a-uuid")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCatalogHandler_SearchCards_InvalidRarityID(t *testing.T) {
	_, api := newTestCatalogHandler(t)

	resp := api.Get("/catalog/cards?rarityId=not-a-uuid")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCatalogHandler_SearchCards_ReturnsResult(t *testing.T) {
	cardID := uuid.New()
	svc, api := newTestCatalogHandler(t)
	svc.EXPECT().SearchCards(mock.Anything, mock.Anything).
		Return([]dto.CardSummary{{ID: cardID, Name: "Pikachu"}}, dto.PaginationMeta{Total: 1, Page: 1, Limit: 24}, nil)

	resp := api.Get("/catalog/cards")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body struct {
		Data []dto.CardSummary  `json:"data"`
		Meta dto.PaginationMeta `json:"meta"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshaling response body: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].ID != cardID || body.Data[0].Name != "Pikachu" {
		t.Fatalf("expected data to hold the cards directly, got %+v", body.Data)
	}
	if body.Meta != (dto.PaginationMeta{Total: 1, Page: 1, Limit: 24}) {
		t.Fatalf("expected meta %+v, got %+v", dto.PaginationMeta{Total: 1, Page: 1, Limit: 24}, body.Meta)
	}
}

func trimTrailingNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func TestCatalogHandler_UnclassifiedErrorsAreRedacted(t *testing.T) {
	leak := errors.New("failed to connect: postgres://admin:s3cr3t@dbhost:5432/cards")
	svc, api := newTestCatalogHandler(t)
	svc.EXPECT().ListSeries(mock.Anything).Return(dto.SeriesBrowseResult{}, leak)
	svc.EXPECT().SearchCards(mock.Anything, mock.Anything).Return(nil, dto.PaginationMeta{}, leak)

	for _, path := range []string{"/catalog/series", "/catalog/cards"} {
		resp := api.Get(path)
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("%s: expected 500, got %d", path, resp.Code)
		}
		if body := resp.Body.String(); strings.Contains(body, "s3cr3t") || strings.Contains(body, "dbhost") {
			t.Fatalf("%s leaks internal detail: %s", path, body)
		}
	}
}
