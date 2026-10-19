package config

import (
	"fmt"
	"net/mail"
)

// Admin names the people allowed to edit the catalog (docs/adr/0019). Emails
// is compared case-insensitively against the verified email claim of the
// Clerk token. With none listed, or without the R2 settings, the API
// registers no admin routes at all.
type Admin struct {
	Emails []string
	// AdminClientUrls are the origins of the Admin app (docs/adr/0024). The
	// name is not ClientUrls: Config embeds App, which has ClientUrls too.
	AdminClientUrls []string `envconfig:"CLIENT_URLS"`
}

func (Admin) Prefix() string { return "ADMIN" }

// ValidateAdminClientUrls refuses an entry that is not a bare origin, for the
// same reason as App.ValidateClientUrls: a token's authorized party is matched
// as an exact string, so a malformed entry would reject every Admin token.
func (a Admin) ValidateAdminClientUrls() error {
	return validateBareOrigins("ADMIN_CLIENT_URLS", a.AdminClientUrls)
}

// ValidateAdminEmails refuses an entry that is not a bare email address. A typo
// would otherwise lock the owner out or, worse, list a different person, with
// nothing failing loudly. An empty list is fine: the admin routes are then off.
func (a Admin) ValidateAdminEmails() error {
	for _, entry := range a.Emails {
		// ParseAddress also accepts "Name <a@b.c>", which can never equal a
		// token's email claim, so require the parsed address to be the entry.
		parsed, err := mail.ParseAddress(entry)
		if err != nil || parsed.Address != entry {
			return fmt.Errorf("ADMIN_EMAILS entry %q is not an email address like owner@example.com (no name, spaces or quotes)", entry)
		}
	}
	return nil
}
