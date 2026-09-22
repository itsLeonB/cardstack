# 02: Auth: registration & login

**What to build:** A user can register an account with an email and password and log in, so that they have a private, persistent session to work in. No email verification or password reset in MVP (ADR-0004) — just registration and login, backed by `go-authkit` in stateful mode wired through custom Huma handlers (ADR-0003).

**Blocked by:** 01 (Project & CI scaffolding) — done.

**Status:** done

**Plan:** `.scratch/cardstack-mvp/plans/02-auth-registration-login.md`

- [x] User can register with an email and password
- [x] User can log in with email and password
- [x] Session persists across requests (stateful mode: session + refresh-token rotation)
- [x] User can log out
- [x] An authenticated-route guard on the backend rejects unauthenticated requests to protected endpoints
- [x] Frontend redirects unauthenticated users away from protected routes
- [x] No email verification or password-reset flow is present — this is deliberate, not a gap
- [x] Auth endpoints appear in the generated OpenAPI spec and the frontend's generated client

## Comments

Implemented via `backend-agent`/`frontend-agent` worktrees per the plan, merged into `claude/focused-tesla-8qtnaj` (PR itsLeonB/cardstack#5). Backend build/vet/test all green against a real Postgres 18 instance; frontend lint/typecheck/test/build all green with the orval client regenerating byte-identical to the committed one.

Drove the real register → login → `/auth/me` → CSRF-guarded logout/refresh flow end-to-end over HTTP before considering this done (not just each side's own mocked/handler-level tests), and caught two real issues that inline review missed:

- Backend: the fingerprint cookie was unconditionally named `__Secure-Fgp` regardless of `AUTH_COOKIE_SECURE`. Per RFC 6265bis a `__Secure-`-prefixed cookie is only valid with the `Secure` attribute set — browsers/modern curl silently refuse to store it otherwise, so `AUTH_COOKIE_SECURE=false` (a legitimate non-TLS local/preview config) broke every authenticated request. Fixed by deriving the cookie name from `CookieSecure`; added `transport_test.go` covering it.
- Frontend: an automated security review caught an open redirect in `login.tsx` (`redirect` search param used unvalidated in `navigate`). Restricted to a same-origin relative path.

Both fixes are on top of the merge, verified green, and pushed.
