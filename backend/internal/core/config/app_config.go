package config

import "time"

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
