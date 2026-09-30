package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/ezutil/v2"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

// collectionNotFoundMsg is the message of the 404 AppError, also returned for
// another profile's Collection so its existence isn't leaked. Each return
// site builds its own ungerr.NotFoundError so ungerr records that line.
const collectionNotFoundMsg = "collection not found"

// CollectionService scopes every method to the calling profileID.
type CollectionService interface {
	Create(ctx context.Context, profileID uuid.UUID, req dto.CollectionRequest) (dto.CollectionSummary, error)
	List(ctx context.Context, profileID uuid.UUID) ([]dto.CollectionSummary, error)
	Get(ctx context.Context, profileID, id uuid.UUID) (dto.CollectionSummary, error)
	Update(ctx context.Context, profileID, id uuid.UUID, req dto.CollectionRequest) (dto.CollectionSummary, error)
	Delete(ctx context.Context, profileID, id uuid.UUID) error
}

type collectionService struct {
	repo crud.Repository[entity.Collection]
}

func NewCollectionService(repo crud.Repository[entity.Collection]) CollectionService {
	return &collectionService{repo: repo}
}

func (s *collectionService) Create(ctx context.Context, profileID uuid.UUID, req dto.CollectionRequest) (dto.CollectionSummary, error) {
	c, err := s.repo.Insert(ctx, entity.Collection{
		ProfileID:    profileID,
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
func (s *collectionService) List(ctx context.Context, profileID uuid.UUID) ([]dto.CollectionSummary, error) {
	collections, err := s.repo.FindAll(ctx, crud.Specification[entity.Collection]{
		Model: entity.Collection{ProfileID: profileID},
	})
	if err != nil {
		return nil, err
	}

	return ezutil.MapSlice(collections, mapper.ToCollectionSummary), nil
}

// findOwned filters by owner in the query itself. A zero-value id would
// otherwise drop the ID condition (see crud.WhereBySpec) and match any of
// the profile's collections.
func (s *collectionService) findOwned(ctx context.Context, profileID, id uuid.UUID) (entity.Collection, error) {
	if id == uuid.Nil {
		return entity.Collection{}, ungerr.NotFoundError(collectionNotFoundMsg)
	}

	c, err := s.repo.FindFirst(ctx, crud.Specification[entity.Collection]{
		Model: entity.Collection{BaseEntity: crud.BaseEntity{ID: id}, ProfileID: profileID},
	})
	if err != nil {
		return entity.Collection{}, err
	}
	if c.IsZero() {
		return entity.Collection{}, ungerr.NotFoundError(collectionNotFoundMsg)
	}

	return c, nil
}

func (s *collectionService) Get(ctx context.Context, profileID, id uuid.UUID) (dto.CollectionSummary, error) {
	c, err := s.findOwned(ctx, profileID, id)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return mapper.ToCollectionSummary(c), nil
}

func (s *collectionService) Update(ctx context.Context, profileID, id uuid.UUID, req dto.CollectionRequest) (dto.CollectionSummary, error) {
	c, err := s.findOwned(ctx, profileID, id)
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

func (s *collectionService) Delete(ctx context.Context, profileID, id uuid.UUID) error {
	c, err := s.findOwned(ctx, profileID, id)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, c)
}
