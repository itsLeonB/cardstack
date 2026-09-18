package handler

import (
	"context"
	"net/http"

	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
)

type HealthHandler struct {
	healthSvc service.HealthService
}

func NewHealthHandler(healthSvc service.HealthService) *HealthHandler {
	return &HealthHandler{healthSvc}
}

type HealthInput struct{}

func (h *HealthHandler) getHealth(_ context.Context, _ HealthInput) (service.HealthStatus, error) {
	return h.healthSvc.Check(), nil
}

// Routes returns every route HealthHandler exposes, for registration via
// endpoint.RegisterAll.
func (h *HealthHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[HealthInput, service.HealthStatus]{
			OperationID: "get-health",
			Method:      http.MethodGet,
			Path:        "/health",
			Summary:     "Health check",
			Tags:        []string{"health"},
			SuccessCode: http.StatusOK,
			Secured:     false,
			HandlerFunc: h.getHealth,
		}),
	}
}
