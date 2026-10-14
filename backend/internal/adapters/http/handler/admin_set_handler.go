package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	"github.com/itsLeonB/ungerr"
)

// AdminSetHandler holds the Admin's Expansion Set routes (docs/adr/0019).
// RegisterRoutes mounts them behind authpkg.AdminOnly, like AdminHandler.
type AdminSetHandler struct {
	sets service.AdminSetService
}

func NewAdminSetHandler(sets service.AdminSetService) *AdminSetHandler {
	return &AdminSetHandler{sets: sets}
}

type createExpansionSetInput struct {
	Body struct {
		Code        string    `json:"code" required:"true" minLength:"1" maxLength:"32" pattern:"^[A-Za-z0-9][A-Za-z0-9_-]*$" doc:"The set's code, e.g. SM-P. A code the set's Game already has is a 409; the code cannot be changed afterwards."`
		Name        string    `json:"name" required:"true" minLength:"1" maxLength:"200" pattern:"\\S" doc:"The set's name."`
		ReleaseDate string    `json:"releaseDate" required:"true" format:"date" doc:"Release date, YYYY-MM-DD."`
		SeriesID    uuid.UUID `json:"seriesId" required:"true" doc:"An existing Series (see GET /catalog/series)."`
	}
}

type updateExpansionSetInput struct {
	ID   uuid.UUID `path:"id" doc:"Expansion Set ID"`
	Body struct {
		Name        string    `json:"name" required:"true" minLength:"1" maxLength:"200" pattern:"\\S" doc:"The set's name."`
		ReleaseDate string    `json:"releaseDate" required:"true" format:"date" doc:"Release date, YYYY-MM-DD."`
		SeriesID    uuid.UUID `json:"seriesId" required:"true" doc:"An existing Series of the set's Game (422 otherwise). The set's code is not editable."`
	}
}

func (h *AdminSetHandler) createExpansionSet(ctx context.Context, in createExpansionSetInput) (dto.ExpansionSetSummary, error) {
	releaseDate, err := time.Parse(time.DateOnly, in.Body.ReleaseDate)
	if err != nil {
		return dto.ExpansionSetSummary{}, ungerr.BadRequestError("invalid releaseDate")
	}

	return h.sets.CreateExpansionSet(ctx, dto.ExpansionSetCreate{
		Code:        in.Body.Code,
		Name:        in.Body.Name,
		ReleaseDate: releaseDate,
		SeriesID:    in.Body.SeriesID,
	})
}

func (h *AdminSetHandler) updateExpansionSet(ctx context.Context, in updateExpansionSetInput) (dto.ExpansionSetSummary, error) {
	releaseDate, err := time.Parse(time.DateOnly, in.Body.ReleaseDate)
	if err != nil {
		return dto.ExpansionSetSummary{}, ungerr.BadRequestError("invalid releaseDate")
	}

	return h.sets.UpdateExpansionSet(ctx, dto.ExpansionSetUpdate{
		ID:          in.ID,
		Name:        in.Body.Name,
		ReleaseDate: releaseDate,
		SeriesID:    in.Body.SeriesID,
	})
}

// Routes returns every route AdminSetHandler exposes, for registration via
// endpoint.RegisterAll.
func (h *AdminSetHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[createExpansionSetInput, dto.ExpansionSetSummary]{
			OperationID: "create-expansion-set",
			Method:      http.MethodPost,
			Path:        "/admin/expansion-sets",
			Summary:     "Create an Expansion Set and mark it Curated; a code its Game already has is a 409",
			Tags:        []string{"admin"},
			SuccessCode: http.StatusCreated,
			Secured:     true,
			HandlerFunc: h.createExpansionSet,
		}),
		endpoint.New(endpoint.Endpoint[updateExpansionSetInput, dto.ExpansionSetSummary]{
			OperationID: "update-expansion-set",
			Method:      http.MethodPut,
			Path:        "/admin/expansion-sets/{id}",
			Summary:     "Replace an Expansion Set's name, release date and Series and mark it Curated; the code stays",
			Tags:        []string{"admin"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.updateExpansionSet,
		}),
	}
}
