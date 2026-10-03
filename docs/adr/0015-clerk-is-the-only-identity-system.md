# Clerk is the only identity system; the API verifies bearer tokens and owns only a user row

Supersedes ADR-0003 (custom Huma handlers over go-authkit) and ADR-0004 (no email verification or reset).

The MVP's home-grown sign-in (go-authkit in stateful mode, HttpOnly cookies, a CSRF guard) was built for one user. Opening the app up needs email verification, password reset, social login, lockout, bot protection and enumeration protection, and building and maintaining those ourselves is the opposite of hardening. We chose Clerk and removed go-authkit entirely rather than running both, because two identity systems double the attack surface. The backend no longer handles credentials, sessions or auth cookies: it verifies a short-lived signed token from an `Authorization: Bearer` header behind a small interface (a Clerk adapter in production, a fake in tests, per ADR-0011) and maps the caller to a `users` row it owns, holding an auth provider, an auth subject (unique together) and an email. `user_profiles` and everything below it are unchanged. A request with no header is a Guest; a header that is present but invalid is a 401, never a silent downgrade to Guest. Because there are no auth cookies the CSRF guard is deleted. Existing users are wiped by the migration (no live users yet).

## Considered Options

- Keep go-authkit and add Clerk alongside it. Rejected: two systems, two sets of bugs, and the old one is the weaker.
- Clerk for sign-in only, with the backend exchanging the Clerk token for its own HttpOnly session cookie. Rejected: it keeps the sessions table, the CSRF guard and the cross-site cookie setup, and makes Clerk's revocation no longer the source of truth.

## Consequences

Clerk's session token is readable by page JavaScript (it lives 60 seconds), whereas the old access cookie was HttpOnly. The long-lived credential stays HttpOnly on Clerk's own domain. In return the app gains lockout, bot and enumeration protection and breach-password checks it did not have. Production needs a domain the owner controls. Email and name reach the backend as signed custom claims on the session token. Syncing account deletion from Clerk is deferred because it needs a webhook endpoint.
