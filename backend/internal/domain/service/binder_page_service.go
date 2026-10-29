package service

import (
	"context"

	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
)

// BinderPageService reads a binder's header and Pages from a Collection the
// caller has already resolved. It checks no owner: binderService.GetPage resolves
// the Collection through the owner's scope and sharedCollectionService through a
// link's Slugs, and both build the binder here, so the owner's view and a Shared
// view of one Page cannot drift apart.
type BinderPageService interface {
	// Header returns the binder's Layout and Unidentified count. A plain
	// Collection has no Layout, so Header returns its not-found AppError.
	Header(ctx context.Context, req dto.BinderReadRequest) (dto.BinderHeader, error)
	// Page returns one Page. A plain Collection, or a Page past the last Leaf,
	// returns its not-found AppError.
	Page(ctx context.Context, req dto.BinderPageReadRequest) (dto.BinderPage, error)
}

type binderPageService struct {
	layouts repository.BinderLayoutRepository
	leaves  repository.BinderLeafRepository
	slots   repository.BinderSlotRepository
	images  mapper.ImageHost
}

func NewBinderPageService(
	layouts repository.BinderLayoutRepository,
	leaves repository.BinderLeafRepository,
	slots repository.BinderSlotRepository,
	images mapper.ImageHost,
) BinderPageService {
	return &binderPageService{layouts: layouts, leaves: leaves, slots: slots, images: images}
}

func (s *binderPageService) Header(ctx context.Context, req dto.BinderReadRequest) (dto.BinderHeader, error) {
	layout, err := s.layouts.GetLayout(ctx, req.CollectionID)
	if err != nil {
		return dto.BinderHeader{}, err
	}

	leafCount, err := s.leaves.CountLeaves(ctx, req.CollectionID)
	if err != nil {
		return dto.BinderHeader{}, err
	}

	// An Unidentified Slot holds no Card, so hiding unpublished Cards leaves this count as it is.
	_, unidentified, err := s.slots.CountSlots(ctx, req.CollectionID, false)
	if err != nil {
		return dto.BinderHeader{}, err
	}

	return dto.BinderHeader{Layout: mapper.ToBinderLayout(layout, leafCount), UnidentifiedCount: unidentified}, nil
}

func (s *binderPageService) Page(ctx context.Context, req dto.BinderPageReadRequest) (dto.BinderPage, error) {
	layout, err := s.layouts.GetLayout(ctx, req.CollectionID)
	if err != nil {
		return dto.BinderPage{}, err
	}

	leafCount, err := s.leaves.CountLeaves(ctx, req.CollectionID)
	if err != nil {
		return dto.BinderPage{}, err
	}
	leafNumber, side := mapper.LeafOf(req.Page)
	leaf, err := s.leaves.GetLeafAt(ctx, req.CollectionID, leafNumber)
	if err != nil {
		return dto.BinderPage{}, err
	}

	slots, err := s.slots.ListSlotsOnPage(ctx, leaf.ID, side, req.IncludeUnpublished)
	if err != nil {
		return dto.BinderPage{}, err
	}

	cards, unidentified, err := s.slots.CountSlots(ctx, req.CollectionID, req.IncludeUnpublished)
	if err != nil {
		return dto.BinderPage{}, err
	}

	return mapper.ToBinderPage(s.images, req.Page, layout, leafCount, slots, cards+unidentified, unidentified), nil
}
