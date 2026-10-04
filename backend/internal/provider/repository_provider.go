package provider

import (
	"github.com/google/wire"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	crud "github.com/itsLeonB/go-crud"
)

// RepositorySet is the wire provider set for the repositories a Service shares
// with the HTTP layer's wiring.
var RepositorySet = wire.NewSet(ProvideUserRepository)

func ProvideUserRepository(ds *DataSources) repository.UserRepository {
	return repository.NewUserRepository(crud.NewRepository[entity.User](ds.Gorm))
}
