package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ezutil/v2"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// sharedNotFoundMsg is the only not-found a shared link answers with: a
// missing profile, a missing Collection, and a private or unshared one look
// the same, so the link reveals nothing (docs/adr/0025).
const sharedNotFoundMsg = "shared collection not found"

// SharedCollectionService serves a Shared Collection to whoever holds its link.
// It takes no caller, so a Guest, a signed-in user and the owner all get the
// same answer, and the owner's email is never part of it.
type SharedCollectionService interface {
	Get(ctx context.Context, req dto.SharedCollectionLookup) (dto.SharedCollectionSummary, error)
	ListEntries(ctx context.Context, req dto.SharedCollectionListRequest) ([]dto.InventoryItem, dto.PaginationMeta, error)
	ListFacets(ctx context.Context, req dto.SharedCollectionListRequest) (dto.CatalogFacets, error)
	GetPage(ctx context.Context, req dto.SharedCollectionPageRequest) (dto.BinderPage, error)
}

type sharedCollectionService struct {
	profiles    crud.Repository[entity.UserProfile]
	collections repository.CollectionRepository
	catalog     repository.CatalogRepository
	binders     BinderPageService
	images      mapper.ImageHost
}

func NewSharedCollectionService(
	profiles crud.Repository[entity.UserProfile],
	collections repository.CollectionRepository,
	catalog repository.CatalogRepository,
	binders BinderPageService,
	images mapper.ImageHost,
) SharedCollectionService {
	return &sharedCollectionService{profiles: profiles, collections: collections, catalog: catalog, binders: binders, images: images}
}

// resolve finds the owner's profile by its Slug, then the Collection by its
// Slug under that profile. A Slug is never empty, so an empty one is a miss.
func (s *sharedCollectionService) resolve(ctx context.Context, lookup dto.SharedCollectionLookup) (entity.UserProfile, entity.Collection, error) {
	if lookup.ProfileSlug == "" {
		return entity.UserProfile{}, entity.Collection{}, ungerr.NotFoundError(sharedNotFoundMsg)
	}

	profile, err := s.profiles.FindFirst(ctx, crud.Specification[entity.UserProfile]{
		Model: entity.UserProfile{Slug: lookup.ProfileSlug},
	})
	if err != nil {
		return entity.UserProfile{}, entity.Collection{}, err
	}
	if profile.IsZero() {
		return entity.UserProfile{}, entity.Collection{}, ungerr.NotFoundError(sharedNotFoundMsg)
	}

	c, err := s.collections.FindSharedCollection(ctx, profile.ID, lookup.CollectionSlug)
	if err != nil {
		return entity.UserProfile{}, entity.Collection{}, err
	}
	if c.IsZero() {
		return entity.UserProfile{}, entity.Collection{}, ungerr.NotFoundError(sharedNotFoundMsg)
	}

	return profile, c, nil
}

func (s *sharedCollectionService) Get(ctx context.Context, req dto.SharedCollectionLookup) (dto.SharedCollectionSummary, error) {
	profile, c, err := s.resolve(ctx, req)
	if err != nil {
		return dto.SharedCollectionSummary{}, err
	}

	summary := dto.SharedCollectionSummary{OwnerName: profile.Name, Title: c.Title, Description: c.Description}

	// A plain Collection has no Layout, so its header has no binder block.
	header, err := s.binders.Header(ctx, dto.BinderReadRequest{CollectionID: c.ID})
	switch {
	case err == nil:
		summary.Binder = &header
	case !isNotFound(err):
		return dto.SharedCollectionSummary{}, err
	}

	return summary, nil
}

// ListEntries pages the Shared Collection's Cards with the owner's listing's
// filters. It leaves out Cards of unpublished sets, since the catalog query
// keeps them out unless IncludeUnpublished is set, and this never sets it.
// It does not apply the catalog's Guest preview lock: a shared page is the
// owner's to show, so a Guest gets every page and filter.
func (s *sharedCollectionService) ListEntries(ctx context.Context, req dto.SharedCollectionListRequest) ([]dto.InventoryItem, dto.PaginationMeta, error) {
	_, c, err := s.resolve(ctx, req.Lookup)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	repoFilter, page, limit := pagedRepoFilter(req.Filter)
	repoFilter.CollectionID = c.ID

	results, total, err := s.catalog.SearchCards(ctx, repoFilter)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	return ezutil.MapSlice(results, func(r repository.CardResult) dto.InventoryItem { return mapper.ToInventoryItem(s.images, r) }), dto.PaginationMeta{Total: int(total), Page: page, Limit: limit}, nil
}

func (s *sharedCollectionService) ListFacets(ctx context.Context, req dto.SharedCollectionListRequest) (dto.CatalogFacets, error) {
	_, c, err := s.resolve(ctx, req.Lookup)
	if err != nil {
		return dto.CatalogFacets{}, err
	}

	f := mapper.ToRepoCardFilter(req.Filter)
	f.CollectionID = c.ID

	facets, err := s.catalog.ListCardFacets(ctx, f)
	if err != nil {
		return dto.CatalogFacets{}, err
	}

	return mapper.ToCatalogFacets(s.images, facets), nil
}

// GetPage reads one Page of a Shared binder for any viewer. Cards of unpublished
// sets read as empty, as ListEntries leaves them out. A plain Collection and a
// Page past the last Leaf answer the same not-found as a missing link.
func (s *sharedCollectionService) GetPage(ctx context.Context, req dto.SharedCollectionPageRequest) (dto.BinderPage, error) {
	_, c, err := s.resolve(ctx, req.Lookup)
	if err != nil {
		return dto.BinderPage{}, err
	}

	page, err := s.binders.Page(ctx, dto.BinderPageReadRequest{CollectionID: c.ID, Page: req.Page})
	if isNotFound(err) {
		return dto.BinderPage{}, ungerr.NotFoundError(sharedNotFoundMsg)
	}

	return page, err
}

// isNotFound reports whether err is a not-found AppError, which the binder reads
// return for a plain Collection or a Page past the last Leaf.
func isNotFound(err error) bool {
	appErr, ok := errors.AsType[ungerr.AppError](err)
	return ok && appErr.HttpStatus() == http.StatusNotFound
}
