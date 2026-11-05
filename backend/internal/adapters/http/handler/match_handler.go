package handler

import (
	"context"
	"net/http"

	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
)

// maxMatchImageBytes caps the upload: the browser sends a JPEG of about 1024
// pixels on the long side, which is a few hundred kilobytes.
const maxMatchImageBytes = 2 << 20

// MatchHandler serves the authenticated scan match route.
type MatchHandler struct {
	matchSvc service.MatchService
}

func NewMatchHandler(matchSvc service.MatchService) *MatchHandler {
	return &MatchHandler{matchSvc: matchSvc}
}

// ScanFilterParams are the optional Scan Filter query parameters shared by the
// scan lookups. Empty means no restriction.
type ScanFilterParams struct {
	ExpansionSetIDs []string `query:"expansionSetId,explode" maxItems:"20" doc:"Only Cards in any of these Expansion Sets (repeatable, at most 20)."`
	RarityIDs       []string `query:"rarityId,explode" maxItems:"20" doc:"Only Cards with any of these Rarities (repeatable, at most 20)."`
}

// parseScanFilter validates the filter ids up front, so a malformed one is a
// 400 before any query runs.
func parseScanFilter(p ScanFilterParams) (dto.ScanFilter, error) {
	setIDs, err := parseUUIDs("expansionSetId", p.ExpansionSetIDs)
	if err != nil {
		return dto.ScanFilter{}, err
	}
	rarityIDs, err := parseUUIDs("rarityId", p.RarityIDs)
	if err != nil {
		return dto.ScanFilter{}, err
	}
	return dto.ScanFilter{ExpansionSetIDs: setIDs, RarityIDs: rarityIDs}, nil
}

// matchInput is the photo itself as the request body, plus the optional Scan
// Filter query. Huma declares the content type in the OpenAPI document but does
// not enforce it; the service checks the bytes.
type matchInput struct {
	ScanFilterParams
	RawBody []byte `contentType:"image/jpeg" doc:"One card photo as a JPEG, at most 2 MiB. It is processed in memory and never stored."`
}

func (h *MatchHandler) match(ctx context.Context, in matchInput) (dto.MatchResult, error) {
	filter, err := parseScanFilter(in.ScanFilterParams)
	if err != nil {
		return dto.MatchResult{}, err
	}
	return h.matchSvc.Match(ctx, dto.MatchRequest{ScanFilter: filter, Image: in.RawBody})
}

// reprintsInput is GET /scan/reprints' query: the Card ids the client holds, in
// the order the matcher offered them, and the page to start at.
type reprintsInput struct {
	ScanFilterParams
	CardIDs []string `query:"cardIds,explode" maxItems:"5" doc:"The Cards the client holds, in the order the matcher offered them (repeatable, at most five). Each one's Reprints are listed in that order, and every id sent is left out of the result."`
	Offset  int      `query:"offset" minimum:"0" doc:"How many Reprints to skip: the next page is at offset plus five. Use the same ids with the same offset to get the same page."`
}

func (h *MatchHandler) reprints(ctx context.Context, in reprintsInput) (dto.ReprintResult, error) {
	ids, err := parseUUIDs("cardIds", in.CardIDs)
	if err != nil {
		return dto.ReprintResult{}, err
	}
	filter, err := parseScanFilter(in.ScanFilterParams)
	if err != nil {
		return dto.ReprintResult{}, err
	}
	return h.matchSvc.Reprints(ctx, dto.ReprintRequest{ScanFilter: filter, CardIDs: ids, Offset: in.Offset})
}

// Routes returns every route MatchHandler exposes, for registration via
// endpoint.RegisterAll.
func (h *MatchHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[matchInput, dto.MatchResult]{
			OperationID:  "match-scanned-card",
			Method:       http.MethodPost,
			Path:         "/scan/match",
			Summary:      "Match one card photo against the catalog, optionally limited to Expansion Sets and Rarities. Returns ranked candidates and whether the server is confident in the first; an empty or non-JPEG body is 400 and a body over 2 MiB is 413",
			Tags:         []string{"scan"},
			SuccessCode:  http.StatusOK,
			Secured:      true,
			MaxBodyBytes: maxMatchImageBytes,
			HandlerFunc:  h.match,
		}),
		endpoint.New(endpoint.Endpoint[reprintsInput, dto.ReprintResult]{
			OperationID: "list-scan-reprints",
			Method:      http.MethodGet,
			Path:        "/scan/reprints",
			Summary:     "List the Reprints of up to five Cards the client holds, optionally limited to Expansion Sets and Rarities: other prints of the same name and artwork, five per page. Returns hasMore while more remain; an invalid id is 400, more than five ids or a negative offset is 422",
			Tags:        []string{"scan"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.reprints,
		}),
	}
}
