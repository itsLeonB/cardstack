package routes

import (
	"github.com/danielgtaylor/huma/v2"
	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/handler"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/ratelimit"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
)

// RegisterRoutes mounts every Huma operation on api. It takes the already
// constructed *provider.Services rather than the full *provider.Providers
// so cmd/genspec can register routes without booting a DB connection.
func RegisterRoutes(api huma.API, services *provider.Services, limits ratelimit.Limits) {
	healthHandler := handler.NewHealthHandler(services.Health)
	catalogHandler := handler.NewCatalogHandler(services.Catalog)
	collectionHandler := handler.NewCollectionHandler(services.Collection)
	inventoryHandler := handler.NewInventoryHandler(services.Inventory)
	matchHandler := handler.NewMatchHandler(services.Match)

	// Secured:true only sets OpenAPI metadata. The guard is what decides: a
	// route group either allows Guests or rejects them with 401, and a bad token
	// is a 401 on both.
	guestsAllowed := authpkg.Guard(api, services.Verifier, services.Users, true)
	private := authpkg.Guard(api, services.Verifier, services.Users, false)
	// Name-searchable card lists and facets are the costliest queries, so they
	// get their own tighter buckets.
	perUser := ratelimit.PerUser(api, limits, map[string]*ratelimit.Limiter{
		handler.OpSearchCatalogCards:        limits.Search,
		handler.OpListCollectionEntries:     limits.Search,
		handler.OpListMasterInventory:       limits.Search,
		handler.OpListCatalogFacets:         limits.Facets,
		handler.OpListCollectionFacets:      limits.Facets,
		handler.OpListMasterInventoryFacets: limits.Facets,
	})

	endpoint.RegisterAll(api, healthHandler.Routes())
	endpoint.RegisterAll(api, catalogHandler.Routes(), guestsAllowed, perUser)
	endpoint.RegisterAll(api, collectionHandler.Routes(), private, perUser)
	endpoint.RegisterAll(api, inventoryHandler.Routes(), private, perUser)
	endpoint.RegisterAll(api, matchHandler.Routes(), private, perUser)
}
