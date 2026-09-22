package provider

import (
	"github.com/google/wire"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/core/service"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/repository"
	authkit "github.com/itsLeonB/go-authkit"
	crud "github.com/itsLeonB/go-crud"
)

// RepositorySet is the wire provider set for the repository adapters
// authkit.Deps needs (UserStore/SessionStore/RefreshTokenStore, a
// crud.Transactor satisfying authkit.Transactor, and the in-process
// SessionCache) plus the crud.Transactor those repositories' transactions
// run through.
var RepositorySet = wire.NewSet(
	ProvideTransactor,
	ProvideUserStore,
	ProvideSessionStore,
	ProvideRefreshTokenStore,
	ProvideSessionCache,
)

func ProvideTransactor(ds *DataSources) authkit.Transactor {
	return crud.NewTransactor(ds.Gorm)
}

func ProvideUserStore(ds *DataSources) authkit.UserStore {
	return repository.NewUserRepository(ds.Gorm)
}

func ProvideSessionStore(ds *DataSources) authkit.SessionStore {
	return repository.NewSessionRepository(ds.Gorm)
}

func ProvideRefreshTokenStore(ds *DataSources) authkit.RefreshTokenStore {
	return repository.NewRefreshTokenRepository(ds.Gorm)
}

func ProvideSessionCache() authkit.SessionCache {
	return service.NewSessionCache()
}
