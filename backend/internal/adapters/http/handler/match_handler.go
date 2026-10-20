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

// matchInput is the photo itself as the request body. Huma declares the
// content type in the OpenAPI document but does not enforce it; the service
// checks the bytes.
type matchInput struct {
	RawBody []byte `contentType:"image/jpeg" doc:"One card photo as a JPEG, at most 2 MiB. It is processed in memory and never stored."`
}

func (h *MatchHandler) match(ctx context.Context, in matchInput) (dto.MatchResult, error) {
	return h.matchSvc.Match(ctx, dto.MatchRequest{Image: in.RawBody})
}

// reprintsInput is GET /scan/reprints' query: the Card ids the client holds, in
// the order the matcher offered them, and the page to start at.
type reprintsInput struct {
	CardIDs []string `query:"cardId,explode" maxItems:"5" doc:"The Cards the client holds, in the order the matcher offered them (repeatable, at most five). Each one's Reprints are listed in that order, and every id sent is left out of the result."`
	Offset  int      `query:"offset" minimum:"0" doc:"How many Reprints to skip: the next page is at offset plus five. Use the same ids with the same offset to get the same page."`
}

func (h *MatchHandler) reprints(ctx context.Context, in reprintsInput) (dto.ReprintResult, error) {
	ids, err := parseUUIDs("cardId", in.CardIDs)
	if err != nil {
		return dto.ReprintResult{}, err
	}
	return h.matchSvc.Reprints(ctx, dto.ReprintRequest{CardIDs: ids, Offset: in.Offset})
}

// Routes returns every route MatchHandler exposes, for registration via
// endpoint.RegisterAll.
func (h *MatchHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[matchInput, dto.MatchResult]{
			OperationID:  "match-scanned-card",
			Method:       http.MethodPost,
			Path:         "/scan/match",
			Summary:      "Match one card photo against the catalog. Returns ranked candidates and whether the server is confident in the first; an empty or non-JPEG body is 400 and a body over 2 MiB is 413",
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
			Summary:     "List the Reprints of up to five Cards the client holds: other Expansion Sets' prints of the same name and artwork, five per page. Returns hasMore while more remain; an invalid id is 400, more than five ids or a negative offset is 422",
			Tags:        []string{"scan"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.reprints,
		}),
	}
}
