package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/apperr"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ezutil/v2"
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
	List(ctx context.Context, req dto.TradeListRequest) ([]dto.TradeSummary, error)
	Join(ctx context.Context, req dto.TradeJoinRequest) (dto.TradeView, error)
	AddItem(ctx context.Context, req dto.TradeItemRequest) (dto.TradeView, error)
	RemoveItem(ctx context.Context, req dto.TradeItemRemovalRequest) (dto.TradeView, error)
	Accept(ctx context.Context, req dto.TradeAcceptRequest) (dto.TradeView, error)
	Confirm(ctx context.Context, req dto.TradeConfirmRequest) (dto.TradeView, error)
	Cancel(ctx context.Context, req dto.TradeCancelRequest) error
}

type inventoryTransfer interface {
	TransferQuantity(context.Context, dto.InventoryTransferRequest) error
}

type tradeService struct {
	transactor  crud.Transactor
	trades      repository.TradeRepository
	collections repository.CollectionRepository
	inventory   repository.InventoryRepository
	transfers   inventoryTransfer
	cards       crud.Repository[entity.Card]
	profiles    crud.Repository[entity.UserProfile]
	images      mapper.ImageHost
}

func NewTradeService(
	transactor crud.Transactor,
	trades repository.TradeRepository,
	collections repository.CollectionRepository,
	inventory repository.InventoryRepository,
	transfers inventoryTransfer,
	cards crud.Repository[entity.Card],
	profiles crud.Repository[entity.UserProfile],
	images mapper.ImageHost,
) TradeService {
	return &tradeService{transactor: transactor, trades: trades, collections: collections, inventory: inventory, transfers: transfers, cards: cards, profiles: profiles, images: images}
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
		other, err := s.profiles.FindFirst(ctx, crud.Specification[entity.UserProfile]{Model: entity.UserProfile{BaseEntity: crud.BaseEntity{ID: *t.ParticipantProfileID}}})
		if err != nil {
			return dto.TradeView{}, err
		}
		if other.IsZero() {
			return dto.TradeView{}, ungerr.NotFoundError(tradeNotFoundMsg)
		}
		otherName = &other.Name
	}
	view := mapper.ToTradeView(s.images, t, owner.Name, otherName, profileID)
	if err := s.markShort(ctx, t, &view); err != nil {
		return dto.TradeView{}, err
	}
	if view.IsParticipant && t.Status == entity.TradeStatusExecuted {
		inbox, err := s.collections.FindFirst(ctx, crud.Specification[entity.Collection]{Model: entity.Collection{ProfileID: profileID, TradeID: &t.ID}})
		if err != nil {
			return dto.TradeView{}, err
		}
		if !inbox.IsZero() {
			view.InboxID = &inbox.ID
		}
	}
	return view, nil
}

