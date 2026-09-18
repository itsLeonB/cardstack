package config

type OTel struct {
	Enabled     bool   `required:"true" default:"false"`
	ServiceName string `split_words:"true" required:"true" default:"cardstack"`
}

func (o OTel) Prefix() string { return "OTEL" }
