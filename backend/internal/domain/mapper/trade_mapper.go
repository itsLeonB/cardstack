package mapper

import (
	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
)

// ToTradeItemView maps one offered Trade Item to its view. Collection is nil
// when the item's source Collection was deleted: the Trade Item is part of
// the agreed deal and outlives it.
func ToTradeItemView(host ImageHost, item entity.TradeItem) dto.TradeItemView {
	var collection *string
	if item.SourceCollectionID != nil {
		title := item.SourceCollection.Title
		collection = &title
	}

	return dto.TradeItemView{
		ID:         item.ID,
		Card:       ToBinderCardSummary(host, item.Card),
		Quantity:   item.Quantity,
		Collection: collection,
	}
}

// tradeSide names the seat the caller holds, or -1 when they hold none.
// It answers the owner seat even while the Trade is open, so the owner can
// be told their accept is refused rather than that the Trade does not exist.
func tradeSide(trade entity.Trade, profileID uuid.UUID) int {
	if trade.OwnerProfileID == profileID {
		return entity.SideOwner
	}
	if trade.ParticipantProfileID != nil && *trade.ParticipantProfileID == profileID {
		return entity.SideParticipant
	}
	return -1
}

// otherAcceptedSeat is the seat across the table from the caller's: the
// participant's for the owner, the owner's for the participant. A caller who
// holds no seat (-1) has no "other", so it answers SideOwner — only used
// where the second seat exists, and OtherAccepted then reads the owner seat.
func otherAcceptedSeat(side int) int {
	if side == entity.SideOwner {
		return entity.SideParticipant
	}
	return entity.SideOwner
}

// ToTradeView assembles the Trade's view: slug, state, the two display names
// (never emails) and the offered items split by seat. The service resolves
// the names and passes them in; otherName is nil while the second seat is
// empty. accepted/waitingOnMe describe the caller's own side, otherAccepted/
// waitingOnOther the other seat's, all derived here from the entity.
func ToTradeView(host ImageHost, t entity.Trade, ownerName string, otherName *string, profileID uuid.UUID) dto.TradeView {
	out := dto.TradeView{
		Slug:       t.Slug,
		Status:     t.Status,
		OwnerName:  ownerName,
		OtherName:  otherName,
		OwnerItems: []dto.TradeItemView{},
		OtherItems: []dto.TradeItemView{},
		IsOwner:    t.OwnerProfileID == profileID,
		IsParticipant: t.OwnerProfileID == profileID ||
			(t.ParticipantProfileID != nil && *t.ParticipantProfileID == profileID),
	}

	side := tradeSide(t, profileID)
	if side >= 0 {
		out.Accepted = t.AcceptedAt(side) != nil
		out.WaitingOnMe = !out.Accepted && t.Status == entity.TradeStatusNegotiating
	}
	if t.ParticipantProfileID != nil {
		other := otherAcceptedSeat(side)
		out.OtherAccepted = t.AcceptedAt(other) != nil
		// waitingOnOther is the mirror of waitingOnMe: my side accepted and
		// the Trade still waits on the other seat.
		out.WaitingOnOther = out.Accepted && !out.OtherAccepted && t.Status == entity.TradeStatusNegotiating
	}

	// Side is the seat whose Collection the item leaves: SideOwner's offer,
	// SideParticipant's.
	for _, item := range t.TradeItems {
		v := ToTradeItemView(host, item)
		if item.Side == entity.SideOwner {
			out.OwnerItems = append(out.OwnerItems, v)
		} else {
			out.OtherItems = append(out.OtherItems, v)
		}
	}

	return out
}
