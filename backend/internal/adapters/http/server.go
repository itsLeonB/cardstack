package http

import (
	"fmt"

	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/routes"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	"github.com/itsLeonB/ezutil/v2/zerolog"
	"github.com/kroma-labs/sentinel-go/httpserver"
)

func Setup(configs config.Config) (*httpserver.Server, func(), error) {
	providers, cleanup, err := provider.InitializeProviders()
	if err != nil {
		return nil, nil, err
	}

	gin.SetMode(configs.Env)
	r := gin.New()
	r.HandleMethodNotAllowed = true

	zerologger := zerolog.Instance(logger.Global)

	skipPaths := []string{"/health", "/metrics", "/favicon.ico"}
	if err = setupSentinel(r, skipPaths, zerologger); err != nil {
		cleanup()
		return nil, nil, err
	}

	api := humagin.New(r, httpapi.NewConfig())
	routes.RegisterRoutes(api, providers.Services)

	httpCfg := httpserver.ProductionConfig()
	httpCfg.LoggerConfig = &httpserver.LoggerConfig{
		Logger:    zerologger,
		SkipPaths: skipPaths,
	}
	httpCfg.Addr = fmt.Sprintf(":%s", configs.App.Port)

	srv := httpserver.New(
		httpserver.WithConfig(httpCfg),
		httpserver.WithServiceName(configs.ServiceName),
		httpserver.WithHandler(r),
		httpserver.WithLogger(zerologger),
	)

	return srv, cleanup, nil
}
