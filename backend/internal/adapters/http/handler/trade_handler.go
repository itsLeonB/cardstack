package handler

import (
	"context"
	"net/http"

	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
)

type TradeHandler struct {
	tradeSvc service.TradeService
}

func NewTradeHandler(tradeSvc service.TradeService) *TradeHandler {
	return &TradeHandler{tradeSvc: tradeSvc}
}

// tradeSlugInput is POST /trades: an empty body; the caller's profile is the
// owner seat.
type tradeSlugInput struct{}

// tradeLookupInput is GET /trades/{slug}: read the room by its link.
type tradeLookupInput struct {
	Slug string `path:"slug" doc:"The Trade's Slug, the last segment of its link."`
}

// tradeJoinInput is POST /trades/{slug}/join: claim the second seat.
type tradeJoinInput struct {
	Slug string `path:"slug" doc:"The Trade's Slug, the last segment of its link."`
}

func (h *TradeHandler) create(ctx context.Context, _ tradeSlugInput) (dto.TradeSlug, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.TradeSlug{}, err
	}

	return h.tradeSvc.Create(ctx, dto.TradeCreateRequest{ProfileID: profileID})
}

func (h *TradeHandler) get(ctx context.Context, in tradeLookupInput) (dto.TradeView, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.TradeView{}, err
	}

	return h.tradeSvc.Get(ctx, dto.TradeLookup{Slug: in.Slug}, profileID)
}

func (h *TradeHandler) join(ctx context.Context, in tradeJoinInput) (dto.TradeView, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.TradeView{}, err
	}

	return h.tradeSvc.Join(ctx, dto.TradeJoinRequest{ProfileID: profileID, Slug: in.Slug})
}

// Routes are private: a Guest gets 401, never the trade page (ADR-0028). The
// slug is the whole address; there is no list or owner-scoped read here.
func (h *TradeHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[tradeSlugInput, dto.TradeSlug]{
			OperationID: "create-trade",
			Method:      http.MethodPost,
			Path:        "/trades",
			Summary:     "Start a Trade and get its link Slug; the caller takes the owner seat in state open",
			Tags:        []string{"trades"},
			SuccessCode: http.StatusCreated,
			HandlerFunc: h.create,
		}),
		endpoint.New(endpoint.Endpoint[tradeLookupInput, dto.TradeView]{
			OperationID: "get-trade",
			Method:      http.MethodGet,
			Path:        "/trades/{slug}",
			Summary:     "Read a Trade by its slug: state, both display names and the Trade Items on each side, without claiming the seat. An unknown, cancelled or not-yours slug is the same 404",
			Tags:        []string{"trades"},
			SuccessCode: http.StatusOK,
			HandlerFunc: h.get,
		}),
		endpoint.New(endpoint.Endpoint[tradeJoinInput, dto.TradeView]{
			OperationID: "join-trade",
			Method:      http.MethodPost,
			Path:        "/trades/{slug}/join",
			Summary:     "Take the Trade's second seat once, moving it to negotiating; a second join gets 409 trade_seat_taken",
			Tags:        []string{"trades"},
			SuccessCode: http.StatusOK,
			HandlerFunc: h.join,
		}),
	}
}
