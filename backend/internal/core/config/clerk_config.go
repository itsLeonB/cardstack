package config

// Clerk configures verification of Clerk session tokens. Both are optional here
// because the job and the ingester load the same config without verifying
// tokens; provider.ProvideTokenVerifier insists on them when the API boots.
type Clerk struct {
	// SecretKey authenticates the fetch of the instance's signing keys.
	SecretKey string `split_words:"true"`
	// Issuer is the instance's Frontend API URL, the iss claim of its tokens
	// (for example https://example.clerk.accounts.dev).
	Issuer string
}

func (Clerk) Prefix() string { return "CLERK" }
