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
