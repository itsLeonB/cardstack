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

	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/routes"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
)

const outputPath = "openapi.json"

func main() {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := humagin.New(router, httpapi.NewConfig())

	routes.RegisterRoutes(api, provider.ProvideServices())

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
