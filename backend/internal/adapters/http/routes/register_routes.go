package routes

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/handler"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
)

// RegisterRoutes mounts every Huma operation on api. It takes the already
// constructed *provider.Services rather than the full *provider.Providers
// so cmd/genspec can register routes without booting a DB connection.
func RegisterRoutes(api huma.API, services *provider.Services) {
	healthHandler := handler.NewHealthHandler(services.Health)
	authHandler := handler.NewAuthHandler(services.Auth)

	endpoint.RegisterAll(api, healthHandler.Routes())
	endpoint.RegisterAll(api, authHandler.Routes())
}
