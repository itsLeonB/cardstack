# 07: Clerk backend

**Parent:** `.scratch/security-hardening/spec.md` (see ADR-0015, which supersedes ADR-0003 and ADR-0004)

**What to build:** The API authenticates with a bearer token verified against Clerk, owns only a `users` row per Auth Identity, creates the user and profile on first use, and no longer contains any credential, session or cookie code. A caller with a valid token reaches their own Collections; a caller with no header is a Guest; a caller with a bad or expired token gets 401. This is the largest ticket: the schema change and the removal of go-authkit cannot be split without an expand/contract migration, so they land together.

**Blocked by:** None (can start immediately). Manual verification against a real Clerk instance needs ticket 01; the tests do not.

**Status:** ready-for-agent

- [ ] A token-verifier interface with a Clerk adapter (checks signature via cached keys, expiry, issuer and the authorized party against the configured frontend origins; confirm the exact mechanism against Clerk's current docs) and a fake for tests. Backend tests need no network access to Clerk.
- [ ] Request classification: no `Authorization` header is a Guest; a valid token is authenticated with the user and profile in context; a present-but-invalid or expired token is 401, never a Guest. Routes declare whether they allow guests; private routes (collections, inventory) reject Guests with 401.
- [ ] The `users` table keeps its id, gains an auth provider and an auth subject (unique together), keeps email as required, non-unique and indexed, and loses the password hash and verified flag. The sessions and refresh-token tables are dropped. The migration first deletes all existing users, cascading to profiles, collections and Inventory Entries (intentional; the migration comment says so).
- [ ] On the first authenticated request a user and profile are created in one transaction, safe under concurrent first requests. Profile name comes from the name claim, falling back to the email's local part, and is never overwritten. Email is updated only when the claim differs. A short in-memory cache maps the Auth Identity to the profile.
- [ ] Removed: the register, login, refresh, logout and current-user endpoints, the auth cookies and fingerprint, the CSRF guard from the earlier ticket 16, the go-authkit dependency and configuration, and all tests of removed code. Ownership isolation is unchanged: foreign and missing Collections still look identical.
- [ ] Route tests cover: Guest, valid, invalid and expired tokens on private and guest-allowed routes, wrong authorized party, lazy creation including concurrent first requests, email refresh, name immutability, ownership isolation. The user repository tests are rewritten for the new shape.
- [ ] The OpenAPI file is regenerated, new settings are documented (deployment doc, env example), the conventions docs are updated where they describe auth, and backend build, vet and tests pass.

## Comments
