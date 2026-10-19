package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ungerr"
)

const cardNotFoundMsg = "card not found"

// CardLookup is the one rule for resolving a Card id, shared by the inventory and
// binder services. A nil id matches no Card, so it is never sent to the database,
// and a Card of an unpublished Expansion Set counts as absent.
type CardLookup interface {
	// CardExists reports whether the Card exists and is visible to a user.
	CardExists(ctx context.Context, cardID uuid.UUID) (bool, error)
	// RequireCard is CardExists, with the not-found AppError for a Card that does not exist.
	RequireCard(ctx context.Context, cardID uuid.UUID) error
}

type cardLookup struct {
	cards repository.CardRepository
}

func NewCardLookup(cards repository.CardRepository) CardLookup {
	return &cardLookup{cards: cards}
}

func (s *cardLookup) CardExists(ctx context.Context, cardID uuid.UUID) (bool, error) {
	if cardID == uuid.Nil {
		return false, nil
	}

	return s.cards.Exists(ctx, cardID)
}

func (s *cardLookup) RequireCard(ctx context.Context, cardID uuid.UUID) error {
	exists, err := s.CardExists(ctx, cardID)
	if err != nil {
		return err
	}
	if !exists {
		return ungerr.NotFoundError(cardNotFoundMsg)
	}

	return nil
}
