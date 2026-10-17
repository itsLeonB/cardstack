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

func (f adminSetFixture) createReq(release time.Time) dto.ExpansionSetCreate {
	return dto.ExpansionSetCreate{Code: "SM-P", Name: "Promo", ReleaseDate: release, SeriesID: f.series.ID}
}

func (f adminSetFixture) expectSeries(forUpdate bool) {
	f.seriesR.EXPECT().FindFirst(f.ctx, crud.Specification[entity.Series]{
		Model:     entity.Series{BaseEntity: baseEntity(f.series.ID)},
		ForUpdate: forUpdate,
	}).Return(f.series, nil).Once()
}

func (f adminSetFixture) expectSetWithCode(found entity.ExpansionSet) {
	f.sets.EXPECT().FindFirst(f.ctx, crud.Specification[entity.ExpansionSet]{
		Model: entity.ExpansionSet{GameID: f.series.GameID, Code: "SM-P"},
	}).Return(found, nil).Once()
}

func (f adminSetFixture) expectLockedSetByID(id uuid.UUID, found entity.ExpansionSet) {
	f.sets.EXPECT().FindFirst(f.ctx, crud.Specification[entity.ExpansionSet]{
		Model:     entity.ExpansionSet{BaseEntity: baseEntity(id)},
		ForUpdate: true,
	}).Return(found, nil).Once()
}

func (f adminSetFixture) expectLocale(found entity.Locale) {
	f.locales.EXPECT().FindFirst(f.ctx, crud.Specification[entity.Locale]{Model: entity.Locale{Code: adminLocaleCode}}).Return(found, nil).Once()
}

func TestAdminSetService_CreateWithUnknownSeriesIsRefused(t *testing.T) {
	f := newAdminSetFixture(t)
	f.seriesR.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entity.Series{}, nil).Once()

	_, err := f.svc.CreateExpansionSet(f.ctx, f.createReq(time.Now()))
	requireStatus(t, err, http.StatusUnprocessableEntity)

	// A nil id never reaches the repository, where it would match any Series.
	_, err = f.svc.CreateExpansionSet(f.ctx, dto.ExpansionSetCreate{Code: "SM-P"})
	requireStatus(t, err, http.StatusUnprocessableEntity)
}

func TestAdminSetService_NewSetFiledUnderANewIndonesianLocale(t *testing.T) {
	f := newAdminSetFixture(t)
	release := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	locale := entity.Locale{BaseEntity: baseEntity(uuid.New()), Code: adminLocaleCode}
	f.expectSeries(true)
	f.expectSetWithCode(entity.ExpansionSet{})
	f.expectLocale(entity.Locale{})
	f.locales.EXPECT().Insert(f.ctx, entity.Locale{Code: adminLocaleCode}).Return(locale, nil).Once()
	want := entity.ExpansionSet{
		GameID: f.series.GameID, Code: "SM-P", Name: "Promo", LocaleID: locale.ID, SeriesID: &f.series.ID, ReleaseDate: &release, CreatedBy: entity.SourceAdmin, UpdatedBy: entity.SourceAdmin,
	}
	written := want
	written.BaseEntity = baseEntity(uuid.New())
	f.sets.EXPECT().Insert(f.ctx, want).Return(written, nil).Once()

	got, err := f.svc.CreateExpansionSet(f.ctx, f.createReq(release))

	require.NoError(t, err)
	assert.Equal(t, dto.ExpansionSetSummary{ID: written.ID, Code: "SM-P", Name: "Promo", ReleaseDate: &release}, got)
}

func TestAdminSetService_NewSetReusesTheExistingLocale(t *testing.T) {
	f := newAdminSetFixture(t)
	release := time.Now()
	locale := entity.Locale{BaseEntity: baseEntity(uuid.New()), Code: adminLocaleCode}
	f.expectSeries(true)
	f.expectSetWithCode(entity.ExpansionSet{})
	f.expectLocale(locale)
	f.sets.EXPECT().Insert(f.ctx, mock.MatchedBy(func(s entity.ExpansionSet) bool {
		return s.LocaleID == locale.ID && s.CreatedBy == entity.SourceAdmin && s.UpdatedBy == entity.SourceAdmin
	})).
		Return(entity.ExpansionSet{BaseEntity: baseEntity(uuid.New())}, nil).Once()

	_, err := f.svc.CreateExpansionSet(f.ctx, f.createReq(release))

	require.NoError(t, err)
}

