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
	Clerk
	Image
	R2
	Gemini
	Embedding
	Match
	RateLimit
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

	var clerk Clerk
	if err := envconfig.Process(clerk.Prefix(), &clerk); err != nil {
		errs = errors.Join(errs, err)
	}

	var image Image
	if err := envconfig.Process(image.Prefix(), &image); err != nil {
		errs = errors.Join(errs, err)
	}

	var r2 R2
	if err := envconfig.Process(r2.Prefix(), &r2); err != nil {
		errs = errors.Join(errs, err)
	}

	var gemini Gemini
	if err := envconfig.Process(gemini.Prefix(), &gemini); err != nil {
		errs = errors.Join(errs, err)
	}

	var embedding Embedding
	if err := envconfig.Process(embedding.Prefix(), &embedding); err != nil {
		errs = errors.Join(errs, err)
	}

	var match Match
	if err := envconfig.Process(match.Prefix(), &match); err != nil {
		errs = errors.Join(errs, err)
	}

	rateLimit := DefaultRateLimit()
	if err := envconfig.Process(rateLimit.Prefix(), &rateLimit); err != nil {
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		return fmt.Errorf("error loading config: %w", errs)
	}

	Global = &Config{app, db, otel, clerk, image, r2, gemini, embedding, match, rateLimit}

	return nil
}
