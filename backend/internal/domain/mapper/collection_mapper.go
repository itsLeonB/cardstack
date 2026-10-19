package mapper

import (
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
)

// ToCollectionSummary converts a Collection into its summary; binder is nil for a
// Collection without a Layout.
func ToCollectionSummary(c entity.Collection, cardCount int, binder *dto.BinderLayout) dto.CollectionSummary {
	return dto.CollectionSummary{
		ID:           c.ID,
		Title:        c.Title,
		Description:  c.Description,
		MaxCardCount: c.MaxCardCount,
		CardCount:    cardCount,
		Binder:       binder,
	}
}
