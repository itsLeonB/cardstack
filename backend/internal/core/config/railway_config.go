package config

// Railway holds what Railway sets in every environment it runs.
type Railway struct {
	// EnvironmentName is unset outside Railway.
	EnvironmentName string `split_words:"true"`
}

func (Railway) Prefix() string { return "RAILWAY" }
