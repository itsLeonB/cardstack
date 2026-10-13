package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type adminSetFixture struct {
	ctx     context.Context
	series  entity.Series
	seriesR *mocks.MockRepository[entity.Series]
	sets    *mocks.MockRepository[entity.ExpansionSet]
	locales *mocks.MockRepository[entity.Locale]
	svc     AdminSetService
}

func newAdminSetFixture(t *testing.T) adminSetFixture {
	t.Helper()
	f := adminSetFixture{ctx: context.Background(), series: entity.Series{BaseEntity: baseEntity(uuid.New()), GameID: uuid.New()}}
	f.seriesR = mocks.NewMockRepository[entity.Series](t)
	f.sets = mocks.NewMockRepository[entity.ExpansionSet](t)
	f.locales = mocks.NewMockRepository[entity.Locale](t)
	transactor := mocks.NewMockTransactor(t)
	transactor.EXPECT().WithinTransaction(f.ctx, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Maybe()
	f.svc = NewAdminSetService(transactor, f.seriesR, f.sets, f.locales, testImages)
	return f
}

func (f adminSetFixture) req(release time.Time) dto.ExpansionSetUpsert {
	return dto.ExpansionSetUpsert{Code: "SM-P", Name: "Promo", ReleaseDate: release, SeriesID: f.series.ID}
}

func (f adminSetFixture) expectSeries() {
	f.seriesR.EXPECT().FindFirst(f.ctx, crud.Specification[entity.Series]{
		Model: entity.Series{BaseEntity: baseEntity(f.series.ID)},
	}).Return(f.series, nil).Once()
}

func (f adminSetFixture) expectLockedSet(found entity.ExpansionSet) {
	f.sets.EXPECT().FindFirst(f.ctx, crud.Specification[entity.ExpansionSet]{
		Model:     entity.ExpansionSet{GameID: f.series.GameID, Code: "SM-P"},
		ForUpdate: true,
	}).Return(found, nil).Once()
}

func (f adminSetFixture) expectLocale(found entity.Locale) {
	f.locales.EXPECT().FindFirst(f.ctx, crud.Specification[entity.Locale]{Model: entity.Locale{Code: adminLocaleCode}}).Return(found, nil).Once()
}

func TestAdminSetService_UnknownSeriesIsRefused(t *testing.T) {
	f := newAdminSetFixture(t)
	f.seriesR.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entity.Series{}, nil).Once()

	_, err := f.svc.UpsertExpansionSet(f.ctx, f.req(time.Now()))
	requireStatus(t, err, http.StatusUnprocessableEntity)

	// A nil id never reaches the repository, where it would match any Series.
	_, err = f.svc.UpsertExpansionSet(f.ctx, dto.ExpansionSetUpsert{Code: "SM-P"})
	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func TestAdminSetService_NewSetFiledUnderANewIndonesianLocale(t *testing.T) {
	f := newAdminSetFixture(t)
	release := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	locale := entity.Locale{BaseEntity: baseEntity(uuid.New()), Code: adminLocaleCode}
	f.expectSeries()
	f.expectLockedSet(entity.ExpansionSet{})
	f.expectLocale(entity.Locale{})
	f.locales.EXPECT().Insert(f.ctx, entity.Locale{Code: adminLocaleCode}).Return(locale, nil).Once()
	want := entity.ExpansionSet{
		GameID: f.series.GameID, Code: "SM-P", Name: "Promo", LocaleID: locale.ID, SeriesID: &f.series.ID, ReleaseDate: &release, Curated: true,
	}
	written := want
	written.BaseEntity = baseEntity(uuid.New())
	f.sets.EXPECT().Insert(f.ctx, want).Return(written, nil).Once()

	got, err := f.svc.UpsertExpansionSet(f.ctx, f.req(release))

	require.NoError(t, err)
	assert.Equal(t, dto.ExpansionSetSummary{ID: written.ID, Code: "SM-P", Name: "Promo", ReleaseDate: &release}, got)
}

func TestAdminSetService_NewSetReusesTheExistingLocale(t *testing.T) {
	f := newAdminSetFixture(t)
	release := time.Now()
	locale := entity.Locale{BaseEntity: baseEntity(uuid.New()), Code: adminLocaleCode}
	f.expectSeries()
	f.expectLockedSet(entity.ExpansionSet{})
	f.expectLocale(locale)
	f.sets.EXPECT().Insert(f.ctx, mock.MatchedBy(func(s entity.ExpansionSet) bool { return s.LocaleID == locale.ID && s.Curated })).
		Return(entity.ExpansionSet{BaseEntity: baseEntity(uuid.New())}, nil).Once()

	_, err := f.svc.UpsertExpansionSet(f.ctx, f.req(release))

	require.NoError(t, err)
}

func TestAdminSetService_ExistingSetIsUpdatedWhateverItsOriginAndMarkedCurated(t *testing.T) {
	f := newAdminSetFixture(t)
	release := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	ingested := entity.ExpansionSet{
		BaseEntity: baseEntity(uuid.New()), GameID: f.series.GameID, Code: "SM-P", Name: "Typo", LocaleID: uuid.New(), CoverKey: "sets/cover",
	}
	f.expectSeries()
	f.expectLockedSet(ingested)
	want := ingested
	want.Name, want.SeriesID, want.ReleaseDate, want.Curated = "Promo", &f.series.ID, &release, true
	f.sets.EXPECT().Update(f.ctx, want).Return(want, nil).Once()

	got, err := f.svc.UpsertExpansionSet(f.ctx, f.req(release))

	require.NoError(t, err)
	assert.Equal(t, ingested.ID, got.ID)
	assert.Equal(t, "Promo", got.Name)
	assert.Equal(t, testImageBase+"/sets/cover", got.ImageURL)
}
