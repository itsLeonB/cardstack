package config

import "errors"

// Clerk configures verification of Clerk session tokens. Both are optional at
// load time because the job and the ingesters load the same config without
// verifying tokens; the API checks them with ValidateClerk when it boots.
type Clerk struct {
	// SecretKey authenticates the fetch of the instance's signing keys.
	SecretKey string `split_words:"true"`
	// Issuer is the instance's Frontend API URL, the iss claim of its tokens
	// (for example https://example.clerk.accounts.dev).
	Issuer string
}

func (Clerk) Prefix() string { return "CLERK" }

// ValidateClerk refuses a config that would make every token fail, which would
// otherwise show up only as a 401 for every user.
func (c Clerk) ValidateClerk() error {
	if c.SecretKey == "" || c.Issuer == "" {
		return errors.New("CLERK_SECRET_KEY and CLERK_ISSUER are required")
	}
	return nil
}
