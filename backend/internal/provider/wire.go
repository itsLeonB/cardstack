//go:build wireinject

package provider

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	DataSourceSet,
	RepositorySet,
	AuthSet,
	ServiceSet,
	wire.Struct(new(Providers), "*"),
)

func InitializeProviders() (*Providers, func(), error) {
	wire.Build(ProviderSet)
	return nil, nil, nil
}
