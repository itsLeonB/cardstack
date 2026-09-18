// Package httpapi hosts the shared Huma v2 wiring for the cardstack API:
// the huma.Config and the Envelope response wrapper.
package httpapi

import "github.com/danielgtaylor/huma/v2"

// NewConfig builds the huma.Config used to construct the huma.API bound to
// the gin engine root.
//
// Successful response bodies are wrapped in a top-level "data" field (see
// httpapi.Envelope) by each handler's Output struct, matching the envelope
// shape the frontend expects. That wrapping is done per-Output-struct, not
// here; this config only disables Huma's own schema-link additions, which
// are unrelated to the envelope:
//   - CreateHooks is cleared so the default SchemaLinkTransformer (which
//     injects a top-level "$schema" field and a describedby Link header into
//     every response) is never registered.
//   - SchemasPath is cleared so the "/schemas/{schema}" route that serves
//     those linked JSON Schema documents is not mounted either.
//
// DocsPath and OpenAPIPath are left at their defaults ("/docs",
// "/openapi.json"/"/openapi.yaml") so Huma auto-mounts docs + spec at the
// engine root, unauthenticated.
//
// BearerAuth is registered as a security scheme now, unused until an
// authenticated route lands, so every future secured endpoint.Endpoint can
// reference it without touching this config again.
func NewConfig() huma.Config {
	cfg := huma.DefaultConfig("Cardstack API", "1.0")

	cfg.CreateHooks = nil
	cfg.SchemasPath = ""

	if cfg.Components.SecuritySchemes == nil {
		cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	cfg.Components.SecuritySchemes["BearerAuth"] = &huma.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
	}

	return cfg
}
