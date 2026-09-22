// Package endpoint provides small, concrete generic wrappers around
// huma.Register, one per route shape, instead of one mega-generic
// mechanism:
//
//   - Endpoint[Req, Res]: the common shape — a request type Req, a response
//     type Res wrapped in httpapi.Envelope, and a service call that either
//     succeeds or returns an error untouched.
//   - NoBodyEndpoint[Req]: a route with no response body at all (e.g. 204).
//   - ListEndpoint[Req, Res]: a route returning Envelope[[]Res] plus an
//     X-Total-Count header set to len(result).
//   - RedirectEndpoint[Req]: a bodyless redirect (Status + Location header).
//
// Every route fits one of these four shapes; each can be wrapped as a
// Registrable (see registrable.go) and mixed in one []Registrable slice for
// RegisterAll.
package endpoint

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
)

// CookieAuthSecurity is the single security requirement used by every
// secured route in this API — see httpapi.NewConfig, which registers
// "CookieAuth" as the only security scheme. Exported so handlers that
// register routes directly with huma.Register (bypassing this package's
// Endpoint wrappers, e.g. auth_handler.go's cookie-setting routes) can
// declare the same security requirement without duplicating the scheme
// name.
var CookieAuthSecurity = []map[string][]string{{"CookieAuth": {}}}

// mergeMiddlewares appends route-specific middlewares after RegisterAll's
// shared ones, so shared middleware always runs first. Returns shared
// unmodified when route is empty, keeping behavior identical to passing mw
// straight through for every route that doesn't set per-route Middlewares.
func mergeMiddlewares(shared, route []func(huma.Context, func(huma.Context))) []func(huma.Context, func(huma.Context)) {
	if len(route) == 0 {
		return shared
	}

	merged := make([]func(huma.Context, func(huma.Context)), 0, len(shared)+len(route))
	merged = append(merged, shared...)
	merged = append(merged, route...)

	return merged
}

// Endpoint describes one route.
type Endpoint[Req, Res any] struct {
	OperationID string
	Method      string
	Path        string
	Summary     string
	Tags        []string
	SuccessCode int
	Secured     bool
	Middlewares []func(huma.Context, func(huma.Context))
	HandlerFunc func(context.Context, Req) (Res, error)
}

// envelopeOutput is the Output struct every Endpoint registers.
type envelopeOutput[Res any] struct {
	Body httpapi.Envelope[Res]
}

// Register builds a huma.Operation from e and registers it on api.
func Register[Req, Res any](api huma.API, e Endpoint[Req, Res], mw ...func(huma.Context, func(huma.Context))) {
	op := huma.Operation{
		OperationID:   e.OperationID,
		Method:        e.Method,
		Path:          e.Path,
		Summary:       e.Summary,
		Tags:          e.Tags,
		DefaultStatus: e.SuccessCode,
		Middlewares:   mergeMiddlewares(mw, e.Middlewares),
	}

	if e.Secured {
		op.Security = CookieAuthSecurity
	}

	huma.Register(api, op, func(ctx context.Context, in *Req) (*envelopeOutput[Res], error) {
		res, err := e.HandlerFunc(ctx, *in)
		if err != nil {
			return nil, err
		}

		return &envelopeOutput[Res]{Body: httpapi.NewEnvelope(res)}, nil
	})
}

// NoBodyEndpoint describes one route with no response body at all (e.g. a
// 204 No Content).
type NoBodyEndpoint[Req any] struct {
	OperationID string
	Method      string
	Path        string
	Summary     string
	Tags        []string
	Secured     bool
	Middlewares []func(huma.Context, func(huma.Context))
	HandlerFunc func(context.Context, Req) error
}

// noBodyOutput is the Output struct every NoBodyEndpoint registers.
type noBodyOutput struct{}

