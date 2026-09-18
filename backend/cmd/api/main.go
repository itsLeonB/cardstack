package main

import (
	"context"
	"os"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/http"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/core/otel"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	var exitCode int
	defer func() {
		os.Exit(exitCode)
	}()

	logger.Init(config.AppName)

	if err := config.Load(); err != nil {
		logger.Error(err)
		exitCode = 1
		return
	}

	ctx := context.Background()
	otelShutdown, err := otel.InitSDK(ctx, config.Global.OTel)
	if err != nil {
		logger.Error(err)
		exitCode = 1
		return
	}
	defer func() {
		if err := otelShutdown(ctx); err != nil {
			logger.Error(err)
		}
	}()

	srv, shutdownFunc, err := http.Setup(*config.Global)
	if err != nil {
		logger.Error(err)
		exitCode = 1
		return
	}
	defer shutdownFunc()

	if err := srv.ListenAndServe(ctx); err != nil {
		logger.Error(err)
		exitCode = 1
	}
}
