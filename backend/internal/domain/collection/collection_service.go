package collection

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
)

// CollectionService is the authenticated Collections CRUD surface (ticket
// 06). Every method scopes to the calling userID; Get/Update/Delete return
// ErrCollectionNotFound both when no such Collection exists and when one
// exists but belongs to another user - the two cases are indistinguishable
// to the caller by design, so a Collection's existence isn't leaked to a
// user who doesn't own it.
type CollectionService interface {
	Create(ctx context.Context, userID uuid.UUID, req dto.CreateCollectionRequest) (dto.CollectionSummary, error)
	List(ctx context.Context, userID uuid.UUID) ([]dto.CollectionSummary, error)
	Get(ctx context.Context, userID, id uuid.UUID) (dto.CollectionSummary, error)
	Update(ctx context.Context, userID, id uuid.UUID, req dto.UpdateCollectionRequest) (dto.CollectionSummary, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
}

type collectionService struct {
	repo CollectionRepository
}

// NewCollectionService builds a CollectionService backed by repo.
func NewCollectionService(repo CollectionRepository) CollectionService {
	return &collectionService{repo: repo}
}

func (s *collectionService) Create(ctx context.Context, userID uuid.UUID, req dto.CreateCollectionRequest) (dto.CollectionSummary, error) {
	c, err := s.repo.Create(ctx, entity.Collection{
		UserID:       userID,
		Title:        req.Title,
		Description:  req.Description,
		MaxCardCount: req.MaxCardCount,
	})
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return mapper.ToCollectionSummary(c), nil
}

func (s *collectionService) List(ctx context.Context, userID uuid.UUID) ([]dto.CollectionSummary, error) {
	collections, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	summaries := make([]dto.CollectionSummary, len(collections))
	for i, c := range collections {
		summaries[i] = mapper.ToCollectionSummary(c)
	}

	return summaries, nil
}

// findOwned looks up id and returns it only if userID owns it, otherwise
// ErrCollectionNotFound (see CollectionService's doc comment).
func (s *collectionService) findOwned(ctx context.Context, userID, id uuid.UUID) (entity.Collection, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return entity.Collection{}, err
	}
	if c.UserID != userID {
		return entity.Collection{}, ErrCollectionNotFound
	}

	return c, nil
}

func (s *collectionService) Get(ctx context.Context, userID, id uuid.UUID) (dto.CollectionSummary, error) {
	c, err := s.findOwned(ctx, userID, id)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return mapper.ToCollectionSummary(c), nil
}

func (s *collectionService) Update(ctx context.Context, userID, id uuid.UUID, req dto.UpdateCollectionRequest) (dto.CollectionSummary, error) {
	c, err := s.findOwned(ctx, userID, id)
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

func (s *collectionService) Delete(ctx context.Context, userID, id uuid.UUID) error {
	c, err := s.findOwned(ctx, userID, id)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, c)
}
