package provider

import (
	"github.com/google/wire"
	coreservice "github.com/itsLeonB/cardstack/backend/internal/adapters/core/service"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	authkit "github.com/itsLeonB/go-authkit"
)

// ServiceSet is the wire provider set for the top-level Services.
var ServiceSet = wire.NewSet(ProvideServices)

type Services struct {
	Health service.HealthService
	Auth   *authkit.AuthKit
}

// ProvideServices takes just the already-built *authkit.AuthKit (not the DB
// it's ultimately backed by) so cmd/genspec can keep calling this with a
// throwaway, DB-free kit of its own construction — see cmd/genspec/main.go
// — without this function needing to know or care where kit came from.
func ProvideServices(kit *authkit.AuthKit) *Services {
	return &Services{
		Health: coreservice.NewHealthService(),
		Auth:   kit,
	}
}