func TestAdminSetService_CreateOfAnExistingCodeIsAConflictWhateverItsOrigin(t *testing.T) {
	f := newAdminSetFixture(t)
	f.expectSeries(true)
	f.expectSetWithCode(entity.ExpansionSet{BaseEntity: baseEntity(uuid.New()), Code: "SM-P"})

	_, err := f.svc.CreateExpansionSet(f.ctx, f.createReq(time.Now()))

	requireStatus(t, err, http.StatusConflict)
}

func TestAdminSetService_UpdateReplacesNameDateAndSeriesKeepsCodeAndMarksCurated(t *testing.T) {
	f := newAdminSetFixture(t)
	release := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	otherSeries := uuid.New()
	ingested := entity.ExpansionSet{
		BaseEntity: baseEntity(uuid.New()), GameID: f.series.GameID, Code: "SM-P", Name: "Typo", LocaleID: uuid.New(),
		CoverKey: "sets/cover", SeriesID: &otherSeries,
	}
	f.expectLockedSetByID(ingested.ID, ingested)
	f.expectSeries(false)
	want := ingested
	want.Name, want.SeriesID, want.ReleaseDate, want.UpdatedBy = "Promo", &f.series.ID, &release, entity.SourceAdmin
	f.sets.EXPECT().Update(f.ctx, want).Return(want, nil).Once()

	got, err := f.svc.UpdateExpansionSet(f.ctx, dto.ExpansionSetUpdate{ID: ingested.ID, Name: "Promo", ReleaseDate: release, SeriesID: f.series.ID})

	require.NoError(t, err)
	assert.Equal(t, ingested.ID, got.ID)
	assert.Equal(t, "SM-P", got.Code)
	assert.Equal(t, "Promo", got.Name)
	assert.Equal(t, testImageBase+"/sets/cover", got.ImageURL)
}

func TestAdminSetService_UpdateOfAnUnknownSetIsNotFound(t *testing.T) {
	f := newAdminSetFixture(t)
	missing := uuid.New()
	f.expectLockedSetByID(missing, entity.ExpansionSet{})

	_, err := f.svc.UpdateExpansionSet(f.ctx, dto.ExpansionSetUpdate{ID: missing, SeriesID: f.series.ID})
	requireStatus(t, err, http.StatusNotFound)

	// A nil id never reaches the repository, where it would match any set.
	_, err = f.svc.UpdateExpansionSet(f.ctx, dto.ExpansionSetUpdate{SeriesID: f.series.ID})
	requireStatus(t, err, http.StatusNotFound)
}

func TestAdminSetService_UpdateWithAnUnknownSeriesOrOneOfAnotherGameIsRefused(t *testing.T) {
	f := newAdminSetFixture(t)
	set := entity.ExpansionSet{BaseEntity: baseEntity(uuid.New()), GameID: f.series.GameID, Code: "SM-P"}
	f.expectLockedSetByID(set.ID, set)
	f.seriesR.EXPECT().FindFirst(f.ctx, mock.Anything).Return(entity.Series{}, nil).Once()
	_, err := f.svc.UpdateExpansionSet(f.ctx, dto.ExpansionSetUpdate{ID: set.ID, SeriesID: f.series.ID})
	requireStatus(t, err, http.StatusUnprocessableEntity)

	foreign := set
	foreign.GameID = uuid.New()
	f.expectLockedSetByID(set.ID, foreign)
	f.expectSeries(false)
	_, err = f.svc.UpdateExpansionSet(f.ctx, dto.ExpansionSetUpdate{ID: set.ID, SeriesID: f.series.ID})
	requireStatus(t, err, http.StatusUnprocessableEntity)
}
