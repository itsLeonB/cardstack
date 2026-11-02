package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ezutil/v2"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

const (
	wishlistQuantityNotPositiveMsg = "wanted quantity must be at least 1"
	wishlistItemNotFoundMsg        = "card is not on this wishlist"
)

// WishlistService owns a profile's Wishlist: the Cards it wants, each against
// the quantity the profile owns in Master Inventory. A Wishlist Item never
// counts toward Master Inventory.
type WishlistService interface {
	// List pages the profile's Wishlist Items matching req.Filter, with the
	// wanted quantity, the owned quantity and the derived status.
	List(ctx context.Context, req dto.WishlistListRequest) ([]dto.WishlistItem, dto.PaginationMeta, error)
	// ListFacets returns filter options computed only from the Items List would show.
	ListFacets(ctx context.Context, req dto.WishlistListRequest) (dto.CatalogFacets, error)
	// SetQuantity creates the item, or replaces the wanted quantity of one
	// already on the Wishlist.
	SetQuantity(ctx context.Context, req dto.WishlistItemRequest) error
	Remove(ctx context.Context, req dto.WishlistItemLookup) error
	// ClearFulfilled removes every item the profile already has enough of.
	ClearFulfilled(ctx context.Context, req dto.WishlistLookup) (dto.WishlistClearResult, error)
}

type wishlistService struct {
	transactor crud.Transactor
	profiles   crud.Repository[entity.UserProfile]
	items      repository.WishlistRepository
	cards      CardLookup
	images     mapper.ImageHost
}

func NewWishlistService(
	transactor crud.Transactor,
	profiles crud.Repository[entity.UserProfile],
	items repository.WishlistRepository,
	cards CardLookup,
	images mapper.ImageHost,
) WishlistService {
	return &wishlistService{transactor: transactor, profiles: profiles, items: items, cards: cards, images: images}
}

func (s *wishlistService) List(ctx context.Context, req dto.WishlistListRequest) ([]dto.WishlistItem, dto.PaginationMeta, error) {
	// A nil ProfileID would match no items anyway; refuse it rather than lean
	// on that, so a forgotten scope can never list everyone's Wishlist.
	if req.ProfileID == uuid.Nil {
		return nil, dto.PaginationMeta{}, ungerr.UnauthorizedError("missing profile")
	}

	filter, page, limit := wishlistRepoFilter(req)
	results, total, err := s.items.List(ctx, filter)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	return ezutil.MapSlice(results, func(r repository.CardResult) dto.WishlistItem { return mapper.ToWishlistItem(s.images, r) }), dto.PaginationMeta{Total: int(total), Page: page, Limit: limit}, nil
}

func (s *wishlistService) ListFacets(ctx context.Context, req dto.WishlistListRequest) (dto.CatalogFacets, error) {
	if req.ProfileID == uuid.Nil {
		return dto.CatalogFacets{}, ungerr.UnauthorizedError("missing profile")
	}

	filter, _, _ := wishlistRepoFilter(req)
	facets, err := s.items.ListFacets(ctx, filter)
	if err != nil {
		return dto.CatalogFacets{}, err
	}

	return mapper.ToCatalogFacets(s.images, facets), nil
}

func (s *wishlistService) SetQuantity(ctx context.Context, req dto.WishlistItemRequest) error {
	if req.Quantity <= 0 {
		return ungerr.BadRequestError(wishlistQuantityNotPositiveMsg)
	}
	// A Card of an unpublished Expansion Set is absent from the Wishlist
	// everywhere, so it cannot be added either.
	if err := s.cards.RequireCard(ctx, req.CardID); err != nil {
		return err
	}

	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := s.lockOwner(ctx, req.ProfileID); err != nil {
			return err
		}

		item, err := s.findItem(ctx, req.ProfileID, req.CardID)
		if err != nil {
			return err
		}
		if item.IsZero() {
			_, err = s.items.Insert(ctx, entity.WishlistItem{ProfileID: req.ProfileID, CardID: req.CardID, WantedQuantity: req.Quantity})
			return err
		}

		item.WantedQuantity = req.Quantity
		_, err = s.items.Update(ctx, item)

		return err
	})
}

func (s *wishlistService) Remove(ctx context.Context, req dto.WishlistItemLookup) error {
	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := s.lockOwner(ctx, req.ProfileID); err != nil {
			return err
		}

		item, err := s.findItem(ctx, req.ProfileID, req.CardID)
		if err != nil {
			return err
		}
		if item.IsZero() {
			return ungerr.NotFoundError(wishlistItemNotFoundMsg)
		}

		return s.items.Delete(ctx, item)
	})
}

func (s *wishlistService) ClearFulfilled(ctx context.Context, req dto.WishlistLookup) (dto.WishlistClearResult, error) {
	var removed int
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := s.lockOwner(ctx, req.ProfileID); err != nil {
			return err
		}

		var err error
		removed, err = s.items.DeleteFulfilled(ctx, req.ProfileID)

		return err
	})
	if err != nil {
		return dto.WishlistClearResult{}, err
	}

	return dto.WishlistClearResult{Removed: removed}, nil
}

// lockOwner row-locks the profile, which serializes every write to its
// Wishlist: a Wishlist has no row of its own yet, so the owner is the row to
// lock. A nil ProfileID would drop the condition and lock an arbitrary
// profile, so it is refused up front.
func (s *wishlistService) lockOwner(ctx context.Context, profileID uuid.UUID) error {
	if profileID == uuid.Nil {
		return ungerr.UnauthorizedError("missing profile")
	}

	profile, err := s.profiles.FindFirst(ctx, crud.Specification[entity.UserProfile]{
		Model:     entity.UserProfile{BaseEntity: crud.BaseEntity{ID: profileID}},
		ForUpdate: true,
	})
	if err != nil {
		return err
	}
	if profile.IsZero() {
		return ungerr.UnauthorizedError("missing profile")
	}

	return nil
}

// findItem returns the profile's item for the Card, or the zero value when
// there is none, holding its row lock until the transaction ends.
func (s *wishlistService) findItem(ctx context.Context, profileID, cardID uuid.UUID) (entity.WishlistItem, error) {
	return s.items.FindFirst(ctx, crud.Specification[entity.WishlistItem]{
		Model:     entity.WishlistItem{ProfileID: profileID, CardID: cardID},
		ForUpdate: true,
	})
}

func wishlistRepoFilter(req dto.WishlistListRequest) (repository.WishlistFilter, int, int) {
	cards, page, limit := pagedRepoFilter(req.Filter)

	return repository.WishlistFilter{
		ProfileID: req.ProfileID,
		Status:    mapper.ToRepoWishlistStatus(req.Status),
		Cards:     cards,
	}, page, limit
}
