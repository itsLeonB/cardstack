package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

const unknownSeriesMsg = "series not found"

// adminLocaleCode is the Locale of a set an Admin creates by hand: the
// Indonesian print edition, the same code the Asia ingester files its sets under.
const adminLocaleCode = "id"

// AdminSetService is the Admin's write path for Expansion Sets (docs/adr/0019).
type AdminSetService interface {
	// UpsertExpansionSet creates the set with req.Code in its Series' Game, or
	// updates the existing one whatever its origin, and marks it Curated. An
	// unknown Series is a 422.
	UpsertExpansionSet(ctx context.Context, req dto.ExpansionSetUpsert) (dto.ExpansionSetSummary, error)
}

type adminSetService struct {
	tx      crud.Transactor
	series  crud.Repository[entity.Series]
	sets    crud.Repository[entity.ExpansionSet]
	locales crud.Repository[entity.Locale]
	images  mapper.ImageHost
}

func NewAdminSetService(
	tx crud.Transactor,
	series crud.Repository[entity.Series],
	sets crud.Repository[entity.ExpansionSet],
	locales crud.Repository[entity.Locale],
	images mapper.ImageHost,
) AdminSetService {
	return &adminSetService{tx: tx, series: series, sets: sets, locales: locales, images: images}
}

func (s *adminSetService) UpsertExpansionSet(ctx context.Context, req dto.ExpansionSetUpsert) (dto.ExpansionSetSummary, error) {
	var written entity.ExpansionSet
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		series, err := s.getSeries(ctx, req.SeriesID)
		if err != nil {
			return err
		}

		set, err := s.sets.FindFirst(ctx, crud.Specification[entity.ExpansionSet]{
			Model:     entity.ExpansionSet{GameID: series.GameID, Code: req.Code},
			ForUpdate: true,
		})
		if err != nil {
			return err
		}
		isNew := set.IsZero()

		if isNew {
			locale, err := s.findOrCreateLocale(ctx)
			if err != nil {
				return err
			}
			set = entity.ExpansionSet{GameID: series.GameID, Code: req.Code, LocaleID: locale.ID}
		}
		set.Name = req.Name
		set.SeriesID = &series.ID
		set.ReleaseDate = &req.ReleaseDate
		set.Curated = true

		if isNew {
			written, err = s.sets.Insert(ctx, set)
		} else {
			written, err = s.sets.Update(ctx, set)
		}
		return err
	})
	if err != nil {
		return dto.ExpansionSetSummary{}, err
	}

	return mapper.ToExpansionSetSummary(s.images, written), nil
}

// getSeries returns the Series with id, or the 422 an unknown one is. A nil id
// would drop the id condition from the lookup and match any Series.
func (s *adminSetService) getSeries(ctx context.Context, id uuid.UUID) (entity.Series, error) {
	if id != uuid.Nil {
		series, err := s.series.FindFirst(ctx, crud.Specification[entity.Series]{
			Model: entity.Series{BaseEntity: crud.BaseEntity{ID: id}},
		})
		if err != nil || !series.IsZero() {
			return series, err
		}
	}
	return entity.Series{}, ungerr.UnprocessableEntityError(unknownSeriesMsg)
}

func (s *adminSetService) findOrCreateLocale(ctx context.Context) (entity.Locale, error) {
	locale, err := s.locales.FindFirst(ctx, crud.Specification[entity.Locale]{Model: entity.Locale{Code: adminLocaleCode}})
	if err != nil || !locale.IsZero() {
		return locale, err
	}
	return s.locales.Insert(ctx, entity.Locale{Code: adminLocaleCode})
}
