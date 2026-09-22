// Command genspec boots just enough of the Huma API (no port bound, no DB
// connection) to dump its OpenAPI spec to backend/openapi.json. Frontend's
// orval config reads that committed file, so this is what bridges the two
// components' CI without either depending on the other at build/test time.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/routes"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	authkit "github.com/itsLeonB/go-authkit"
)

const outputPath = "openapi.json"

func main() {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := humagin.New(router, httpapi.NewConfig())

	// auth_handler.go reads config.Global.Auth (matching setup_sentinel.go's
	// existing convention), but genspec never calls config.Load() — it's
	// meant to boot without any real infra or env, DB included. A minimal
	// stand-in config.Global plus a throwaway *authkit.AuthKit (valid JWT
	// config, no real stores) is enough for route *registration*: handler
	// closures reference the kit but never invoke it until a real request
	// comes in, and none does here.
	config.Global = &config.Config{}
	kit, err := authkit.New(
		authkit.Config{JWTSecret: "genspec", JWTIssuer: "genspec", JWTDuration: time.Minute},
		authkit.Deps{},
		authkit.Hooks{},
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error building genspec authkit:", err)
		os.Exit(1)
	}

	// nil is safe here: genspec only registers routes to dump the OpenAPI
	// spec, it never serves a request, so SessionGuard's ProfileLookup is
	// stored in the route closures but never actually called.
	routes.RegisterRoutes(api, provider.ProvideServices(kit, nil))

	spec, err := api.OpenAPI().MarshalJSON()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error marshaling OpenAPI spec:", err)
		os.Exit(1)
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, spec, "", "  "); err != nil {
		fmt.Fprintln(os.Stderr, "error formatting OpenAPI spec:", err)
		os.Exit(1)
	}
	pretty.WriteByte('\n')

	if err := os.WriteFile(outputPath, pretty.Bytes(), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error writing openapi.json:", err)
		os.Exit(1)
	}

	fmt.Printf("wrote %s\n", outputPath)
}
