package auth

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	"github.com/itsLeonB/ungerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type whoamiOutput struct {
	Body struct {
		Guest     bool   `json:"guest"`
		ProfileID string `json:"profileId"`
	}
}

func newGuardedAPI(t *testing.T, allowGuests bool) (humatest.TestAPI, *mocks.MockTokenVerifier, *mocks.MockUserService) {
	t.Helper()
	verifier := mocks.NewMockTokenVerifier(t)
	users := mocks.NewMockUserService(t)
	_, api := humatest.New(t, httpapi.NewConfig())
	huma.Register(api, huma.Operation{
		OperationID: "whoami",
		Method:      http.MethodGet,
		Path:        "/whoami",
		Middlewares: huma.Middlewares{Guard(api, verifier, users, allowGuests)},
	}, func(ctx context.Context, _ *struct{}) (*whoamiOutput, error) {
		caller := CallerFrom(ctx)
		out := &whoamiOutput{}
		out.Body.Guest = caller.IsGuest()
		out.Body.ProfileID = caller.ProfileID.String()
		return out, nil
	})
	return api, verifier, users
}

func TestGuard_NoHeaderIsAGuestWhereGuestsAreAllowed(t *testing.T) {
	api, _, _ := newGuardedAPI(t, true)

	resp := api.Get("/whoami")

	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.JSONEq(t, `{"guest":true,"profileId":"`+uuid.Nil.String()+`"}`, resp.Body.String())
}

func TestGuard_NoHeaderIsRejectedOnPrivateRoutes(t *testing.T) {
	api, _, _ := newGuardedAPI(t, false)

	assert.Equal(t, http.StatusUnauthorized, api.Get("/whoami").Code)
}

func TestGuard_ValidTokenIsAuthenticated(t *testing.T) {
	for _, allowGuests := range []bool{true, false} {
		api, verifier, users := newGuardedAPI(t, allowGuests)
		identity := dto.AuthIdentity{Provider: "clerk", Subject: "user_1", Email: "a@example.com"}
		caller := dto.CallerIdentity{UserID: uuid.New(), ProfileID: uuid.New()}
		verifier.EXPECT().Verify(mock.Anything, "good-token").Return(identity, nil).Once()
		users.EXPECT().ResolveCaller(mock.Anything, identity).Return(caller, nil).Once()

		resp := api.Get("/whoami", "Authorization: Bearer good-token")

		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		assert.JSONEq(t, `{"guest":false,"profileId":"`+caller.ProfileID.String()+`"}`, resp.Body.String())
	}
}

func TestGuard_BearerSchemeIsCaseInsensitive(t *testing.T) {
	api, verifier, users := newGuardedAPI(t, false)
	verifier.EXPECT().Verify(mock.Anything, "good-token").Return(dto.AuthIdentity{}, nil).Once()
	users.EXPECT().ResolveCaller(mock.Anything, dto.AuthIdentity{}).Return(dto.CallerIdentity{ProfileID: uuid.New()}, nil).Once()

	assert.Equal(t, http.StatusOK, api.Get("/whoami", "Authorization: bearer good-token").Code)
}

func TestGuard_InvalidTokenIs401NeverAGuest(t *testing.T) {
	for _, allowGuests := range []bool{true, false} {
		api, verifier, _ := newGuardedAPI(t, allowGuests)
		verifier.EXPECT().Verify(mock.Anything, "bad-token").Return(dto.AuthIdentity{}, ungerr.UnauthorizedError("invalid or expired token")).Once()

		resp := api.Get("/whoami", "Authorization: Bearer bad-token")

		assert.Equal(t, http.StatusUnauthorized, resp.Code, "allowGuests=%v: %s", allowGuests, resp.Body.String())
	}
}

func TestGuard_MalformedHeaderIs401WithoutCallingTheVerifier(t *testing.T) {
	for _, header := range []string{"Basic abc", "Bearer", "Bearer ", "good-token"} {
		api, _, _ := newGuardedAPI(t, true)

		resp := api.Get("/whoami", "Authorization: "+header)

		assert.Equal(t, http.StatusUnauthorized, resp.Code, "header %q: %s", header, resp.Body.String())
	}
}

func TestGuard_VerifierOutageIsARedacted500(t *testing.T) {
	api, verifier, _ := newGuardedAPI(t, true)
	verifier.EXPECT().Verify(mock.Anything, "any").Return(dto.AuthIdentity{}, errors.New("secret internals")).Once()

	resp := api.Get("/whoami", "Authorization: Bearer any")

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.NotContains(t, resp.Body.String(), "secret internals")
}

func TestGuard_ProvisioningFailureIsNotAGuest(t *testing.T) {
	api, verifier, users := newGuardedAPI(t, true)
	verifier.EXPECT().Verify(mock.Anything, "any").Return(dto.AuthIdentity{}, nil).Once()
	users.EXPECT().ResolveCaller(mock.Anything, dto.AuthIdentity{}).Return(dto.CallerIdentity{}, errors.New("db down")).Once()

	assert.Equal(t, http.StatusInternalServerError, api.Get("/whoami", "Authorization: Bearer any").Code)
}
