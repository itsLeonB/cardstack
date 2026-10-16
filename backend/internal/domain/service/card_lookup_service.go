package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

const cardNotFoundMsg = "card not found"

// CardLookup is the one rule for resolving a Card id, shared by the inventory and
// binder services. A nil id matches no Card, so it is never sent to the database.
type CardLookup interface {
	// FindCard returns the Card, or the zero value with no error when there is none.
	FindCard(ctx context.Context, cardID uuid.UUID) (entity.Card, error)
	// GetCard is FindCard, with the not-found AppError for a Card that does not exist.
	GetCard(ctx context.Context, cardID uuid.UUID) (entity.Card, error)
}

type cardLookup struct {
	cards crud.Repository[entity.Card]
}

func NewCardLookup(cards crud.Repository[entity.Card]) CardLookup {
	return &cardLookup{cards: cards}
}

func (s *cardLookup) FindCard(ctx context.Context, cardID uuid.UUID) (entity.Card, error) {
	if cardID == uuid.Nil {
		return entity.Card{}, nil
	}

	return s.cards.FindFirst(ctx, crud.Specification[entity.Card]{
		Model: entity.Card{BaseEntity: crud.BaseEntity{ID: cardID}},
	})
}

func (s *cardLookup) GetCard(ctx context.Context, cardID uuid.UUID) (entity.Card, error) {
	card, err := s.FindCard(ctx, cardID)
	if err != nil {
		return entity.Card{}, err
	}
	if card.IsZero() {
		return entity.Card{}, ungerr.NotFoundError(cardNotFoundMsg)
	}

	return card, nil
}
