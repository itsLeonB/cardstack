package mapper

import (
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
)

func ToCollectionSummary(c entity.Collection) dto.CollectionSummary {
	return dto.CollectionSummary{
		ID:           c.ID,
		Title:        c.Title,
		Description:  c.Description,
		MaxCardCount: c.MaxCardCount,
	}
}
