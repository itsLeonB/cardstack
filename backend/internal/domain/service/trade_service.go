package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/apperr"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// tradeSeatTakenMsg is the message of the seat-taken failure; the client
// branches on apperr.CodeTradeSeatTaken, never on this text.
const tradeSeatTakenMsg = "the trade's second seat is already taken"

// tradeNotFoundMsg is the only not-found a Trade answers with: an unknown
// slug, a cancelled Trade and a signed-in non-participant look the same, so
// the link reveals nothing (docs/adr/0028).
const tradeNotFoundMsg = "trade not found"

// TradeService is the room a Trade link opens (GLOSSARY.md: Trade, ADR-0028).
// Every method takes the caller's profile, so a Guest never reaches it — the
// routes' private guard answers 401 first — and a signed-in caller who is not
// a participant gets the same not-found as an unknown slug.
type TradeService interface {
	// Create starts a Trade with the caller on the owner seat, state open,
	// and returns its Slug for the link.
	Create(ctx context.Context, req dto.TradeCreateRequest) (dto.TradeSlug, error)
	// Get reads a Trade by slug without claiming anything: its state, both
	// participants' display names, and the Trade Items on each side. While the
	// Trade is open any signed-in holder of the link may read it (that is how
	// the second user reads the offer before joining); once someone holds the
	// second seat only participants may, and an unknown, cancelled or
	// not-yours slug is the same not-found.
	Get(ctx context.Context, req dto.TradeLookup, profileID uuid.UUID) (dto.TradeView, error)
	// Join fills the second seat once, moving the Trade to negotiating. A
	// second attempt fails with CodeTradeSeatTaken, and two joins at once
	// leave exactly one winner.
	Join(ctx context.Context, req dto.TradeJoinRequest) (dto.TradeView, error)
}

type tradeService struct {
	transactor crud.Transactor
	trades     repository.TradeRepository
	profiles   crud.Repository[entity.UserProfile]
	images     mapper.ImageHost
}

func NewTradeService(
	transactor crud.Transactor,
	trades repository.TradeRepository,
	profiles crud.Repository[entity.UserProfile],
	images mapper.ImageHost,
) TradeService {
	return &tradeService{transactor: transactor, trades: trades, profiles: profiles, images: images}
}

func (s *tradeService) Create(ctx context.Context, req dto.TradeCreateRequest) (dto.TradeSlug, error) {
	// The slug embeds the row id, so the id is made here rather than left to
	// the database default (same trick as the profile Slug, ADR-0025).
	trade := entity.Trade{
		BaseEntity:     crud.BaseEntity{ID: uuid.New()},
		Slug:           "",
		OwnerProfileID: req.ProfileID,
		Status:         entity.TradeStatusOpen,
	}
	trade.Slug = mapper.SlugOf("trade", "trade", trade.ID)

	created, err := s.trades.Insert(ctx, trade)
	if err != nil {
		return dto.TradeSlug{}, err
	}

	return dto.TradeSlug{Slug: created.Slug}, nil
}

func (s *tradeService) view(ctx context.Context, t entity.Trade, profileID uuid.UUID) (dto.TradeView, error) {
	owner, err := s.profiles.FindFirst(ctx, crud.Specification[entity.UserProfile]{
		Model: entity.UserProfile{BaseEntity: crud.BaseEntity{ID: t.OwnerProfileID}},
	})
	if err != nil {
		return dto.TradeView{}, err
	}
	if owner.IsZero() {
		return dto.TradeView{}, ungerr.NotFoundError(tradeNotFoundMsg)
	}

	out := dto.TradeView{
		Slug:          t.Slug,
		Status:        string(t.Status),
		OwnerName:     owner.Name,
		IsOwner:       t.OwnerProfileID == profileID,
		OwnerItems:    []dto.TradeItemView{},
		OtherItems:    []dto.TradeItemView{},
		IsParticipant: true,
	}

	if t.ParticipantProfileID != nil {
		other, err := s.profiles.FindFirst(ctx, crud.Specification[entity.UserProfile]{
			Model: entity.UserProfile{BaseEntity: crud.BaseEntity{ID: *t.ParticipantProfileID}},
		})
		if err != nil {
			return dto.TradeView{}, err
		}
		if other.IsZero() {
			return dto.TradeView{}, ungerr.NotFoundError(tradeNotFoundMsg)
		}
		out.OtherName = &other.Name
	}

	// Side 0 is the owner's offer, side 1 the participant's.
	for _, item := range t.TradeItems {
		v := dto.TradeItemView{
			Card:       mapper.ToBinderCardSummary(s.images, item.Card),
			Quantity:   item.Quantity,
			Collection: item.SourceCollection.Title,
		}
		if item.Side == 0 {
			out.OwnerItems = append(out.OwnerItems, v)
		} else {
			out.OtherItems = append(out.OtherItems, v)
		}
	}

	return out, nil
}

