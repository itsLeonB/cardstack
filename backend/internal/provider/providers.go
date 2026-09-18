package provider

//go:generate go tool wire

type Providers struct {
	*DataSources
	*Services
}
