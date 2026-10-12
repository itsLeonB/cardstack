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

// OpUpsertExpansionSet is the operation id of the Admin's set write.
const OpUpsertExpansionSet = "upsert-expansion-set"

// AdminSetHandler holds the Admin's Expansion Set routes (docs/adr/0019).
// RegisterRoutes mounts them behind authpkg.AdminOnly, like AdminHandler.
type AdminSetHandler struct {
	sets service.AdminSetService
}

func NewAdminSetHandler(sets service.AdminSetService) *AdminSetHandler {
	return &AdminSetHandler{sets: sets}
}

type upsertExpansionSetInput struct {
	Code string `path:"code" minLength:"1" maxLength:"32" pattern:"^[A-Za-z0-9][A-Za-z0-9_-]*$" doc:"The set's code, e.g. SM-P. Creates the set, or updates the one with this code."`
	Body struct {
		Name        string    `json:"name" required:"true" minLength:"1" maxLength:"200" pattern:"\\S" doc:"The set's name."`
		ReleaseDate string    `json:"releaseDate" required:"true" format:"date" doc:"Release date, YYYY-MM-DD."`
		SeriesID    uuid.UUID `json:"seriesId" required:"true" doc:"An existing Series (see GET /catalog/series)."`
	}
}

func (h *AdminSetHandler) upsertExpansionSet(ctx context.Context, in upsertExpansionSetInput) (dto.ExpansionSetSummary, error) {
	releaseDate, err := time.Parse(time.DateOnly, in.Body.ReleaseDate)
	if err != nil {
		return dto.ExpansionSetSummary{}, ungerr.BadRequestError("invalid releaseDate")
	}

	return h.sets.UpsertExpansionSet(ctx, dto.ExpansionSetUpsert{
		Code:        in.Code,
		Name:        in.Body.Name,
		ReleaseDate: releaseDate,
		SeriesID:    in.Body.SeriesID,
	})
}

// Routes returns every route AdminSetHandler exposes, for registration via
// endpoint.RegisterAll.
func (h *AdminSetHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[upsertExpansionSetInput, dto.ExpansionSetSummary]{
			OperationID: OpUpsertExpansionSet,
			Method:      http.MethodPut,
			Path:        "/admin/expansion-sets/{code}",
			Summary:     "Create or update an Expansion Set and mark it Curated",
			Tags:        []string{"admin"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.upsertExpansionSet,
		}),
	}
}
