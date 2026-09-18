package routes

import (
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
)

type stubHealthService struct{}

func (stubHealthService) Check() service.HealthStatus {
	return service.HealthStatus{Status: "ok"}
}

// TestRegisterRoutes_Health proves the Huma -> OpenAPI -> handler wiring
// actually serves GET /health end to end, envelope included.
func TestRegisterRoutes_Health(t *testing.T) {
	_, api := humatest.New(t, httpapi.NewConfig())

	RegisterRoutes(api, &provider.Services{Health: stubHealthService{}})

	resp := api.Get("/health")

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	const want = `{"data":{"status":"ok"}}`
	if got := strings.TrimSpace(resp.Body.String()); got != want {
		t.Fatalf("expected body %q, got %q", want, got)
	}
}
