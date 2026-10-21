package http

import (
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/ratelimit"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/routes"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	"github.com/itsLeonB/ezutil/v2/zerolog"
	"github.com/kroma-labs/sentinel-go/httpserver"
)

func Setup(configs config.Config) (*httpserver.Server, func(), error) {
	if err := configs.ValidateClerk(); err != nil {
		return nil, nil, err
	}
	if err := configs.ValidateRateLimit(); err != nil {
		return nil, nil, err
	}
	if err := configs.ValidateMatch(); err != nil {
		return nil, nil, err
	}
	if err := configs.ValidateClientUrls(); err != nil {
		return nil, nil, err
	}
	if err := configs.ValidateAdminEmails(); err != nil {
		return nil, nil, err
	}
	if err := configs.ValidateAdminClientUrls(); err != nil {
		return nil, nil, err
	}
	if err := configs.ValidateAdminOrigins(); err != nil {
		return nil, nil, err
	}
	if err := configs.ValidateScanPhotoR2(); err != nil {
		return nil, nil, err
	}
	if noClientOrigins(configs) {
		// A preview has none until its frontend is deployed.
		logger.Error("APP_CLIENT_URLS and ADMIN_CLIENT_URLS are both empty: every token will be rejected, since none can come from a configured frontend origin")
	}

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

	api := humagin.New(r, httpapi.NewServerConfig(configs.Env))
	httpapi.UseRecovery(api)
	if err := routes.RegisterRoutes(api, providers.Services, ratelimit.NewLimits(configs.RateLimit, time.Now), adminEmails(configs)); err != nil {
		cleanup()
		return nil, nil, err
	}

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

// adminEmails returns the Admins when the admin routes may exist: the
// allowlist and the R2 write settings must both be set (docs/adr/0019), else
// none are returned and RegisterRoutes mounts no admin route.
func adminEmails(configs config.Config) []string {
	if !configs.Configured() {
		if len(configs.Emails) > 0 {
			logger.Error("ADMIN_EMAILS is set but the R2 settings are not: the admin routes are off")
		}
		return nil
	}
	return configs.Emails
}

func noClientOrigins(configs config.Config) bool {
	return len(configs.ClientUrls) == 0 && len(configs.AdminClientUrls) == 0
}
