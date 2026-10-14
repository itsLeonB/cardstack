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

const (
	unknownSeriesMsg   = "series not found"
	setCodeExistsMsg   = "an expansion set with this code already exists"
	seriesOtherGameMsg = "series belongs to another game than the expansion set"
)

// adminLocaleCode is the Locale of a set an Admin creates by hand: the
// Indonesian print edition, the same code the Asia ingester files its sets under.
const adminLocaleCode = "id"

// AdminSetService is the Admin's write path for Expansion Sets (docs/adr/0019).
type AdminSetService interface {
	// CreateExpansionSet creates the set with req.Code in its Series' Game and
	// marks it Curated. A code the Game already has is a conflict, never an
	// update; an unknown Series is a 422.
	CreateExpansionSet(ctx context.Context, req dto.ExpansionSetCreate) (dto.ExpansionSetSummary, error)
	// UpdateExpansionSet replaces the name, release date and Series of the set
	// with req.ID, whatever its origin, and marks it Curated. The code never
	// changes. A Series of another Game or an unknown one is a 422, an unknown
	// set a 404.
	UpdateExpansionSet(ctx context.Context, req dto.ExpansionSetUpdate) (dto.ExpansionSetSummary, error)
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

func (s *adminSetService) CreateExpansionSet(ctx context.Context, req dto.ExpansionSetCreate) (dto.ExpansionSetSummary, error) {
	var written entity.ExpansionSet
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		// Locking the Series serializes creates under it, so two creates of the
		// same code cannot both insert.
		series, err := s.getSeries(ctx, req.SeriesID, true)
		if err != nil {
			return err
		}

		existing, err := s.sets.FindFirst(ctx, crud.Specification[entity.ExpansionSet]{
			Model: entity.ExpansionSet{GameID: series.GameID, Code: req.Code},
		})
		if err != nil {
			return err
		}
		if !existing.IsZero() {
			return ungerr.ConflictError(setCodeExistsMsg)
		}

		locale, err := s.findOrCreateLocale(ctx)
		if err != nil {
			return err
		}
		written, err = s.sets.Insert(ctx, entity.ExpansionSet{
			GameID: series.GameID, Code: req.Code, LocaleID: locale.ID,
			Name: req.Name, SeriesID: &series.ID, ReleaseDate: &req.ReleaseDate, Curated: true,
		})
		return err
	})
	if err != nil {
		return dto.ExpansionSetSummary{}, err
	}

	return mapper.ToExpansionSetSummary(s.images, written), nil
}

func (s *adminSetService) UpdateExpansionSet(ctx context.Context, req dto.ExpansionSetUpdate) (dto.ExpansionSetSummary, error) {
	var written entity.ExpansionSet
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		set, err := s.getSet(ctx, req.ID)
		if err != nil {
			return err
		}
		series, err := s.getSeries(ctx, req.SeriesID, false)
		if err != nil {
			return err
		}
		if series.GameID != set.GameID {
			return ungerr.UnprocessableEntityError(seriesOtherGameMsg)
		}

		set.Name, set.SeriesID, set.ReleaseDate, set.Curated = req.Name, &series.ID, &req.ReleaseDate, true
		written, err = s.sets.Update(ctx, set)
		return err
	})
	if err != nil {
		return dto.ExpansionSetSummary{}, err
	}

	return mapper.ToExpansionSetSummary(s.images, written), nil
}

// getSet returns the set with id under a row lock, or a 404. A nil id would
// drop the id condition from the lookup and match any set.
func (s *adminSetService) getSet(ctx context.Context, id uuid.UUID) (entity.ExpansionSet, error) {
	if id != uuid.Nil {
		set, err := s.sets.FindFirst(ctx, crud.Specification[entity.ExpansionSet]{
			Model:     entity.ExpansionSet{BaseEntity: crud.BaseEntity{ID: id}},
			ForUpdate: true,
		})
		if err != nil || !set.IsZero() {
			return set, err
		}
	}
	return entity.ExpansionSet{}, ungerr.NotFoundError(adminSetNotFoundMsg)
}

// getSeries returns the Series with id, or the 422 an unknown one is. A nil id
// would drop the id condition from the lookup and match any Series.
func (s *adminSetService) getSeries(ctx context.Context, id uuid.UUID, forUpdate bool) (entity.Series, error) {
	if id != uuid.Nil {
		series, err := s.series.FindFirst(ctx, crud.Specification[entity.Series]{
			Model:     entity.Series{BaseEntity: crud.BaseEntity{ID: id}},
			ForUpdate: forUpdate,
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
