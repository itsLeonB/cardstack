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

// ToTradeView assembles the Trade's view: slug, state, the two display names
// (never emails) and the offered items split by seat. The service resolves
// the names and passes them in; otherName is nil while the second seat is
// empty.
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
