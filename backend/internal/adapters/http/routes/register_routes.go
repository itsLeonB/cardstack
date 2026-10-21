package routes

import (
	"errors"

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
//
// adminEmails are the Admins (ADMIN_EMAILS). The caller passes none unless the
// R2 settings are also present, and with none the admin routes are not
// registered at all (docs/adr/0019).
func RegisterRoutes(api huma.API, services *provider.Services, limits ratelimit.Limits, adminEmails []string) error {
	healthHandler := handler.NewHealthHandler(services.Health)
	catalogHandler := handler.NewCatalogHandler(services.Catalog)
	collectionHandler := handler.NewCollectionHandler(services.Collection)
	inventoryHandler := handler.NewInventoryHandler(services.Inventory)
	binderHandler := handler.NewBinderHandler(services.Binder, services.BinderSlot)
	binderPlacementHandler := handler.NewBinderPlacementHandler(services.BinderSlot)
	matchHandler := handler.NewMatchHandler(services.Match)
	scanPhotoHandler := handler.NewScanPhotoHandler(services.ScanPhoto)

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
	// The catalog serves Admins the unpublished sets too (GLOSSARY.md Published),
	// so it flags the caller rather than gating the route.
	endpoint.RegisterAll(api, catalogHandler.Routes(), guestsAllowed, perUser, authpkg.AdminFlag(adminEmails))
	endpoint.RegisterAll(api, collectionHandler.Routes(), private, perUser)
	endpoint.RegisterAll(api, inventoryHandler.Routes(), private, perUser)
	endpoint.RegisterAll(api, binderHandler.Routes(), private, perUser)
	endpoint.RegisterAll(api, binderPlacementHandler.Routes(), private, perUser)
	endpoint.RegisterAll(api, matchHandler.Routes(), private, perUser)
	endpoint.RegisterAll(api, scanPhotoHandler.Routes(), private, perUser)
	if err := markBinderNullable(api); err != nil {
		return err
	}

	if len(adminEmails) > 0 {
		// Guests pass Guard so AdminOnly can answer them with the same 404 as a
		// signed-in non-Admin. perUser runs first, so probing is rate-limited.
		adminOnly := authpkg.AdminOnly(api, adminEmails)
		endpoint.RegisterAll(api, handler.NewAdminHandler().Routes(), guestsAllowed, perUser, adminOnly)
		endpoint.RegisterAll(api, handler.NewAdminSetHandler(services.AdminSets).Routes(), guestsAllowed, perUser, adminOnly)
		endpoint.RegisterAll(api, handler.NewAdminCardsHandler(services.CardAdmin).Routes(), guestsAllowed, perUser, adminOnly)
	}

	return nil
}

// markBinderNullable makes CollectionSummary.binder nullable in the spec. Huma
// rejects `nullable` on a field whose schema is a $ref to an object, so the
// property is patched after registration: oneOf of the BinderLayout ref and
// null, the OpenAPI 3.1 way to say nullable, which generators read as
// BinderLayout | null.
func markBinderNullable(api huma.API) error {
	summary, ok := api.OpenAPI().Components.Schemas.Map()["CollectionSummary"]
	if !ok {
		return errors.New("markBinderNullable: CollectionSummary schema is not registered")
	}
	binder, ok := summary.Properties["binder"]
	if !ok {
		return errors.New("markBinderNullable: CollectionSummary has no binder property")
	}
	summary.Properties["binder"] = &huma.Schema{
		OneOf:       []*huma.Schema{{Ref: binder.Ref}, {Type: "null"}},
		Description: binder.Description,
	}

	return nil
}
