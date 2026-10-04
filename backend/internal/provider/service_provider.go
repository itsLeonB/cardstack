package provider

import (
	"github.com/google/wire"
	coreservice "github.com/itsLeonB/cardstack/backend/internal/adapters/core/service"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	catalogrepository "github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	crud "github.com/itsLeonB/go-crud"
)

// ServiceSet is the wire provider set for the top-level Services.
var ServiceSet = wire.NewSet(ProvideImageHost, ProvideUserService, ProvideCatalogService, ProvideCollectionService, ProvideInventoryService, ProvideServices)

type Services struct {
	Health     service.HealthService
	Verifier   auth.TokenVerifier
	Users      service.UserService
	Catalog    service.CatalogService
	Collection service.CollectionService
	Inventory  service.InventoryService
}

// ProvideUserService builds the user service over ds's DB. It's a separate
// provider (rather than being built inline in ProvideServices, like Health
// is) because it needs a DataSources — ProvideServices otherwise takes only
// already-built, DB-free dependencies so cmd/genspec can call it without a DB
// (see ProvideServices's own doc comment).
func ProvideUserService(ds *DataSources, users catalogrepository.UserRepository, cache service.IdentityCache) service.UserService {
	return service.NewUserService(crud.NewTransactor(ds.Gorm), users, crud.NewRepository[entity.UserProfile](ds.Gorm), cache)
}

// ProvideImageHost builds the host that turns hosted-image keys into public
// addresses from IMAGE_BASE_URL (docs/adr/0016). An unset base address yields
// empty image addresses, never source addresses.
func ProvideImageHost() mapper.ImageHost {
	return mapper.NewImageHost(config.Global.BaseURL)
}

// ProvideCatalogService builds the catalog service over ds's DB. It's a
// separate provider (rather than being built inline in ProvideServices,
// like Health is) because it needs a DataSources to build its repository —
// ProvideServices otherwise takes only already-built, DB-free dependencies
// so cmd/genspec can call it without a DB (see ProvideServices's own doc
// comment).
func ProvideCatalogService(ds *DataSources, images mapper.ImageHost) service.CatalogService {
	return service.NewCatalogService(catalogrepository.NewCatalogRepository(crud.NewRepository[entity.Card](ds.Gorm)), images)
}

// ProvideCollectionService builds the collection service over ds's DB, for
// the same reason ProvideCatalogService is its own provider.
func ProvideCollectionService(ds *DataSources) service.CollectionService {
	return service.NewCollectionService(catalogrepository.NewCollectionRepository(crud.NewRepository[entity.Collection](ds.Gorm)))
}

// ProvideInventoryService builds the inventory service over ds's DB.
func ProvideInventoryService(ds *DataSources, images mapper.ImageHost) service.InventoryService {
	return service.NewInventoryService(
		crud.NewTransactor(ds.Gorm),
		catalogrepository.NewCollectionRepository(crud.NewRepository[entity.Collection](ds.Gorm)),
		catalogrepository.NewInventoryRepository(crud.NewRepository[entity.InventoryEntry](ds.Gorm)),
		catalogrepository.NewCatalogRepository(crud.NewRepository[entity.Card](ds.Gorm)),
		crud.NewRepository[entity.Card](ds.Gorm),
		images,
	)
}

// ProvideServices takes just the already-built verifier and services (not the
// DB they're ultimately backed by) so cmd/genspec can keep calling this with
// throwaway, DB-free values of its own construction — see cmd/genspec/main.go
// — without this function needing to know or care where they came from.
func ProvideServices(verifier auth.TokenVerifier, users service.UserService, catalog service.CatalogService, collection service.CollectionService, inventory service.InventoryService) *Services {
	return &Services{
		Health:     coreservice.NewHealthService(),
		Verifier:   verifier,
		Users:      users,
		Catalog:    catalog,
		Collection: collection,
		Inventory:  inventory,
	}
}