// RegisterNoBody builds a huma.Operation from e and registers it on api.
// DefaultStatus is always http.StatusNoContent.
func RegisterNoBody[Req any](api huma.API, e NoBodyEndpoint[Req], mw ...func(huma.Context, func(huma.Context))) {
	op := huma.Operation{
		OperationID:   e.OperationID,
		Method:        e.Method,
		Path:          e.Path,
		Summary:       e.Summary,
		Tags:          e.Tags,
		DefaultStatus: http.StatusNoContent,
		Middlewares:   mergeMiddlewares(mw, e.Middlewares),
	}

	if e.Secured {
		op.Security = CookieAuthSecurity
	}

	huma.Register(api, op, func(ctx context.Context, in *Req) (*noBodyOutput, error) {
		if err := e.HandlerFunc(ctx, *in); err != nil {
			return nil, err
		}

		return &noBodyOutput{}, nil
	})
}

// ListEndpoint describes one "get list" route: a body of Envelope[[]Res]
// plus an X-Total-Count header set to len(result).
type ListEndpoint[Req, Res any] struct {
	OperationID string
	Method      string
	Path        string
	Summary     string
	Tags        []string
	Secured     bool
	Middlewares []func(huma.Context, func(huma.Context))
	HandlerFunc func(context.Context, Req) ([]Res, error)
}

// listOutput is the Output struct every ListEndpoint registers.
type listOutput[Res any] struct {
	XTotalCount int `header:"X-Total-Count"`
	Body        httpapi.Envelope[[]Res]
}

// RegisterList builds a huma.Operation from e and registers it on api.
// DefaultStatus is always http.StatusOK.
func RegisterList[Req, Res any](api huma.API, e ListEndpoint[Req, Res], mw ...func(huma.Context, func(huma.Context))) {
	op := huma.Operation{
		OperationID:   e.OperationID,
		Method:        e.Method,
		Path:          e.Path,
		Summary:       e.Summary,
		Tags:          e.Tags,
		DefaultStatus: http.StatusOK,
		Middlewares:   mergeMiddlewares(mw, e.Middlewares),
	}

	if e.Secured {
		op.Security = CookieAuthSecurity
	}

	huma.Register(api, op, func(ctx context.Context, in *Req) (*listOutput[Res], error) {
		res, err := e.HandlerFunc(ctx, *in)
		if err != nil {
			return nil, err
		}

		return &listOutput[Res]{XTotalCount: len(res), Body: httpapi.NewEnvelope(res)}, nil
	})
}

// RedirectEndpoint describes one redirect route with no body.
type RedirectEndpoint[Req any] struct {
	OperationID string
	Method      string
	Path        string
	Summary     string
	Tags        []string
	Secured     bool
	Middlewares []func(huma.Context, func(huma.Context))
	HandlerFunc func(context.Context, Req) (string, error)
}

// redirectOutput is the Output struct every RedirectEndpoint registers.
type redirectOutput struct {
	Status   int
	Location string `header:"Location"`
}

// RegisterRedirect builds a huma.Operation from e and registers it on api.
// Status is always http.StatusTemporaryRedirect.
func RegisterRedirect[Req any](api huma.API, e RedirectEndpoint[Req], mw ...func(huma.Context, func(huma.Context))) {
	op := huma.Operation{
		OperationID:   e.OperationID,
		Method:        e.Method,
		Path:          e.Path,
		Summary:       e.Summary,
		Tags:          e.Tags,
		DefaultStatus: http.StatusTemporaryRedirect,
		Middlewares:   mergeMiddlewares(mw, e.Middlewares),
	}

	if e.Secured {
		op.Security = CookieAuthSecurity
	}

	huma.Register(api, op, func(ctx context.Context, in *Req) (*redirectOutput, error) {
		url, err := e.HandlerFunc(ctx, *in)
		if err != nil {
			return nil, err
		}

		return &redirectOutput{Status: http.StatusTemporaryRedirect, Location: url}, nil
	})
}
