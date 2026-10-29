package handler

import (
	"context"
	"net/http"

	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
)

type ProfileHandler struct {
	userSvc service.UserService
}

func NewProfileHandler(userSvc service.UserService) *ProfileHandler {
	return &ProfileHandler{userSvc: userSvc}
}

type profileInput struct{}

func (h *ProfileHandler) get(ctx context.Context, _ profileInput) (dto.ProfileSummary, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.ProfileSummary{}, err
	}

	return h.userSvc.Profile(ctx, dto.ProfileLookup{ProfileID: profileID})
}

// Routes sets Secured:true only sets OpenAPI security metadata; the router must pass
// SessionGuard to RegisterAll.
func (h *ProfileHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[profileInput, dto.ProfileSummary]{
			OperationID: "get-profile",
			Method:      http.MethodGet,
			Path:        "/profile",
			Summary:     "Get the caller's own profile: its display name and Slug",
			Tags:        []string{"profile"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.get,
		}),
	}
}
