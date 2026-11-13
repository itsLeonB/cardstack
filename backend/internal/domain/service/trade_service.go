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

// tradeNotAcceptableMsg is the message of the accept refusal while the Trade
// is not negotiating; the client branches on
// apperr.CodeTradeNotNegotiating, never on this text.
const tradeNotAcceptableMsg = "a Trade accepts only while negotiating"

// tradeItemsLockedMsg is the message of the item-edit refusal once the Trade
// is no longer negotiating; the client branches on
// apperr.CodeTradeNotNegotiating, never on this text.
const tradeItemsLockedMsg = "trade items can only change while negotiating"

// tradeStaleRevisionMsg is the message of the accept refusal when the offer
// changed since the caller read it — item edits bump offer_revision while
// an acceptance moves only updated_at; the client branches on
// apperr.CodeTradeStaleRevision, never on this text.
const tradeStaleRevisionMsg = "the trade changed since you read it; read it again and accept the deal as it now stands"

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
	Accept(ctx context.Context, req dto.TradeAcceptRequest) (dto.TradeView, error)
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
		side, ok := tradeSide(trade, req.ProfileID)
		if trade.IsZero() || !ok {
			return ungerr.NotFoundError(tradeNotFoundMsg)
		}
		if trade.Status != entity.TradeStatusNegotiating {
			return apperr.WithCode(ungerr.ConflictError(tradeItemsLockedMsg), apperr.CodeTradeNotNegotiating)
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
		side, ok := tradeSide(trade, req.ProfileID)
		if trade.IsZero() || !ok {
			return ungerr.NotFoundError(tradeNotFoundMsg)
		}
		if trade.Status != entity.TradeStatusNegotiating {
			return apperr.WithCode(ungerr.ConflictError(tradeItemsLockedMsg), apperr.CodeTradeNotNegotiating)
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

// tradeSide names the seat the caller holds, or false when they hold none.
// It answers the owner seat even while the Trade is open, so the owner can
// be told their accept is refused rather than that the Trade does not exist.
func tradeSide(trade entity.Trade, profileID uuid.UUID) (int, bool) {
	if trade.OwnerProfileID == profileID {
		return entity.SideOwner, true
	}
	if trade.ParticipantProfileID != nil && *trade.ParticipantProfileID == profileID {
		return entity.SideParticipant, true
	}
	return 0, false
}

// Accept records the caller's side as accepting the deal as it stood when
// they read it. The Trade row is locked FOR UPDATE for the whole check and
// write, so a concurrent item edit either lands first — and the revision
// compare below refuses this accept as stale — or waits and then sees the
// acceptance. The second acceptance moves the Trade to locked in the same
// transaction (spec: both sides freeze so the set cannot drift). Accepting
// again from a side that already accepted changes nothing and does not
// error. Only a negotiating Trade accepts: open has no second seat to agree
// with, and executed or cancelled Trades are done — both refuse with
// apperr.CodeTradeNotNegotiating.
func (s *tradeService) Accept(ctx context.Context, req dto.TradeAcceptRequest) (dto.TradeView, error) {
	var trade entity.Trade
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		var err error
		trade, err = s.trades.FindBySlugForUpdate(ctx, req.Slug)
		if err != nil {
			return err
		}
		// A cancelled Trade answers the same not-found as an unknown slug,
		// the rule Get and Join already follow.
		if trade.IsZero() || trade.Status == entity.TradeStatusCancelled {
			return ungerr.NotFoundError(tradeNotFoundMsg)
		}
		side, ok := tradeSide(trade, req.ProfileID)
		if !ok {
			return ungerr.NotFoundError(tradeNotFoundMsg)
		}
		// A side that already accepted re-accepting changes nothing, even
		// once the Trade has locked. An executed Trade keeps its
		// timestamps, so the status still has to be live.
		if trade.AcceptedAt(side) != nil &&
			(trade.Status == entity.TradeStatusNegotiating || trade.Status == entity.TradeStatusLocked) {
			return nil
		}
		if trade.Status != entity.TradeStatusNegotiating {
			return apperr.WithCode(ungerr.ConflictError(tradeNotAcceptableMsg), apperr.CodeTradeNotNegotiating)
		}
		// The caller accepts the offer revision they last read; an edit
		// since then bumped offer_revision and this accept would lock a
		// deal they never saw.
		if !req.Revision.Equal(trade.OfferRevision) {
			return apperr.WithCode(ungerr.ConflictError(tradeStaleRevisionMsg), apperr.CodeTradeStaleRevision)
		}
		return s.trades.Accept(ctx, trade, side)
	})
	if err != nil {
		return dto.TradeView{}, err
	}
	return s.Get(ctx, dto.TradeLookup{Slug: req.Slug, ProfileID: req.ProfileID})
}