// markShort re-checks every Trade Item against what its offering
// participant currently holds and marks the ones that no longer match: a
// locked Trade can go stale while the two sides arrange to meet, because
// nothing reserves the offered quantities. Available is the participant's
// total holding of the Card across all their Collections — the same total
// execution checks — and Short is Available below the offered Quantity.
// It reads only: no state and no quantity changes. An executed Trade's
// items already moved, so its holdings say nothing about the deal.
func (s *tradeService) markShort(ctx context.Context, t entity.Trade, view *dto.TradeView) error {
	if t.Status == entity.TradeStatusExecuted {
		return nil
	}
	type holdingKey struct{ profileID, cardID uuid.UUID }
	totals := make(map[holdingKey]int)
	total := func(profileID uuid.UUID, cardID uuid.UUID) (int, error) {
		key := holdingKey{profileID, cardID}
		if total, ok := totals[key]; ok {
			return total, nil
		}
		holdings, err := s.inventory.ListHoldings(ctx, profileID, cardID)
		if err != nil {
			return 0, err
		}
		sum := 0
		for _, h := range holdings {
			sum += h.Quantity
		}
		totals[key] = sum
		return sum, nil
	}
	mark := func(items []dto.TradeItemView, profileID uuid.UUID) error {
		for i := range items {
			available, err := total(profileID, items[i].Card.ID)
			if err != nil {
				return err
			}
			items[i].Available = available
			items[i].Short = available < items[i].Quantity
		}
		return nil
	}
	if err := mark(view.OwnerItems, t.OwnerProfileID); err != nil {
		return err
	}
	if t.ParticipantProfileID != nil {
		return mark(view.OtherItems, *t.ParticipantProfileID)
	}
	return nil
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

// List answers the account area's Trades page: every Trade the caller holds a
// seat in, newest first, executed and cancelled included as history. Only the
// other seat's display name is resolved — never an email (ADR-0028).
func (s *tradeService) List(ctx context.Context, req dto.TradeListRequest) ([]dto.TradeSummary, error) {
	trades, err := s.trades.FindForProfile(ctx, req.ProfileID)
	if err != nil {
		return nil, err
	}

	// Only the other seat's display name is shown, so only those ids are
	// worth a profile lookup; the caller's own name is never rendered.
	otherIDs := make(map[uuid.UUID]struct{})
	for _, t := range trades {
		if id := otherSeatProfileID(t, req.ProfileID); id != nil {
			otherIDs[*id] = struct{}{}
		}
	}
	names := make(map[uuid.UUID]string, len(otherIDs))
	if len(otherIDs) > 0 {
		ids := make([]uuid.UUID, 0, len(otherIDs))
		for id := range otherIDs {
			ids = append(ids, id)
		}
		db, err := s.profiles.GetGormInstance(ctx)
		if err != nil {
			return nil, err
		}
		var profiles []entity.UserProfile
		if err := db.Where("id IN ?", ids).Find(&profiles).Error; err != nil {
			return nil, ungerr.Wrap(err, "listing trade counterpart profiles")
		}
		for _, profile := range profiles {
			names[profile.ID] = profile.Name
		}
	}

	return ezutil.MapSlice(trades, func(t entity.Trade) dto.TradeSummary {
		var other *string
		if otherID := otherSeatProfileID(t, req.ProfileID); otherID != nil {
			if name, ok := names[*otherID]; ok {
				other = &name
			}
		}
		var slug *string
		if t.Status != entity.TradeStatusCancelled {
			// A cancelled Trade's slug is retired: the row is history, so the
			// list hands out no link that would 404.
			s := t.Slug
			slug = &s
		}
		return dto.TradeSummary{
			ID:        t.ID,
			Slug:      slug,
			Status:    t.Status,
			OtherName: other,
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		}
	}), nil
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
		for _, existing := range trade.TradeItems {
			if existing.Side == side && existing.CardID == req.CardID && existing.SourceCollectionID != nil && *existing.SourceCollectionID == req.SourceCollectionID {
				return ungerr.ConflictError("trade item already exists")
			}
		}
		source, err := s.collections.GetOwnedCollection(ctx, req.ProfileID, req.SourceCollectionID, false)
		if err != nil {
			return err
		}
		if source.TradeID != nil {
			return ungerr.ConflictError("trade inbox cannot be offered")
		}
		card, err := s.cards.FindFirst(ctx, crud.Specification[entity.Card]{Model: entity.Card{BaseEntity: crud.BaseEntity{ID: req.CardID}}})
		if err != nil {
			return err
		}
		if card.IsZero() {
			return ungerr.NotFoundError("card not found")
		}
		collectionID := req.SourceCollectionID
		return s.trades.AddItem(ctx, entity.TradeItem{TradeID: trade.ID, Side: side, CardID: req.CardID, SourceCollectionID: &collectionID, Quantity: req.Quantity})
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
func (s *tradeService) Confirm(ctx context.Context, req dto.TradeConfirmRequest) (dto.TradeView, error) {
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		trade, err := s.trades.FindBySlugForUpdate(ctx, req.Slug)
		if err != nil {
			return err
		}
		if trade.IsZero() || trade.Status == entity.TradeStatusCancelled {
			return ungerr.NotFoundError(tradeNotFoundMsg)
		}
		side, ok := tradeSide(trade, req.ProfileID)
		if !ok {
			return ungerr.NotFoundError(tradeNotFoundMsg)
		}
		if trade.Status == entity.TradeStatusExecuted {
			return apperr.WithCode(ungerr.ConflictError("trade is already executed"), apperr.CodeTradeExecuted)
		}
		if trade.Status != entity.TradeStatusLocked {
			return apperr.WithCode(ungerr.ConflictError("trade must be locked before confirmation"), apperr.CodeTradeNotNegotiating)
		}
		if side == entity.SideOwner {
			if trade.OwnerConfirmedAt != nil {
				return nil
			}
			confirmed := time.Now()
			trade.OwnerConfirmedAt = &confirmed
		} else {
			if trade.ParticipantConfirmedAt != nil {
				return nil
			}
			confirmed := time.Now()
			trade.ParticipantConfirmedAt = &confirmed
		}
		if trade.OwnerConfirmedAt == nil || trade.ParticipantConfirmedAt == nil {
			_, err := s.trades.Update(ctx, trade)
			return err
		}
		if len(trade.TradeItems) == 0 {
			return apperr.WithCode(ungerr.ConflictError("an empty trade cannot be executed"), apperr.CodeTradeEmpty)
		}
		return s.execute(ctx, &trade)
	})
	if err != nil {
		return dto.TradeView{}, err
	}
	return s.Get(ctx, dto.TradeLookup(req))
}

