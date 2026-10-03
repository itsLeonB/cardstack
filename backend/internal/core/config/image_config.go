package config

// Image configures image hosting (docs/adr/0016). BaseUrl is the public
// address the API prepends to a hosted-image key.
type Image struct {
	BaseUrl string `split_words:"true"`
}

func (Image) Prefix() string { return "IMAGE" }

// R2 configures the Cloudflare R2 bucket the ingester copies images into.
// Image hosting is optional: with any of the four required fields empty the
// ingester logs a warning and hosts nothing. Endpoint overrides the address
// derived from AccountID, for tests and S3-compatible stand-ins.
type R2 struct {
	AccountID       string `envconfig:"ACCOUNT_ID"`
	AccessKeyID     string `envconfig:"ACCESS_KEY_ID"`
	SecretAccessKey string `envconfig:"SECRET_ACCESS_KEY"`
	Bucket          string
	Endpoint        string
}

func (R2) Prefix() string { return "R2" }

// Configured reports whether every field needed to talk to R2 is set.
func (r R2) Configured() bool {
	return r.AccountID != "" && r.AccessKeyID != "" && r.SecretAccessKey != "" && r.Bucket != ""
}
