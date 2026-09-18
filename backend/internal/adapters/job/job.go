package job

import (
	"github.com/itsLeonB/cardstack/backend/internal/adapters/job/migrate"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
)

type Job struct {
	cleanup func()
	migrate *migrate.Migrate
}

func Setup(_ *config.Config) (*Job, error) {
	providers, cleanup, err := provider.InitializeProviders()
	if err != nil {
		return nil, err
	}

	migrator, err := migrate.Setup(providers)
	if err != nil {
		cleanup()
		return nil, err
	}

	return &Job{cleanup, migrator}, nil
}

func (j *Job) Run() {
	logger.Info("running all jobs...")

	defer j.cleanup()

	logger.Info("running migrations...")
	if err := j.migrate.Run(); err != nil {
		logger.Fatal(err)
	}
	logger.Info("success running migrations")

	logger.Info("success running all jobs")
}
