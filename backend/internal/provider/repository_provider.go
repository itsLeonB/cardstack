package provider

import (
	"github.com/google/wire"
	coreservice "github.com/itsLeonB/cardstack/backend/internal/adapters/core/service"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	crud "github.com/itsLeonB/go-crud"
)

// RepositorySet is the wire provider set for the repositories a Service shares
// with the HTTP layer's wiring.
var RepositorySet = wire.NewSet(ProvideUserRepository, ProvideIdentityCache)

func ProvideUserRepository(ds *DataSources) repository.UserRepository {
	return repository.NewUserRepository(crud.NewRepository[entity.User](ds.Gorm))
}

// ProvideIdentityCache is one instance for the process: the cache is the shared
// state between requests.
func ProvideIdentityCache() service.IdentityCache {
	return coreservice.NewIdentityCache()
}
