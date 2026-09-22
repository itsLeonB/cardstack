package main

import (
	"context"

	"github.com/itsLeonB/cardstack/backend/internal/adapters/job"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/core/otel"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	logger.Init("Job")

	if err := config.Load(); err != nil {
		logger.Fatal(err)
	}

	ctx := context.Background()
	otelShutdown, err := otel.InitSDK(ctx, config.Global.OTel)
	if err != nil {
		logger.Fatalf("failed to initialize OTel SDK: %v", err)
	}
	defer func() {
		if err := otelShutdown(ctx); err != nil {
			logger.Errorf("error shutting down OTel SDK: %v", err)
		}
	}()

	j, err := job.Setup(config.Global)
	if err != nil {
		logger.Fatal(err)
	}

	j.Run()
}