type tradeTransfer struct {
	profileID            uuid.UUID
	destinationProfileID uuid.UUID
	cardID               uuid.UUID
	sourceID             uuid.UUID
	destinationID        uuid.UUID
	quantity             int
}

func (s *tradeService) execute(ctx context.Context, trade *entity.Trade) error {
	type holdingKey struct{ profileID, cardID, collectionID uuid.UUID }
	type cardKey struct{ profileID, cardID uuid.UUID }
	holdings := make(map[cardKey][]repository.CardHolding)
	available := make(map[holdingKey]int)
	required := make(map[cardKey]int)
	for _, item := range trade.TradeItems {
		profileID := trade.OwnerProfileID
		if item.Side == entity.SideParticipant && trade.ParticipantProfileID != nil {
			profileID = *trade.ParticipantProfileID
		}
		key := cardKey{profileID, item.CardID}
		if _, ok := holdings[key]; !ok {
			rows, err := s.inventory.ListHoldings(ctx, profileID, item.CardID)
			if err != nil {
				return err
			}
			holdings[key] = rows
			for _, row := range rows {
				available[holdingKey{profileID, item.CardID, row.CollectionID}] = row.Quantity
				required[key] += 0
			}
		}
		required[key] += item.Quantity
		owned := 0
		for _, row := range holdings[key] {
			owned += row.Quantity
		}
		if owned < required[key] {
			return apperr.WithCode(ungerr.ConflictError(fmt.Sprintf("not enough %s to execute the trade", item.Card.Name)), apperr.CodeTradeItemShort)
		}
	}

	ownerReceives := false
	participantReceives := false
	for _, item := range trade.TradeItems {
		if item.Side == entity.SideOwner {
			participantReceives = true
		} else {
			ownerReceives = true
		}
	}
	inboxes := make(map[uuid.UUID]uuid.UUID, 2)
	if ownerReceives {
		name, err := s.profileName(ctx, *trade.ParticipantProfileID)
		if err != nil {
			return err
		}
		inbox, err := s.collections.Insert(ctx, entity.Collection{ProfileID: trade.OwnerProfileID, Title: name, TradeID: &trade.ID})
		if err != nil {
			return err
		}
		inboxes[trade.OwnerProfileID] = inbox.ID
	}
	if participantReceives {
		name, err := s.profileName(ctx, trade.OwnerProfileID)
		if err != nil {
			return err
		}
		inbox, err := s.collections.Insert(ctx, entity.Collection{ProfileID: *trade.ParticipantProfileID, Title: name, TradeID: &trade.ID})
		if err != nil {
			return err
		}
		inboxes[*trade.ParticipantProfileID] = inbox.ID
	}

	plans := make([]tradeTransfer, 0, len(trade.TradeItems))
	for _, item := range trade.TradeItems {
		profileID := trade.OwnerProfileID
		receiverID := *trade.ParticipantProfileID
		if item.Side == entity.SideParticipant {
			profileID, receiverID = receiverID, trade.OwnerProfileID
		}
		key := cardKey{profileID, item.CardID}
		left := item.Quantity
		if item.SourceCollectionID != nil {
			sourceKey := holdingKey{profileID, item.CardID, *item.SourceCollectionID}
			quantity := min(left, available[sourceKey])
			if quantity > 0 {
				plans = append(plans, tradeTransfer{profileID: profileID, destinationProfileID: receiverID, cardID: item.CardID, sourceID: *item.SourceCollectionID, destinationID: inboxes[receiverID], quantity: quantity})
				available[sourceKey] -= quantity
				left -= quantity
			}
		}
		for _, row := range holdings[key] {
			if left == 0 {
				break
			}
			sourceKey := holdingKey{profileID, item.CardID, row.CollectionID}
			quantity := min(left, available[sourceKey])
			if quantity == 0 {
				continue
			}
			plans = append(plans, tradeTransfer{profileID: profileID, destinationProfileID: receiverID, cardID: item.CardID, sourceID: row.CollectionID, destinationID: inboxes[receiverID], quantity: quantity})
			available[sourceKey] -= quantity
			left -= quantity
		}
		if left != 0 {
			return apperr.WithCode(ungerr.ConflictError(fmt.Sprintf("not enough %s to execute the trade", item.Card.Name)), apperr.CodeTradeItemShort)
		}
	}
	for _, plan := range plans {
		if err := s.transfers.TransferQuantity(ctx, dto.InventoryTransferRequest{ProfileID: plan.profileID, DestinationProfileID: plan.destinationProfileID, SourceID: plan.sourceID, DestinationID: plan.destinationID, CardID: plan.cardID, Quantity: plan.quantity}); err != nil {
			return err
		}
	}
	trade.Status = entity.TradeStatusExecuted
	_, err := s.trades.Update(ctx, *trade)
	return err
}

