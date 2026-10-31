package config

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	App
	DB
	OTel
	Clerk
	Image
	R2
	ScanPhotoR2
	Admin
	Gemini
	Embedding
	Match
	RateLimit
}

// ValidateAdminOrigins refuses a boot where a token's origin cannot say which
// app it came from: an origin in both lists, or Admins with no Admin app origin
// to accept them from (docs/adr/0024). With neither set it is fine.
func (c Config) ValidateAdminOrigins() error {
	if len(c.Emails) > 0 && len(c.AdminClientUrls) == 0 {
		return errors.New("ADMIN_EMAILS is set but ADMIN_CLIENT_URLS is empty: the admin routes would answer no one, so list the Admin app's origin")
	}
	for _, origin := range c.AdminClientUrls {
		if slices.Contains(c.ClientUrls, origin) {
			return fmt.Errorf("%q is in both APP_CLIENT_URLS and ADMIN_CLIENT_URLS: a token's origin must belong to one app", origin)
		}
	}
	return nil
}

// previewAppEnv is the APP_ENV the preview workflow sets in every PR
// environment it provisions.
const previewAppEnv = "preview"

// validateProvisioned refuses a Railway environment other than production whose
// APP_ENV is not "preview": such an environment was never provisioned, so it
// still carries production's variables, including its database. Outside Railway
// the environment name is unset and the check passes. Every binary loads
// configuration through Load, so the API and the pre-deploy migration are both
// covered.
func validateProvisioned(railwayEnv, appEnv string) error {
	if railwayEnv == "" || railwayEnv == "production" || appEnv == previewAppEnv {
		return nil
	}
	return fmt.Errorf("RAILWAY_ENVIRONMENT_NAME is %q and APP_ENV is %q: this Railway environment was never provisioned as a preview (it would use production's database), so refusing to boot; the preview workflow sets APP_ENV=preview, and a push reruns it", railwayEnv, appEnv)
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

	var scanPhoto ScanPhotoR2
	if err := envconfig.Process(scanPhoto.Prefix(), &scanPhoto); err != nil {
		errs = errors.Join(errs, err)
	}

	var admin Admin
	if err := envconfig.Process(admin.Prefix(), &admin); err != nil {
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

	if err := validateProvisioned(os.Getenv("RAILWAY_ENVIRONMENT_NAME"), app.Env); err != nil {
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		return fmt.Errorf("error loading config: %w", errs)
	}

	Global = &Config{app, db, otel, clerk, image, r2, scanPhoto, admin, gemini, embedding, match, rateLimit}

	return nil
}
