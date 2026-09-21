package config

import (
	"errors"
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	App
	DB
	OTel
	Auth
}

var Global *Config

func Load() error {
	var errs error

	var app App
	if err := envconfig.Process(app.Prefix(), &app); err != nil {
		errs = errors.Join(errs, err)
	}

	var db DB
	if err := envconfig.Process(db.Prefix(), &db); err != nil {
		errs = errors.Join(errs, err)
	}

	var otel OTel
	if err := envconfig.Process(otel.Prefix(), &otel); err != nil {
		errs = errors.Join(errs, err)
	}

	var auth Auth
	if err := envconfig.Process(auth.Prefix(), &auth); err != nil {
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		return fmt.Errorf("error loading config: %w", errs)
	}

	Global = &Config{app, db, otel, auth}

	return nil
}