func (s *tradeService) Get(ctx context.Context, req dto.TradeLookup, profileID uuid.UUID) (dto.TradeView, error) {
	if profileID == uuid.Nil {
		return dto.TradeView{}, ungerr.UnauthorizedError("authentication required")
	}

	t, err := s.trades.FindBySlug(ctx, req.Slug)
	if err != nil {
		return dto.TradeView{}, err
	}
	if t.IsZero() || t.Status == entity.TradeStatusCancelled {
		return dto.TradeView{}, ungerr.NotFoundError(tradeNotFoundMsg)
	}
	// Open reads are link-public to signed-in users; once the second seat is
	// claimed the room closes to anyone who is not in it.
	if t.Status != entity.TradeStatusOpen && !s.isParticipant(t, profileID) {
		return dto.TradeView{}, ungerr.NotFoundError(tradeNotFoundMsg)
	}

	return s.view(ctx, t, profileID)
}

func (s *tradeService) isParticipant(t entity.Trade, profileID uuid.UUID) bool {
	return t.OwnerProfileID == profileID ||
		(t.ParticipantProfileID != nil && *t.ParticipantProfileID == profileID)
}

func (s *tradeService) Join(ctx context.Context, req dto.TradeJoinRequest) (dto.TradeView, error) {
	if req.ProfileID == uuid.Nil {
		return dto.TradeView{}, ungerr.UnauthorizedError("authentication required")
	}

	// Read once for the not-found answer; the claim itself is the
	// repository's one conditional UPDATE, which decides the winner.
	t, err := s.trades.FindBySlug(ctx, req.Slug)
	if err != nil {
		return dto.TradeView{}, err
	}
	if t.IsZero() || t.Status == entity.TradeStatusCancelled {
		return dto.TradeView{}, ungerr.NotFoundError(tradeNotFoundMsg)
	}
	if t.OwnerProfileID == req.ProfileID {
		// The owner already holds a seat; the second seat cannot also be
		// theirs, so this is only reachable if they somehow are the
		// participant already.
		if t.ParticipantProfileID != nil && *t.ParticipantProfileID == req.ProfileID {
			return dto.TradeView{}, apperr.WithCode(ungerr.ConflictError(tradeSeatTakenMsg), apperr.CodeTradeSeatTaken)
		}
		return dto.TradeView{}, ungerr.ConflictError("the trade's owner cannot take its second seat")
	}

	var joined bool
	err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		ok, err := s.trades.Join(ctx, t.ID, req.ProfileID)
		if err != nil {
			return err
		}
		joined = ok
		return nil
	})
	if err != nil {
		return dto.TradeView{}, err
	}
	if !joined {
		return dto.TradeView{}, apperr.WithCode(ungerr.ConflictError(tradeSeatTakenMsg), apperr.CodeTradeSeatTaken)
	}

	fresh, err := s.trades.FindBySlug(ctx, req.Slug)
	if err != nil {
		return dto.TradeView{}, err
	}

	return s.view(ctx, fresh, req.ProfileID)
}
