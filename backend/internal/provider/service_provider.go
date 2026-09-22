package provider

import (
	"github.com/google/wire"
	coreservice "github.com/itsLeonB/cardstack/backend/internal/adapters/core/service"
	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	authkit "github.com/itsLeonB/go-authkit"
)

// ServiceSet is the wire provider set for the top-level Services.
var ServiceSet = wire.NewSet(ProvideServices)

type Services struct {
	Health   service.HealthService
	Auth     *authkit.AuthKit
	Profiles authpkg.ProfileLookup
}

// ProvideServices takes just the already-built *authkit.AuthKit and
// auth.ProfileLookup (not the DB they're ultimately backed by) so
// cmd/genspec can keep calling this with throwaway, DB-free values of its
// own construction — see cmd/genspec/main.go — without this function
// needing to know or care where they came from.
func ProvideServices(kit *authkit.AuthKit, profiles authpkg.ProfileLookup) *Services {
	return &Services{
		Health:   coreservice.NewHealthService(),
		Auth:     kit,
		Profiles: profiles,
	}
}
