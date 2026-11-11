package handler

import (
	"context"
	"net/http"

	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
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
	// A Guest reads nothing either: the service answers the same not-found as
	// an unknown slug (user story 43), never a 401 that would confirm the link.
	caller := authpkg.CallerFrom(ctx)

	return h.tradeSvc.Get(ctx, dto.TradeLookup{Slug: in.Slug, ProfileID: caller.ProfileID})
}

func (h *TradeHandler) join(ctx context.Context, in tradeJoinInput) (dto.TradeView, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.TradeView{}, err
	}

	return h.tradeSvc.Join(ctx, dto.TradeJoinRequest{ProfileID: profileID, Slug: in.Slug})
}

// Routes are private: create and join need the Auth Identity, so a Guest gets
// 401 (ADR-0028). The slug is the whole address; there is no list or
// owner-scoped read here.
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

// ReadRoutes is the link-public read: a Guest passes the guest-allowed guard
// and the service answers the same not-found as an unknown slug (user story
// 43), so no 401 ever confirms a Trade link exists.
func (h *TradeHandler) ReadRoutes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[tradeLookupInput, dto.TradeView]{
			OperationID: "get-trade",
			Method:      http.MethodGet,
			Path:        "/trades/{slug}",
			Summary:     "Read a Trade by its slug: state, both display names and the Trade Items on each side, without claiming the seat. An unknown, cancelled, not-yours or guest-opened slug is the same 404",
			Tags:        []string{"trades"},
			SuccessCode: http.StatusOK,
			HandlerFunc: h.get,
		}),
	}
}
