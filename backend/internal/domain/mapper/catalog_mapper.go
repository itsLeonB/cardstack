package mapper

import (
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
)

// ToExpansionSetSummary converts an entity.ExpansionSet into the catalog
// service's browsable summary DTO.
func ToExpansionSetSummary(set entity.ExpansionSet) dto.ExpansionSetSummary {
	return dto.ExpansionSetSummary{
		ID:          set.ID,
		Code:        set.Code,
		Name:        set.Name,
		ReleaseDate: set.ReleaseDate,
	}
}

// ToCardSummary converts a repository.CardResult row (a Card already joined
// with its Rarity and Expansion Set) into the catalog service's search
// result DTO.
func ToCardSummary(r repository.CardResult) dto.CardSummary {
	return dto.CardSummary{
		ID: r.ID,
		ExpansionSet: dto.ExpansionSetSummary{
			ID:          r.ExpansionSetID,
			Code:        r.ExpansionSetCode,
			Name:        r.ExpansionSetName,
			ReleaseDate: r.ExpansionSetReleaseDate,
		},
		LocalID:  r.LocalID,
		Name:     r.Name,
		Category: r.Category,
		Tags:     []string(r.Tags),
		Rarity: dto.RaritySummary{
			ID:   r.RarityID,
			Code: r.RarityCode,
			Name: r.RarityName,
		},
		Illustrator: r.Illustrator,
		ImageURL:    r.ImageURL,
	}
}

func ToExpansionSetFacetOption(o repository.ExpansionSetFacetOption) dto.ExpansionSetFacetOption {
	return dto.ExpansionSetFacetOption{
		ExpansionSetSummary: ToExpansionSetSummary(o.ExpansionSet),
		SeriesID:            o.SeriesID,
		Available:           o.Available,
	}
}

func ToRarityFacetOption(o repository.RarityFacetOption) dto.RarityFacetOption {
	return dto.RarityFacetOption{RaritySummary: dto.RaritySummary{ID: o.ID, Code: o.Code, Name: o.Name}, Available: o.Available}
}

func ToStringFacetOption(o repository.StringFacetOption) dto.StringFacetOption {
	return dto.StringFacetOption{Value: o.Value, Available: o.Available}
}
