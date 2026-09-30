package routes

import (
	"github.com/danielgtaylor/huma/v2"
	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/handler"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
)

// RegisterRoutes mounts every Huma operation on api. It takes the already
// constructed *provider.Services rather than the full *provider.Providers
// so cmd/genspec can register routes without booting a DB connection.
func RegisterRoutes(api huma.API, services *provider.Services) {
	healthHandler := handler.NewHealthHandler(services.Health)
	authHandler := handler.NewAuthHandler(services.Auth, services.Profiles)
	catalogHandler := handler.NewCatalogHandler(services.Catalog)
	collectionHandler := handler.NewCollectionHandler(services.Collection)

	endpoint.RegisterAll(api, healthHandler.Routes())
	endpoint.RegisterAll(api, authHandler.Routes())
	endpoint.RegisterAll(api, catalogHandler.Routes())
	// Secured:true only sets OpenAPI metadata, so enforce the session here.
	sessionGuard := authpkg.SessionGuard(api, services.Auth, authpkg.NewTransport(config.Global.Auth), services.Profiles)
	endpoint.RegisterAll(api, collectionHandler.Routes(), sessionGuard)
}
