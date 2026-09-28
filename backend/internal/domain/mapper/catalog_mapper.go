// Package mapper converts persistence-shaped results (domain/entity types,
// repository query results) into the domain services' own DTOs, so a
// service like catalogService stays thin - calling into a mapper function
// rather than defining the conversion logic inline (ticket 05 PR review).
package mapper

import (
	"github.com/itsLeonB/cardstack/backend/internal/adapters/repository"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
)

// ToExpansionSetSummary converts an entity.ExpansionSet into the catalog
// service's browsable summary DTO.
func ToExpansionSetSummary(set entity.ExpansionSet) service.ExpansionSetSummary {
	return service.ExpansionSetSummary{
		ID:          set.ID,
		Code:        set.Code,
		Name:        set.Name,
		ReleaseDate: set.ReleaseDate,
	}
}

// ToCardSummary converts a repository.CardResult row (a Card already joined
// with its Rarity and Expansion Set) into the catalog service's search
// result DTO.
func ToCardSummary(r repository.CardResult) service.CardSummary {
	return service.CardSummary{
		ID: r.ID,
		ExpansionSet: service.ExpansionSetSummary{
			ID:          r.ExpansionSetID,
			Code:        r.ExpansionSetCode,
			Name:        r.ExpansionSetName,
			ReleaseDate: r.ExpansionSetReleaseDate,
		},
		LocalID:  r.LocalID,
		Name:     r.Name,
		Category: r.Category,
		Tags:     []string(r.Tags),
		Rarity: service.RaritySummary{
			ID:   r.RarityID,
			Code: r.RarityCode,
			Name: r.RarityName,
		},
		Illustrator: r.Illustrator,
		ImageURL:    r.ImageURL,
	}
}
