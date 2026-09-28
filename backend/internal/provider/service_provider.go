package provider

import (
	"github.com/google/wire"
	coreservice "github.com/itsLeonB/cardstack/backend/internal/adapters/core/service"
	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/repository"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	authkit "github.com/itsLeonB/go-authkit"
)

// ServiceSet is the wire provider set for the top-level Services.
var ServiceSet = wire.NewSet(ProvideCatalogService, ProvideServices)

type Services struct {
	Health   service.HealthService
	Auth     *authkit.AuthKit
	Profiles authpkg.ProfileLookup
	Catalog  service.CatalogService
}

// ProvideCatalogService builds the catalog service over ds's DB. It's a
// separate provider (rather than being built inline in ProvideServices,
// like Health is) because it needs a DataSources to build its repository —
// ProvideServices otherwise takes only already-built, DB-free dependencies
// so cmd/genspec can call it without a DB (see ProvideServices's own doc
// comment).
func ProvideCatalogService(ds *DataSources) service.CatalogService {
	return coreservice.NewCatalogService(repository.NewCatalogRepository(ds.Gorm))
}

// ProvideServices takes just the already-built *authkit.AuthKit,
// auth.ProfileLookup and service.CatalogService (not the DB they're
// ultimately backed by) so cmd/genspec can keep calling this with
// throwaway, DB-free values of its own construction — see
// cmd/genspec/main.go — without this function needing to know or care where
// they came from.
func ProvideServices(kit *authkit.AuthKit, profiles authpkg.ProfileLookup, catalog service.CatalogService) *Services {
	return &Services{
		Health:   coreservice.NewHealthService(),
		Auth:     kit,
		Profiles: profiles,
		Catalog:  catalog,
	}
}
