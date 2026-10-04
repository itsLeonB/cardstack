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
}

// CallerIdentity is the user and profile an Auth Identity maps to.
type CallerIdentity struct {
	UserID    uuid.UUID
	ProfileID uuid.UUID
}
