package provider

import (
	objectstoreadapter "github.com/itsLeonB/cardstack/backend/internal/adapters/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	crud "github.com/itsLeonB/go-crud"
)

// ProvideObjectStore builds the R2 store the admin signs uploads for. Without
// the R2 settings it is nil: the API then registers no admin route, so nothing
// calls it.
func ProvideObjectStore() objectstore.ObjectStore {
	if !config.Global.Configured() {
		return nil
	}
	return objectstoreadapter.NewR2Store(config.Global.R2)
}

// ProvideCardAdminService builds the Admin's card service over ds's DB.
func ProvideCardAdminService(ds *DataSources, images mapper.ImageHost, store objectstore.ObjectStore) service.CardAdminService {
	return service.NewCardAdminService(
		crud.NewTransactor(ds.Gorm),
		crud.NewRepository[entity.Card](ds.Gorm),
		crud.NewRepository[entity.ExpansionSet](ds.Gorm),
		crud.NewRepository[entity.Rarity](ds.Gorm),
		store,
		images,
	)
}
