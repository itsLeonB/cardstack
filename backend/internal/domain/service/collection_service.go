package service

import (
	"context"

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

const binderMaxCardCountMsg = "a binder's capacity is its Slot count, so max_card_count must stay 0"

// CollectionService scopes every method to the request's ProfileID.
type CollectionService interface {
	Create(ctx context.Context, req dto.CollectionRequest) (dto.CollectionSummary, error)
	List(ctx context.Context, req dto.CollectionListRequest) ([]dto.CollectionSummary, error)
	Get(ctx context.Context, req dto.CollectionLookup) (dto.CollectionSummary, error)
	Update(ctx context.Context, req dto.CollectionRequest) (dto.CollectionSummary, error)
	Delete(ctx context.Context, req dto.CollectionLookup) error
	// Share turns sharing on, giving the Collection a Slug. Sharing an
	// already-Shared Collection keeps its Slug, so the link does not move.
	Share(ctx context.Context, req dto.CollectionLookup) (dto.CollectionSummary, error)
	// Unshare turns sharing off and clears the Slug, ending the link at once.
	Unshare(ctx context.Context, req dto.CollectionLookup) (dto.CollectionSummary, error)
}

type collectionService struct {
	transactor crud.Transactor
	repo       repository.CollectionRepository
	layouts    repository.BinderLayoutRepository
	leaves     repository.BinderLeafRepository
}

func NewCollectionService(
	transactor crud.Transactor,
	repo repository.CollectionRepository,
	layouts repository.BinderLayoutRepository,
	leaves repository.BinderLeafRepository,
) CollectionService {
	return &collectionService{transactor: transactor, repo: repo, layouts: layouts, leaves: leaves}
}

func (s *collectionService) Create(ctx context.Context, req dto.CollectionRequest) (dto.CollectionSummary, error) {
	c, err := s.repo.Insert(ctx, entity.Collection{
		ProfileID:    req.ProfileID,
		Title:        req.Title,
		Description:  req.Description,
		MaxCardCount: req.MaxCardCount,
	})
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return mapper.ToCollectionSummary(c, 0, nil), nil
}

// List returns most recently created first (crud.Repository's DefaultOrder).
func (s *collectionService) List(ctx context.Context, req dto.CollectionListRequest) ([]dto.CollectionSummary, error) {
	// A nil ProfileID would drop the owner condition and list everyone's.
	if req.ProfileID == uuid.Nil {
		return []dto.CollectionSummary{}, nil
	}

	collections, err := s.repo.FindAll(ctx, crud.Specification[entity.Collection]{
		Model:            entity.Collection{ProfileID: req.ProfileID},
		PreloadRelations: []string{"Layout"},
	})
	if err != nil {
		return nil, err
	}

	ids := ezutil.MapSlice(collections, func(c entity.Collection) uuid.UUID { return c.ID })
	counts, err := s.repo.SumQuantities(ctx, ids)
	if err != nil {
		return nil, err
	}
	leafCounts, err := s.leaves.LeafCounts(ctx, ids)
	if err != nil {
		return nil, err
	}

	return ezutil.MapSlice(collections, func(c entity.Collection) dto.CollectionSummary {
		var layout entity.CollectionLayout
		if c.Layout != nil {
			layout = *c.Layout
		}
		return mapper.ToCollectionSummary(c, counts[c.ID], binderSummary(layout, leafCounts[c.ID]))
	}), nil
}

func (s *collectionService) Get(ctx context.Context, req dto.CollectionLookup) (dto.CollectionSummary, error) {
	c, err := s.repo.GetOwnedCollection(ctx, req.ProfileID, req.ID, false)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return s.summarize(ctx, c)
}

// Update row-locks the Collection, so an update cannot race CreateBinder and
// leave a binder with a max_card_count.
func (s *collectionService) Update(ctx context.Context, req dto.CollectionRequest) (dto.CollectionSummary, error) {
	var updated entity.Collection
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, err := s.repo.GetOwnedCollection(ctx, req.ProfileID, req.ID, true)
		if err != nil {
			return err
		}

		if req.MaxCardCount != 0 {
			isBinder, err := s.layouts.IsBinder(ctx, c.ID)
			if err != nil {
				return err
			}
			if isBinder {
				return apperr.WithCode(ungerr.ConflictError(binderMaxCardCountMsg), apperr.CodeBinderMaxCardCount)
			}
		}

		c.Title = req.Title
		c.Description = req.Description
		c.MaxCardCount = req.MaxCardCount

		updated, err = s.repo.Update(ctx, c)
		return err
	})
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return s.summarize(ctx, updated)
}

func (s *collectionService) Delete(ctx context.Context, req dto.CollectionLookup) error {
	c, err := s.repo.GetOwnedCollection(ctx, req.ProfileID, req.ID, false)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, c)
}

// Share gives the Collection a Slug from its title. The tail is the last 5 hex
// characters of a new id, not the row's own: the row's id never changes, so
// reusing it would make a re-share reproduce the old link, which must never
// come back (ADR-0025, Consequences). A new uuid's tail is random too.
func (s *collectionService) Share(ctx context.Context, req dto.CollectionLookup) (dto.CollectionSummary, error) {
	var shared entity.Collection
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, err := s.repo.GetOwnedCollection(ctx, req.ProfileID, req.ID, true)
		if err != nil {
			return err
		}
		if c.Slug != nil {
			shared = c
			return nil
		}

		// ponytail: a collision on (profile, stem, 5-hex tail) fails the unique index and surfaces as a 500, with no retry. Upgrade: on that unique violation, retry in a new transaction (this one is aborted) with a fresh tail.
		slug := mapper.SlugOf(c.Title, "collection", uuid.New())
		c.Slug = &slug
		shared, err = s.repo.Update(ctx, c)
		return err
	})
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return s.summarize(ctx, shared)
}

func (s *collectionService) Unshare(ctx context.Context, req dto.CollectionLookup) (dto.CollectionSummary, error) {
	var unshared entity.Collection
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		c, err := s.repo.GetOwnedCollection(ctx, req.ProfileID, req.ID, true)
		if err != nil {
			return err
		}
		if c.Slug == nil {
			unshared = c
			return nil
		}

		c.Slug = nil
		unshared, err = s.repo.Update(ctx, c)
		return err
	})
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return s.summarize(ctx, unshared)
}

func (s *collectionService) summarize(ctx context.Context, c entity.Collection) (dto.CollectionSummary, error) {
	counts, err := s.repo.SumQuantities(ctx, []uuid.UUID{c.ID})
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	layout, err := s.layouts.FindLayout(ctx, c.ID)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	var leafCount int
	if !layout.IsZero() {
		if leafCount, err = s.leaves.CountLeaves(ctx, c.ID); err != nil {
			return dto.CollectionSummary{}, err
		}
	}

	return mapper.ToCollectionSummary(c, counts[c.ID], binderSummary(layout, leafCount)), nil
}

// binderSummary is nil for a Collection without a Layout.
func binderSummary(layout entity.CollectionLayout, leafCount int) *dto.BinderLayout {
	if layout.IsZero() {
		return nil
	}

	binder := mapper.ToBinderLayout(layout, leafCount)
	return &binder
}
