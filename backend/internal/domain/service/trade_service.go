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

// tradeOwnerJoinMsg is the message of the owner-join failure; the client
// branches on apperr.CodeTradeOwnerJoin, never on this text.
const tradeOwnerJoinMsg = "the Trade's owner cannot claim its own second seat"

// TradeService is the room a Trade link opens (GLOSSARY.md: Trade, ADR-0028).
// Create and Join need the Auth Identity, so their routes answer a Guest 401.
// Get is guest-allowed and answers the same not-found as an unknown slug, so
// a signed-in caller who is not a participant gets that too.
type TradeService interface {
	Create(ctx context.Context, req dto.TradeCreateRequest) (dto.TradeSlug, error)
	Get(ctx context.Context, req dto.TradeLookup) (dto.TradeView, error)
	Join(ctx context.Context, req dto.TradeJoinRequest) (dto.TradeView, error)
	AddItem(ctx context.Context, req dto.TradeItemRequest) (dto.TradeView, error)
	RemoveItem(ctx context.Context, req dto.TradeItemRemovalRequest) (dto.TradeView, error)
}

type tradeService struct {
	transactor  crud.Transactor
	trades      repository.TradeRepository
	collections repository.CollectionRepository
	cards       crud.Repository[entity.Card]
	profiles    crud.Repository[entity.UserProfile]
	images      mapper.ImageHost
}

func NewTradeService(
	transactor crud.Transactor,
	trades repository.TradeRepository,
	collections repository.CollectionRepository,
	cards crud.Repository[entity.Card],
	profiles crud.Repository[entity.UserProfile],
	images mapper.ImageHost,
) TradeService {
	return &tradeService{transactor: transactor, trades: trades, collections: collections, cards: cards, profiles: profiles, images: images}
}

func (s *tradeService) Create(ctx context.Context, req dto.TradeCreateRequest) (dto.TradeSlug, error) {
	// The slug embeds the row id, so the id is made here rather than left to
	// the database default (same trick as the profile Slug, ADR-0025). The
	// 12-hex tail is the only entropy in the link — it is the room's only
	// address — so uuid.New keeps it fully random.
	trade := entity.Trade{
		BaseEntity:     crud.BaseEntity{ID: uuid.New()},
		Slug:           "",
		OwnerProfileID: req.ProfileID,
		Status:         entity.TradeStatusOpen,
	}
	trade.Slug = mapper.SlugOfLong("trade", "trade", trade.ID)

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

	var otherName *string
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
		otherName = &other.Name
	}

	return mapper.ToTradeView(s.images, t, owner.Name, otherName, profileID), nil
}

func (s *tradeService) Get(ctx context.Context, req dto.TradeLookup) (dto.TradeView, error) {
	// A Guest reads nothing: the same not-found as an unknown slug, so a link
	// never confirms itself to someone without an Auth Identity (story 43).
	if req.ProfileID == uuid.Nil {
		return dto.TradeView{}, ungerr.NotFoundError(tradeNotFoundMsg)
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
	if t.Status != entity.TradeStatusOpen && !s.isParticipant(t, req.ProfileID) {
		return dto.TradeView{}, ungerr.NotFoundError(tradeNotFoundMsg)
	}

	return s.view(ctx, t, req.ProfileID)
}

func (s *tradeService) isParticipant(t entity.Trade, profileID uuid.UUID) bool {
	return t.OwnerProfileID == profileID ||
		(t.ParticipantProfileID != nil && *t.ParticipantProfileID == profileID)
}

func (s *tradeService) Join(ctx context.Context, req dto.TradeJoinRequest) (dto.TradeView, error) {
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
		return dto.TradeView{}, apperr.WithCode(
			ungerr.ConflictError(tradeOwnerJoinMsg), apperr.CodeTradeOwnerJoin)
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

func (s *tradeService) AddItem(ctx context.Context, req dto.TradeItemRequest) (dto.TradeView, error) {
	if req.CardID == uuid.Nil {
		return dto.TradeView{}, ungerr.NotFoundError("card not found")
	}
	if req.Quantity <= 0 {
		return dto.TradeView{}, ungerr.BadRequestError("trade item quantity must be positive")
	}

	var trade entity.Trade
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		var err error
		trade, err = s.trades.FindBySlugForUpdate(ctx, req.Slug)
		if err != nil {
			return err
		}
		side, ok := tradeItemSide(trade, req.ProfileID)
		if trade.IsZero() || !ok {
			return ungerr.NotFoundError(tradeNotFoundMsg)
		}
		if trade.Status != entity.TradeStatusNegotiating && (trade.Status != entity.TradeStatusOpen || side != entity.SideOwner) {
			return ungerr.ConflictError("trade items can only change while open or negotiating")
		}
		for _, existing := range trade.TradeItems {
			if existing.Side == side && existing.CardID == req.CardID && existing.SourceCollectionID != nil && *existing.SourceCollectionID == req.SourceCollectionID {
				return ungerr.ConflictError("trade item already exists")
			}
		}
		if _, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.SourceCollectionID, false); err != nil {
			return err
		}
		card, err := s.cards.FindFirst(ctx, crud.Specification[entity.Card]{
			Model: entity.Card{BaseEntity: crud.BaseEntity{ID: req.CardID}},
		})
		if err != nil {
			return err
		}
		if card.IsZero() {
			return ungerr.NotFoundError("card not found")
		}
		collectionID := req.SourceCollectionID
		return s.trades.AddItem(ctx, entity.TradeItem{
			TradeID: trade.ID, Side: side, CardID: req.CardID,
			SourceCollectionID: &collectionID, Quantity: req.Quantity,
		})
	})
	if err != nil {
		return dto.TradeView{}, err
	}
	return s.Get(ctx, dto.TradeLookup{Slug: req.Slug, ProfileID: req.ProfileID})
}

func (s *tradeService) RemoveItem(ctx context.Context, req dto.TradeItemRemovalRequest) (dto.TradeView, error) {
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		trade, err := s.trades.FindBySlugForUpdate(ctx, req.Slug)
		if err != nil {
			return err
		}
		side, ok := tradeItemSide(trade, req.ProfileID)
		if trade.IsZero() || !ok {
			return ungerr.NotFoundError(tradeNotFoundMsg)
		}
		if trade.Status != entity.TradeStatusNegotiating && (trade.Status != entity.TradeStatusOpen || side != entity.SideOwner) {
			return ungerr.ConflictError("trade items can only change while open or negotiating")
		}
		removed, err := s.trades.RemoveItem(ctx, trade.ID, req.ItemID, side)
		if err != nil {
			return err
		}
		if !removed {
			return ungerr.NotFoundError("trade item not found")
		}
		return nil
	})
	if err != nil {
		return dto.TradeView{}, err
	}
	return s.Get(ctx, dto.TradeLookup{Slug: req.Slug, ProfileID: req.ProfileID})
}

func tradeItemSide(trade entity.Trade, profileID uuid.UUID) (int, bool) {
	if trade.OwnerProfileID == profileID {
		return entity.SideOwner, true
	}
	if trade.ParticipantProfileID != nil && *trade.ParticipantProfileID == profileID {
		return entity.SideParticipant, true
	}
	return 0, false
}
