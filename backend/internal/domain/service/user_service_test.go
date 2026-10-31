package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type userFixture struct {
	ctx      context.Context
	users    *mocks.MockUserRepository
	profiles *mocks.MockRepository[entity.UserProfile]
	cache    *mocks.MockIdentityCache
	svc      UserService
	identity dto.AuthIdentity
}

// newUserFixture's Transactor mock runs the transaction body inline.
func newUserFixture(t *testing.T) userFixture {
	t.Helper()
	return newUserFixtureAdopting(t, false)
}

func newUserFixtureAdopting(t *testing.T, adopt AdoptByEmail) userFixture {
	t.Helper()
	f := userFixture{
		ctx:      context.Background(),
		users:    mocks.NewMockUserRepository(t),
		profiles: mocks.NewMockRepository[entity.UserProfile](t),
		cache:    mocks.NewMockIdentityCache(t),
		identity: dto.AuthIdentity{Provider: dto.AuthProviderClerk, Subject: "user_1", Email: "ada@example.com", Name: "Ada Lovelace"},
	}
	transactor := mocks.NewMockTransactor(t)
	transactor.EXPECT().WithinTransaction(f.ctx, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Maybe()
	f.svc = NewUserService(transactor, f.users, f.profiles, f.cache, adopt)
	return f
}

func (f userFixture) userSpec() crud.Specification[entity.User] {
	return crud.Specification[entity.User]{Model: entity.User{AuthProvider: f.identity.Provider, AuthSubject: f.identity.Subject}}
}

func profileSpec(userID uuid.UUID) crud.Specification[entity.UserProfile] {
	return crud.Specification[entity.UserProfile]{Model: entity.UserProfile{UserID: userID}}
}

// newProfile matches the profile the service inserts: its id is generated in
// the service, so the match checks that the Slug is the stem plus the id's tail.
func newProfile(userID uuid.UUID, name, stem string) any {
	return mock.MatchedBy(func(p entity.UserProfile) bool {
		return p.ID != uuid.Nil && p.UserID == userID && p.Name == name && p.Slug == stem+"-"+p.ID.String()[31:]
	})
}

// expectProvisioning expects a cache miss, the identity lock, and the mapping
// being cached for the caller the test resolves to.
func (f userFixture) expectProvisioning(caller dto.CallerIdentity) {
	f.cache.EXPECT().Get(f.identity).Return(dto.CallerIdentity{}, false).Once()
	f.users.EXPECT().LockIdentity(f.ctx, f.identity.Provider, f.identity.Subject).Return(nil).Once()
	f.cache.EXPECT().Set(f.identity, caller).Once()
}

func (f userFixture) existingUser() entity.User {
	return entity.User{BaseEntity: baseEntity(uuid.New()), AuthProvider: f.identity.Provider, AuthSubject: f.identity.Subject, Email: f.identity.Email}
}

func TestUserService_ResolveCaller_FirstUseCreatesUserAndProfile(t *testing.T) {
	f := newUserFixture(t)
	user := f.existingUser()
	profile := entity.UserProfile{BaseEntity: baseEntity(uuid.New()), UserID: user.ID, Name: "Ada Lovelace"}

	f.expectProvisioning(dto.CallerIdentity{UserID: user.ID, ProfileID: profile.ID})
	f.users.EXPECT().FindFirst(f.ctx, f.userSpec()).Return(entity.User{}, nil).Once()
	f.users.EXPECT().Insert(f.ctx, entity.User{AuthProvider: "clerk", AuthSubject: "user_1", Email: "ada@example.com"}).Return(user, nil).Once()
	f.profiles.EXPECT().FindFirst(f.ctx, profileSpec(user.ID)).Return(entity.UserProfile{}, nil).Once()
	f.profiles.EXPECT().Insert(f.ctx, newProfile(user.ID, "Ada Lovelace", "ada-lovelace")).Return(profile, nil).Once()

	caller, err := f.svc.ResolveCaller(f.ctx, f.identity)

	require.NoError(t, err)
	assert.Equal(t, dto.CallerIdentity{UserID: user.ID, ProfileID: profile.ID}, caller)
}

func TestUserService_ResolveCaller_NameFallsBackToEmailLocalPart(t *testing.T) {
	for _, name := range []string{"", "   "} {
		f := newUserFixture(t)
		f.identity.Name = name
		user := f.existingUser()

		created := entity.UserProfile{BaseEntity: baseEntity(uuid.New())}
		f.expectProvisioning(dto.CallerIdentity{UserID: user.ID, ProfileID: created.ID})
		f.users.EXPECT().FindFirst(f.ctx, f.userSpec()).Return(user, nil).Once()
		f.profiles.EXPECT().FindFirst(f.ctx, profileSpec(user.ID)).Return(entity.UserProfile{}, nil).Once()
		f.profiles.EXPECT().Insert(f.ctx, newProfile(user.ID, "ada", "ada")).Return(created, nil).Once()

		_, err := f.svc.ResolveCaller(f.ctx, f.identity)

		require.NoError(t, err, "name %q", name)
	}
}

// sqlStateError stands in for the driver's Postgres error (pgconn.PgError),
// which exposes its SQLSTATE through SQLState.
type sqlStateError string

func (e sqlStateError) Error() string    { return "postgres error " + string(e) }
func (e sqlStateError) SQLState() string { return string(e) }

// expectProvisionAttempts expects the database calls of n provisioning
// attempts for a new user and profile. Each attempt reruns the whole
// transaction, so every call repeats n times.
func (f userFixture) expectProvisionAttempts(n int, user entity.User) {
	f.cache.EXPECT().Get(f.identity).Return(dto.CallerIdentity{}, false).Once()
	f.users.EXPECT().LockIdentity(f.ctx, f.identity.Provider, f.identity.Subject).Return(nil).Times(n)
	f.users.EXPECT().FindFirst(f.ctx, f.userSpec()).Return(entity.User{}, nil).Times(n)
	f.users.EXPECT().Insert(f.ctx, entity.User{AuthProvider: "clerk", AuthSubject: "user_1", Email: "ada@example.com"}).Return(user, nil).Times(n)
	f.profiles.EXPECT().FindFirst(f.ctx, profileSpec(user.ID)).Return(entity.UserProfile{}, nil).Times(n)
}

func TestUserService_ResolveCaller_RetriesAProfileSlugClashWithAFreshID(t *testing.T) {
	f := newUserFixture(t)
	user := f.existingUser()
	profile := entity.UserProfile{BaseEntity: baseEntity(uuid.New()), UserID: user.ID, Name: "Ada Lovelace"}
	clash := ungerr.Wrap(sqlStateError("23505"), "error inserting data")

	f.expectProvisionAttempts(2, user)
	f.profiles.EXPECT().Insert(f.ctx, newProfile(user.ID, "Ada Lovelace", "ada-lovelace")).Return(entity.UserProfile{}, clash).Once()
	f.profiles.EXPECT().Insert(f.ctx, newProfile(user.ID, "Ada Lovelace", "ada-lovelace")).Return(profile, nil).Once()
	f.cache.EXPECT().Set(f.identity, dto.CallerIdentity{UserID: user.ID, ProfileID: profile.ID}).Once()

	caller, err := f.svc.ResolveCaller(f.ctx, f.identity)

	require.NoError(t, err)
	assert.Equal(t, dto.CallerIdentity{UserID: user.ID, ProfileID: profile.ID}, caller)
}

func TestUserService_ResolveCaller_GivesUpAfterTheAttemptBudget(t *testing.T) {
	f := newUserFixture(t)
	user := f.existingUser()
	clash := ungerr.Wrap(sqlStateError("23505"), "error inserting data")

	f.expectProvisionAttempts(maxProvisionAttempts, user)
	f.profiles.EXPECT().Insert(f.ctx, mock.Anything).Return(entity.UserProfile{}, clash).Times(maxProvisionAttempts)

	_, err := f.svc.ResolveCaller(f.ctx, f.identity)

	assert.Equal(t, clash, err)
}

func TestUserService_ResolveCaller_DoesNotRetryOtherInsertErrors(t *testing.T) {
	for label, insertErr := range map[string]error{
		"a not-null violation": ungerr.Wrap(sqlStateError("23502"), "error inserting data"),
		"a plain driver error": ungerr.Wrap(errors.New("connection reset"), "error inserting data"),
	} {
		t.Run(label, func(t *testing.T) {
			f := newUserFixture(t)
			f.expectProvisionAttempts(1, f.existingUser())
			f.profiles.EXPECT().Insert(f.ctx, mock.Anything).Return(entity.UserProfile{}, insertErr).Once()

			_, err := f.svc.ResolveCaller(f.ctx, f.identity)

			assert.Equal(t, insertErr, err)
		})
	}
}

func TestUserService_ResolveCaller_ExistingUserKeepsNameAndEmail(t *testing.T) {
	f := newUserFixture(t)
	f.identity.Name = "A New Name"
	user := f.existingUser()
	profile := entity.UserProfile{BaseEntity: baseEntity(uuid.New()), UserID: user.ID, Name: "Ada Lovelace"}

	f.expectProvisioning(dto.CallerIdentity{UserID: user.ID, ProfileID: profile.ID})
	f.users.EXPECT().FindFirst(f.ctx, f.userSpec()).Return(user, nil).Once()
	f.profiles.EXPECT().FindFirst(f.ctx, profileSpec(user.ID)).Return(profile, nil).Once()

	caller, err := f.svc.ResolveCaller(f.ctx, f.identity)

	require.NoError(t, err)
	assert.Equal(t, profile.ID, caller.ProfileID)
	// No Update or Insert is expected: the mocks fail the test on any other call.
}

func TestUserService_ResolveCaller_UpdatesEmailOnlyWhenItDiffers(t *testing.T) {
	f := newUserFixture(t)
	user := f.existingUser()
	user.Email = "old@example.com"
	updated := user
	updated.Email = f.identity.Email
	profile := entity.UserProfile{BaseEntity: baseEntity(uuid.New()), UserID: user.ID, Name: "Ada Lovelace"}

	f.expectProvisioning(dto.CallerIdentity{UserID: user.ID, ProfileID: profile.ID})
	f.users.EXPECT().FindFirst(f.ctx, f.userSpec()).Return(user, nil).Once()
	f.users.EXPECT().Update(f.ctx, updated).Return(updated, nil).Once()
	f.profiles.EXPECT().FindFirst(f.ctx, profileSpec(user.ID)).Return(profile, nil).Once()

	_, err := f.svc.ResolveCaller(f.ctx, f.identity)

	require.NoError(t, err)
}

func TestUserService_ResolveCaller_CacheHitSkipsTheDatabase(t *testing.T) {
	f := newUserFixture(t)
	cached := dto.CallerIdentity{UserID: uuid.New(), ProfileID: uuid.New()}
	f.cache.EXPECT().Get(f.identity).Return(cached, true).Once()

	caller, err := f.svc.ResolveCaller(f.ctx, f.identity)

	require.NoError(t, err)
	assert.Equal(t, cached, caller)
	// No repository call is expected: the mocks fail the test on any.
}

func TestUserService_ResolveCaller_RejectsIdentityWithoutProviderOrSubject(t *testing.T) {
	for _, identity := range []dto.AuthIdentity{
		{Provider: "", Subject: "user_1", Email: "a@example.com"},
		{Provider: "clerk", Subject: "", Email: "a@example.com"},
	} {
		f := newUserFixture(t)

		_, err := f.svc.ResolveCaller(f.ctx, identity)

		appErr, ok := err.(ungerr.AppError)
		require.True(t, ok, "%v", err)
		assert.Equal(t, http.StatusUnauthorized, appErr.HttpStatus())
	}
}

// A nil profile ID must never reach the repository, where it would match any
// profile; a missing profile is the same not-found.
func TestUserService_Profile_RefusesNilIDAndMissingProfile(t *testing.T) {
	f := newUserFixture(t)

	_, err := f.svc.Profile(f.ctx, dto.ProfileLookup{})
	appErr, ok := err.(ungerr.AppError)
	require.True(t, ok, "%v", err)
	assert.Equal(t, http.StatusNotFound, appErr.HttpStatus())

	id := uuid.New()
	f.profiles.EXPECT().FindFirst(f.ctx, crud.Specification[entity.UserProfile]{
		Model: entity.UserProfile{BaseEntity: crud.BaseEntity{ID: id}},
	}).Return(entity.UserProfile{}, nil)

	_, err = f.svc.Profile(f.ctx, dto.ProfileLookup{ProfileID: id})
	appErr, ok = err.(ungerr.AppError)
	require.True(t, ok, "%v", err)
	assert.Equal(t, http.StatusNotFound, appErr.HttpStatus())
}

func TestUserService_ResolveCaller_AdoptsTheCopiedUserOnlyWhenAdoptionIsOn(t *testing.T) {
	adopted := entity.User{BaseEntity: baseEntity(uuid.New()), Email: "Ada@Example.com"}
	profile := entity.UserProfile{BaseEntity: baseEntity(uuid.New()), UserID: adopted.ID, Name: "Ada"}

	t.Run("on: the adopted user and its profile are the caller", func(t *testing.T) {
		f := newUserFixtureAdopting(t, true)
		f.expectProvisioning(dto.CallerIdentity{UserID: adopted.ID, ProfileID: profile.ID})
		f.users.EXPECT().FindFirst(f.ctx, f.userSpec()).Return(entity.User{}, nil).Once()
		f.users.EXPECT().AdoptByEmail(f.ctx, f.identity.Provider, f.identity.Subject, f.identity.Email).Return(adopted, nil).Once()
		f.users.EXPECT().Update(f.ctx, entity.User{BaseEntity: adopted.BaseEntity, Email: f.identity.Email}).Return(adopted, nil).Once()
		f.profiles.EXPECT().FindFirst(f.ctx, profileSpec(adopted.ID)).Return(profile, nil).Once()

		caller, err := f.svc.ResolveCaller(f.ctx, f.identity)

		require.NoError(t, err)
		assert.Equal(t, dto.CallerIdentity{UserID: adopted.ID, ProfileID: profile.ID}, caller)
	})

	t.Run("on, no unique match: a new user is created", func(t *testing.T) {
		f := newUserFixtureAdopting(t, true)
		user := f.existingUser()
		f.expectProvisioning(dto.CallerIdentity{UserID: user.ID, ProfileID: profile.ID})
		f.users.EXPECT().FindFirst(f.ctx, f.userSpec()).Return(entity.User{}, nil).Once()
		f.users.EXPECT().AdoptByEmail(f.ctx, f.identity.Provider, f.identity.Subject, f.identity.Email).Return(entity.User{}, nil).Once()
		f.users.EXPECT().Insert(f.ctx, entity.User{AuthProvider: "clerk", AuthSubject: "user_1", Email: "ada@example.com"}).Return(user, nil).Once()
		f.profiles.EXPECT().FindFirst(f.ctx, profileSpec(user.ID)).Return(profile, nil).Once()

		_, err := f.svc.ResolveCaller(f.ctx, f.identity)

		require.NoError(t, err)
	})

	t.Run("on, a known identity: nothing is adopted", func(t *testing.T) {
		f := newUserFixtureAdopting(t, true)
		user := f.existingUser()
		f.expectProvisioning(dto.CallerIdentity{UserID: user.ID, ProfileID: profile.ID})
		f.users.EXPECT().FindFirst(f.ctx, f.userSpec()).Return(user, nil).Once()
		f.profiles.EXPECT().FindFirst(f.ctx, profileSpec(user.ID)).Return(profile, nil).Once()

		_, err := f.svc.ResolveCaller(f.ctx, f.identity)

		require.NoError(t, err)
	})

	t.Run("off: an unknown identity never looks for an email match", func(t *testing.T) {
		f := newUserFixture(t)
		user := f.existingUser()
		f.expectProvisioning(dto.CallerIdentity{UserID: user.ID, ProfileID: profile.ID})
		f.users.EXPECT().FindFirst(f.ctx, f.userSpec()).Return(entity.User{}, nil).Once()
		f.users.EXPECT().Insert(f.ctx, entity.User{AuthProvider: "clerk", AuthSubject: "user_1", Email: "ada@example.com"}).Return(user, nil).Once()
		f.profiles.EXPECT().FindFirst(f.ctx, profileSpec(user.ID)).Return(profile, nil).Once()

		_, err := f.svc.ResolveCaller(f.ctx, f.identity)

		require.NoError(t, err)
	})
}