func (s *tradeService) profileName(ctx context.Context, profileID uuid.UUID) (string, error) {
	profile, err := s.profiles.FindFirst(ctx, crud.Specification[entity.UserProfile]{Model: entity.UserProfile{BaseEntity: crud.BaseEntity{ID: profileID}}})
	if err != nil {
		return "", err
	}
	if profile.IsZero() {
		return "", ungerr.NotFoundError(tradeNotFoundMsg)
	}
	return profile.Name, nil
}

func tradeSide(trade entity.Trade, profileID uuid.UUID) (int, bool) {
	if trade.OwnerProfileID == profileID {
		return entity.SideOwner, true
	}
	if trade.ParticipantProfileID != nil && *trade.ParticipantProfileID == profileID {
		return entity.SideParticipant, true
	}
	return 0, false
}

// otherSeatProfileID names the profile across the table from the caller's:
// the participant for the owner, the owner for the participant, nil while the
// second seat is empty or the caller holds none.
func otherSeatProfileID(trade entity.Trade, profileID uuid.UUID) *uuid.UUID {
	if trade.OwnerProfileID == profileID {
		return trade.ParticipantProfileID
	}
	if trade.ParticipantProfileID != nil && *trade.ParticipantProfileID == profileID {
		id := trade.OwnerProfileID
		return &id
	}
	return nil
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

// Cancel ends a live Trade from open, negotiating or locked, by either
// participant. An executed Trade is done — it refuses with
// apperr.CodeTradeExecuted. The repository's Cancel also retires the slug, so
// the link answers the same not-found as an unknown one from then on, and a
// new Trade draws a fresh slug so the old link never comes back.
func (s *tradeService) Cancel(ctx context.Context, req dto.TradeCancelRequest) error {
	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		trade, err := s.trades.FindBySlugForUpdate(ctx, req.Slug)
		if err != nil {
			return err
		}
		if trade.IsZero() || trade.Status == entity.TradeStatusCancelled {
			return ungerr.NotFoundError(tradeNotFoundMsg)
		}
		if !s.isParticipant(trade, req.ProfileID) {
			return ungerr.NotFoundError(tradeNotFoundMsg)
		}
		if trade.Status == entity.TradeStatusExecuted {
			return apperr.WithCode(ungerr.ConflictError("a Trade that executed cannot be cancelled"), apperr.CodeTradeExecuted)
		}
		return s.trades.Cancel(ctx, trade)
	})
}
