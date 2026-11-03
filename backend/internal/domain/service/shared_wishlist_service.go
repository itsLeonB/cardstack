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

// sharedWishlistNotFoundMsg is the only not-found a shared Wishlist answers
// with: a missing profile, a missing Wishlist row, and a private or unshared
// one look the same, so the link reveals nothing (docs/adr/0025).
const sharedWishlistNotFoundMsg = "shared collection not found"

// SharedWishlistService serves a Shared Wishlist to whoever holds its link.
// It takes no caller, so a Guest, a signed-in user and the owner all get the
// same answer, and the owner's email is never part of it.
type SharedWishlistService interface {
	Get(ctx context.Context, req dto.SharedWishlistLookup) (dto.SharedWishlistSummary, error)
	ListEntries(ctx context.Context, req dto.SharedWishlistListRequest) ([]dto.WishlistItem, dto.PaginationMeta, error)
	ListFacets(ctx context.Context, req dto.SharedWishlistListRequest) (dto.CatalogFacets, error)
}

type sharedWishlistService struct {
	profiles  crud.Repository[entity.UserProfile]
	wishlists crud.Repository[entity.Wishlist]
	items     repository.WishlistRepository
	images    mapper.ImageHost
}

func NewSharedWishlistService(
	profiles crud.Repository[entity.UserProfile],
	wishlists crud.Repository[entity.Wishlist],
	items repository.WishlistRepository,
	images mapper.ImageHost,
) SharedWishlistService {
	return &sharedWishlistService{profiles: profiles, wishlists: wishlists, items: items, images: images}
}

// resolve finds the owner's profile by its Slug, then the Wishlist if shared.
// A Slug is never empty, so an empty one is a miss.
func (s *sharedWishlistService) resolve(ctx context.Context, lookup dto.SharedWishlistLookup) (entity.UserProfile, entity.Wishlist, error) {
	if lookup.ProfileSlug == "" {
		return entity.UserProfile{}, entity.Wishlist{}, ungerr.NotFoundError(sharedWishlistNotFoundMsg)
	}

	profile, err := s.profiles.FindFirst(ctx, crud.Specification[entity.UserProfile]{
		Model: entity.UserProfile{Slug: lookup.ProfileSlug},
	})
	if err != nil {
		return entity.UserProfile{}, entity.Wishlist{}, err
	}
	if profile.IsZero() {
		return entity.UserProfile{}, entity.Wishlist{}, ungerr.NotFoundError(sharedWishlistNotFoundMsg)
	}

	w, err := s.wishlists.FindFirst(ctx, crud.Specification[entity.Wishlist]{
		Model: entity.Wishlist{ProfileID: profile.ID, Shared: true},
	})
	if err != nil {
		return entity.UserProfile{}, entity.Wishlist{}, err
	}
	if w.IsZero() {
		return entity.UserProfile{}, entity.Wishlist{}, ungerr.NotFoundError(sharedWishlistNotFoundMsg)
	}

	return profile, w, nil
}

func (s *sharedWishlistService) Get(ctx context.Context, req dto.SharedWishlistLookup) (dto.SharedWishlistSummary, error) {
	profile, _, err := s.resolve(ctx, req)
	if err != nil {
		return dto.SharedWishlistSummary{}, err
	}

	return dto.SharedWishlistSummary{OwnerName: profile.Name}, nil
}

// ListEntries pages the Shared Wishlist's Items with the owner's listing's
// filters. It leaves out Cards of unpublished sets, since the catalog query
// keeps them out unless IncludeUnpublished is set, and this never sets it.
// It does not apply the catalog's Guest preview lock: a shared page is the
// owner's to show, so a Guest gets every page and filter.
// A non-owner, signed-in viewer also gets their own owned quantity per item
// (viewer overlay) and a "have"/"dont-have" filter on it. A Guest or the owner
// gets no viewer data.
func (s *sharedWishlistService) ListEntries(ctx context.Context, req dto.SharedWishlistListRequest) ([]dto.WishlistItem, dto.PaginationMeta, error) {
	profile, _, err := s.resolve(ctx, req.Lookup)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	filter, page, limit := wishlistRepoFilter(dto.WishlistListRequest{
		ProfileID: profile.ID,
		Status:    req.Status,
		Filter:    req.Filter,
	})
	// Viewer overlay: only a signed-in viewer who is not the owner gets their
	// own owned quantity per item and the viewer filter.
	if req.ViewerProfileID != uuid.Nil && req.ViewerProfileID != profile.ID {
		filter.ViewerProfileID = req.ViewerProfileID
		filter.ViewerFilter = req.ViewerFilter
	}
	results, total, err := s.items.List(ctx, filter)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	return ezutil.MapSlice(results, func(r repository.CardResult) dto.WishlistItem { return mapper.ToWishlistItem(s.images, r) }), dto.PaginationMeta{Total: int(total), Page: page, Limit: limit}, nil
}

func (s *sharedWishlistService) ListFacets(ctx context.Context, req dto.SharedWishlistListRequest) (dto.CatalogFacets, error) {
	profile, _, err := s.resolve(ctx, req.Lookup)
	if err != nil {
		return dto.CatalogFacets{}, err
	}

	filter, _, _ := wishlistRepoFilter(dto.WishlistListRequest{
		ProfileID: profile.ID,
		Status:    req.Status,
		Filter:    req.Filter,
	})
	facets, err := s.items.ListFacets(ctx, filter)
	if err != nil {
		return dto.CatalogFacets{}, err
	}

	return mapper.ToCatalogFacets(s.images, facets), nil
}
