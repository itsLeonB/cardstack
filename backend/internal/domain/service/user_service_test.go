package service

import (
	"context"
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
	f.svc = NewUserService(transactor, f.users, f.profiles, f.cache)
	return f
}

func (f userFixture) userSpec() crud.Specification[entity.User] {
	return crud.Specification[entity.User]{Model: entity.User{AuthProvider: f.identity.Provider, AuthSubject: f.identity.Subject}}
}

func profileSpec(userID uuid.UUID) crud.Specification[entity.UserProfile] {
	return crud.Specification[entity.UserProfile]{Model: entity.UserProfile{UserID: userID}}
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
	f.profiles.EXPECT().Insert(f.ctx, entity.UserProfile{UserID: user.ID, Name: "Ada Lovelace"}).Return(profile, nil).Once()

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
		f.profiles.EXPECT().Insert(f.ctx, entity.UserProfile{UserID: user.ID, Name: "ada"}).Return(created, nil).Once()

		_, err := f.svc.ResolveCaller(f.ctx, f.identity)

		require.NoError(t, err, "name %q", name)
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
