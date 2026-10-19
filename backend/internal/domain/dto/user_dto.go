package dto

import "github.com/google/uuid"

// AuthProviderClerk is the AuthProvider of every Auth Identity today.
const AuthProviderClerk = "clerk"

// AuthIdentity is who a verified token says the caller is. Subject is unique
// within Provider; Name is empty when the account has none.
type AuthIdentity struct {
	Provider string
	Subject  string
	Email    string
	Name     string
	// AdminOrigin is set when the token was minted for the Admin app's origin
	// (ADMIN_CLIENT_URLS), not the user app's (docs/adr/0024).
	AdminOrigin bool
}

// CallerIdentity is the user and profile an Auth Identity maps to.
type CallerIdentity struct {
	UserID    uuid.UUID
	ProfileID uuid.UUID
}
