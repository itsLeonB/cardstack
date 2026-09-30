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
)

// CollectionService scopes every method to the request's ProfileID.
type CollectionService interface {
	Create(ctx context.Context, req dto.CollectionRequest) (dto.CollectionSummary, error)
	List(ctx context.Context, req dto.CollectionListRequest) ([]dto.CollectionSummary, error)
	Get(ctx context.Context, req dto.CollectionLookup) (dto.CollectionSummary, error)
	Update(ctx context.Context, req dto.CollectionRequest) (dto.CollectionSummary, error)
	Delete(ctx context.Context, req dto.CollectionLookup) error
}

type collectionService struct {
	repo repository.CollectionRepository
}

func NewCollectionService(repo repository.CollectionRepository) CollectionService {
	return &collectionService{repo: repo}
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

	return mapper.ToCollectionSummary(c), nil
}

// List returns most recently created first (crud.Repository's DefaultOrder).
func (s *collectionService) List(ctx context.Context, req dto.CollectionListRequest) ([]dto.CollectionSummary, error) {
	// A nil ProfileID would drop the owner condition and list everyone's.
	if req.ProfileID == uuid.Nil {
		return []dto.CollectionSummary{}, nil
	}

	collections, err := s.repo.FindAll(ctx, crud.Specification[entity.Collection]{
		Model: entity.Collection{ProfileID: req.ProfileID},
	})
	if err != nil {
		return nil, err
	}

	return ezutil.MapSlice(collections, mapper.ToCollectionSummary), nil
}

func (s *collectionService) Get(ctx context.Context, req dto.CollectionLookup) (dto.CollectionSummary, error) {
	c, err := s.repo.GetOwnedCollection(ctx, req.ProfileID, req.ID, false)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return mapper.ToCollectionSummary(c), nil
}

func (s *collectionService) Update(ctx context.Context, req dto.CollectionRequest) (dto.CollectionSummary, error) {
	c, err := s.repo.GetOwnedCollection(ctx, req.ProfileID, req.ID, false)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	c.Title = req.Title
	c.Description = req.Description
	c.MaxCardCount = req.MaxCardCount

	updated, err := s.repo.Update(ctx, c)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return mapper.ToCollectionSummary(updated), nil
}

func (s *collectionService) Delete(ctx context.Context, req dto.CollectionLookup) error {
	c, err := s.repo.GetOwnedCollection(ctx, req.ProfileID, req.ID, false)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, c)
}
