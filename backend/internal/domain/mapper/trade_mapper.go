package mapper

import (
	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
)

// ToTradeItemView maps one offered Trade Item to its view.
func ToTradeItemView(host ImageHost, item entity.TradeItem) dto.TradeItemView {
	return dto.TradeItemView{
		Card:       ToBinderCardSummary(host, item.Card),
		Quantity:   item.Quantity,
		Collection: item.SourceCollection.Title,
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
