package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const AppName = "Cardstack"

type App struct {
	Env        string        `default:"debug"`
	Port       string        `default:"8080"`
	Timeout    time.Duration `default:"10s"`
	ClientUrls []string      `split_words:"true"`
	// EdgeSecret is the value Cloudflare sends in X-Edge-Secret. Leave unset
	// outside production: previews have no Cloudflare in front of them.
	EdgeSecret string `split_words:"true"`
}

func (App) Prefix() string { return "APP" }

// ValidateClientUrls refuses an entry that is not a bare origin. A token's
// authorized party is compared to each entry as an exact string, so a trailing
// slash, a space after a comma or quotes would not fail anything loudly: every
// sign-in would just get a 401. An empty list is fine; a preview has none until
// its frontend is deployed.
func (a App) ValidateClientUrls() error {
	for _, entry := range a.ClientUrls {
		origin, err := url.Parse(entry)
		if err != nil || entry != strings.TrimSpace(entry) ||
			(origin.Scheme != "http" && origin.Scheme != "https") ||
			origin.Host == "" || origin.User != nil ||
			origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
			return fmt.Errorf("APP_CLIENT_URLS entry %q is not a bare origin like https://www.example.com (no path, trailing slash, spaces or quotes)", entry)
		}
	}
	return nil
}
