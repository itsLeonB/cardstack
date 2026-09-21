# 02: Auth: registration & login

**What to build:** A user can register an account with an email and password and log in, so that they have a private, persistent session to work in. No email verification or password reset in MVP (ADR-0004) — just registration and login, backed by `go-authkit` in stateful mode wired through custom Huma handlers (ADR-0003).

**Blocked by:** 01 (Project & CI scaffolding) — done.

**Status:** ready-for-agent

**Plan:** `.scratch/cardstack-mvp/plans/02-auth-registration-login.md`

- [ ] User can register with an email and password
- [ ] User can log in with email and password
- [ ] Session persists across requests (stateful mode: session + refresh-token rotation)
- [ ] User can log out
- [ ] An authenticated-route guard on the backend rejects unauthenticated requests to protected endpoints
- [ ] Frontend redirects unauthenticated users away from protected routes
- [ ] No email verification or password-reset flow is present — this is deliberate, not a gap
- [ ] Auth endpoints appear in the generated OpenAPI spec and the frontend's generated client
