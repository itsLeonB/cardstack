package config

import (
	"errors"
	"fmt"
	"strings"

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
	if err := validateAuthCookiePolicy(auth); err != nil {
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		return fmt.Errorf("error loading config: %w", errs)
	}

	Global = &Config{app, db, otel, auth}

	return nil
}

// validateAuthCookiePolicy rejects SameSite=None without Secure: browsers
// reject that combination outright, so every auth cookie
// (access/refresh/fingerprint/csrf) would silently never get stored —
// identical in effect to the __Secure-Fgp naming bug fixed earlier. Caught
// at boot instead of at the first login attempt.
func validateAuthCookiePolicy(auth Auth) error {
	if strings.EqualFold(auth.CookieSamesite, "none") && !auth.CookieSecure {
		return errors.New("AUTH_COOKIE_SAMESITE=None requires AUTH_COOKIE_SECURE=true")
	}
	return nil
}
