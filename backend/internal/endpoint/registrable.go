package endpoint

import "github.com/danielgtaylor/huma/v2"

// Registrable lets a handler collect routes of different Req/Res types into
// one homogeneous slice for RegisterAll. Go generics can't put
// Endpoint[Req,Res] directly into such a slice, so New wraps one in a
// non-generic adapter closing over its own type parameters. NewNoBody,
// NewList, and NewRedirect do the same for the other route shapes in this
// package, so all four can be mixed in one []Registrable.
type Registrable interface {
	Register(api huma.API, mw ...func(huma.Context, func(huma.Context)))
}

type registrable[Req, Res any] struct {
	endpoint Endpoint[Req, Res]
}

// New wraps an Endpoint so it can be collected as a Registrable.
func New[Req, Res any](e Endpoint[Req, Res]) Registrable {
	return registrable[Req, Res]{e}
}

func (r registrable[Req, Res]) Register(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
	Register(api, r.endpoint, mw...)
}

type noBodyRegistrable[Req any] struct {
	endpoint NoBodyEndpoint[Req]
}

// NewNoBody wraps a NoBodyEndpoint so it can be collected as a Registrable.
func NewNoBody[Req any](e NoBodyEndpoint[Req]) Registrable {
	return noBodyRegistrable[Req]{e}
}

func (r noBodyRegistrable[Req]) Register(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
	RegisterNoBody(api, r.endpoint, mw...)
}

type listRegistrable[Req, Res any] struct {
	endpoint ListEndpoint[Req, Res]
}

// NewList wraps a ListEndpoint so it can be collected as a Registrable.
func NewList[Req, Res any](e ListEndpoint[Req, Res]) Registrable {
	return listRegistrable[Req, Res]{e}
}

func (r listRegistrable[Req, Res]) Register(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
	RegisterList(api, r.endpoint, mw...)
}

type redirectRegistrable[Req any] struct {
	endpoint RedirectEndpoint[Req]
}

// NewRedirect wraps a RedirectEndpoint so it can be collected as a
// Registrable.
func NewRedirect[Req any](e RedirectEndpoint[Req]) Registrable {
	return redirectRegistrable[Req]{e}
}

func (r redirectRegistrable[Req]) Register(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
	RegisterRedirect(api, r.endpoint, mw...)
}

// RegisterAll registers every route in routes on api.
func RegisterAll(api huma.API, routes []Registrable, mw ...func(huma.Context, func(huma.Context))) {
	for _, r := range routes {
		r.Register(api, mw...)
	}
}
