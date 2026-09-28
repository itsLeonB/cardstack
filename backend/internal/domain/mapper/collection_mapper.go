package mapper

import (
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
)

// ToCollectionSummary converts an entity.Collection into the collection
// service's response DTO.
func ToCollectionSummary(c entity.Collection) dto.CollectionSummary {
	return dto.CollectionSummary{
		ID:           c.ID,
		Title:        c.Title,
		Description:  c.Description,
		MaxCardCount: c.MaxCardCount,
	}
}
