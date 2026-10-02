package mapper

import (
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ezutil/v2"
)

// ToExpansionSetSummary converts an entity.ExpansionSet into the catalog
// service's browsable summary DTO.
func ToExpansionSetSummary(set entity.ExpansionSet) dto.ExpansionSetSummary {
	return dto.ExpansionSetSummary{
		ID:          set.ID,
		Code:        set.Code,
		Name:        set.Name,
		ReleaseDate: set.ReleaseDate,
		ImageURL:    set.ImageURL,
	}
}

// ToSeriesSummary converts an entity.Series plus its already-mapped
// Expansion Sets into the browsable summary DTO.
func ToSeriesSummary(sr entity.Series, sets []dto.ExpansionSetSummary) dto.SeriesSummary {
	return dto.SeriesSummary{ID: sr.ID, Code: sr.Code, Name: sr.Name, ExpansionSets: sets}
}

// ToRaritySummary converts an entity.Rarity into its summary DTO.
func ToRaritySummary(r entity.Rarity) dto.RaritySummary {
	return dto.RaritySummary{ID: r.ID, Code: r.Code, Name: r.Name}
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
			ImageURL:    r.ExpansionSetImageURL,
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

// ToRepoCardFilter copies the search criteria of a dto.CardFilter. Paging and
// collection scope are the caller's to set.
func ToRepoCardFilter(filter dto.CardFilter) repository.CardFilter {
	return repository.CardFilter{
		Name:            filter.Name,
		ExpansionSetIDs: filter.ExpansionSetIDs,
		LocalID:         filter.LocalID,
		RarityIDs:       filter.RarityIDs,
		Categories:      filter.Categories,
		Tags:            filter.Tags,
		CardIDs:         filter.CardIDs,
	}
}

func ToCatalogFacets(facets repository.CardFacets) dto.CatalogFacets {
	return dto.CatalogFacets{
		ExpansionSets: ezutil.MapSlice(facets.ExpansionSets, ToExpansionSetFacetOption),
		Rarities:      ezutil.MapSlice(facets.Rarities, ToRarityFacetOption),
		Categories:    ezutil.MapSlice(facets.Categories, ToStringFacetOption),
		Tags:          ezutil.MapSlice(facets.Tags, ToStringFacetOption),
	}
}
