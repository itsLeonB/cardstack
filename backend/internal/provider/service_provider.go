package provider

import (
	"github.com/google/wire"
	coreservice "github.com/itsLeonB/cardstack/backend/internal/adapters/core/service"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
)

// ServiceSet is the wire provider set for the top-level Services.
var ServiceSet = wire.NewSet(ProvideServices)

type Services struct {
	Health service.HealthService
}

func ProvideServices() *Services {
	return &Services{
		Health: coreservice.NewHealthService(),
	}
}
