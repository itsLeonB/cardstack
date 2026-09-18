package config

import "time"

const AppName = "Cardstack"

type App struct {
	Env        string        `default:"debug"`
	Port       string        `default:"8080"`
	Timeout    time.Duration `default:"10s"`
	ClientUrls []string      `split_words:"true"`
}

func (App) Prefix() string { return "APP" }
