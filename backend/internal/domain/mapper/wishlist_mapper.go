package mapper

import (
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
)

// ToWishlistItem derives an item's status from the two quantities, so the rule
// lives in one place: fulfilled is owned at least wanted.
func ToWishlistItem(host ImageHost, r repository.CardResult) dto.WishlistItem {
	return dto.WishlistItem{
		Card:           ToCardSummary(host, r),
		WantedQuantity: r.WantedQuantity,
		OwnedQuantity:  r.Quantity,
		Fulfilled:      r.Quantity >= r.WantedQuantity,
	}
}

// ToRepoWishlistStatus maps the API's status filter; anything but the two
// narrower values means All.
func ToRepoWishlistStatus(status string) repository.WishlistStatus {
	switch status {
	case dto.WishlistStatusNeeded:
		return repository.WishlistStatusNeeded
	case dto.WishlistStatusFulfilled:
		return repository.WishlistStatusFulfilled
	default:
		return repository.WishlistStatusAll
	}
}
