# Plan: 02 — Auth: registration & login

Handoff doc for `backend-agent` / `frontend-agent`. Ticket: `.scratch/cardstack-mvp/issues/02-auth-registration-login.md`. Blocked by ticket 01 (done — see `01-project-ci-scaffolding.md`).

Reference implementation: `github.com/itsLeonB/go-authkit` (same author, the library ADR-0003/ADR-0004 and `spec.md` name explicitly). No local reference backend was available for this ticket the way cashus was for ticket 01, so this plan is grounded directly in `go-authkit`'s source (`auth.go`, `session.go`, `middleware.go`, `stores.go`, `config.go`, `errors.go`, `authgin/{handler,transport,middleware}.go`) rather than a sibling app's usage of it. Verify exact signatures against the actual installed module version at implementation time — the library is under active development.

## Key decisions

| Area | Decision | Why |
|---|---|---|
| Auth mode | `authkit.New(cfg, deps, hooks)` in **stateful** mode (`Stateless: false`, the zero value) | ADR-0003/spec.md: session + refresh-token rotation, not a bare Bearer flow. |
| Wiring style | Custom Huma handlers in `adapters/http/handler/auth_handler.go` call `kit.Register`/`kit.Login`/`kit.Logout`/`kit.RefreshToken`/`kit.VerifyToken` directly — no `authgin.Handler`, no `authgin.AuthMiddleware`, no second router | ADR-0003. `authgin.Handler`'s routes and `authgin.AuthMiddleware` are `gin.HandlerFunc`-shaped and only usable if authgin owns routing; we keep Huma as the single router. |
| Email verification | `Config.VerificationURL` and `Config.ResetPasswordURL` left `""`. `Deps.Mail` left `nil`. | `authkit.Register`'s own logic is `isVerified := kit.cfg.VerificationURL == ""` — with it empty, `Register` marks the user verified immediately and never calls `mail.SendVerification`. This *is* ADR-0004's "no email verification, no Resend integration" — not a workaround bolted on top, the library's own no-URL path already does it. Same reasoning keeps `ResetPassword`/`SendPasswordReset` unreachable (frontend simply never calls them) without needing to strip anything out of `authkit`. |
| Cookie transport | Import and reuse `github.com/itsLeonB/go-authkit/authgin.CookieTransport` directly (just this one type) for setting/reading the access/refresh/fingerprint/CSRF cookies | Despite living in the `authgin` package, `CookieTransport`'s methods operate on plain `http.ResponseWriter`/`*http.Request` — zero Gin dependency. Reusing it isn't "running authgin" in the sense ADR-0003 rejected (a second router); it just saves re-deriving cookie names/paths/TTLs/the `__Secure-Fgp` fingerprint cookie by hand. If this reasoning doesn't hold up on a closer read of ADR-0003 at implementation time, the fallback is a ~40-line hand-rolled equivalent in `adapters/http/auth/transport.go` — the shape is simple enough to port. |
| Session guard middleware | New Huma middleware (`func(huma.Context, func(huma.Context))`) in `adapters/http/auth/middleware.go` that ports `authgin.AuthMiddleware`'s three-line body (`transport.ReadAccessToken` → `transport.ReadFingerprint` → `kit.VerifyToken` → stash claims) to Huma's context shape | The logic is already framework-agnostic; only its `gin.HandlerFunc` signature and `gin.Context.Set` calls don't transfer. Claims go into `context.Context` via a typed key (e.g. `authctx.UserID(ctx)`) instead of Gin's context map. |
| CSRF | Same treatment for `authgin.CSRFMiddleware` (double-submit `csrf_token` cookie vs. `X-CSRF-Token` header, skipped on GET/HEAD/OPTIONS) — port as a second Huma middleware, applied to `logout` and `refresh` (the two mutating routes that run against an *existing* session/cookie set) | Register/login are the requests that *create* the CSRF cookie in the first place, so there's nothing to double-submit against yet on that first call — matches why `sentinel-go`'s CORS config (already scaffolded in ticket 01, `setup_sentinel.go`) already allow-lists `X-CSRF-Token` and sets `AllowCredentials: true`, i.e. this was anticipated, not new scope creep. |
| **OpenAPI security scheme correction** | Replace `huma/config.go`'s `"BearerAuth"` scheme (`type: http, scheme: bearer`) with a cookie-based one — `type: apiKey, in: cookie, name: "access_token"`, and rename `internal/endpoint`'s hardcoded `bearerAuthSecurity` var (and the `"BearerAuth"` map key) to match, e.g. `cookieAuthSecurity` / `"CookieAuth"` | Ticket 01 scaffolded `BearerAuth` as a placeholder before any real auth flow existed ("registered but unused"). This ticket is that flow landing, and the flow the project actually chose (ADR-0003 stateful mode) authenticates via cookies, not an `Authorization: Bearer` header. Leaving `BearerAuth` as-is would make `Secured: true` routes advertise a transport the middleware doesn't actually check — the generated OpenAPI spec (and therefore orval's client) would describe the wrong auth mechanism. This is a one-time correction to `internal/adapters/http/huma/config.go` and the one `var` in `internal/endpoint/endpoint.go`; no other file in `endpoint/` needs to change since `Secured: true` already plumbs through generically. |
| Stores | Hand-written GORM repositories (`adapters/repository/{user,session,refresh_token}_repository.go`) implementing `authkit.UserStore`, `authkit.SessionStore`, `authkit.RefreshTokenStore`. `Deps.Resets`/`Deps.OAuth`/`Deps.State` left `nil` (unused — no password reset, no OAuth this ticket). | Matches ticket 01's own repository rule (spec.md / decision table): `go-crud`'s generic `crud.Repository[T]` covers plain CRUD, but `authkit`'s store interfaces need queries `go-crud` doesn't shape (`FindByEmail`, `SetVerified` with side-fields, hashed-token lookup by hash, session touch). This is also the first ticket that actually needs a repository, so it's the point ticket 01 deferred `go-crud`/`crud.Transactor` wiring to (`provider/repository_provider.go`, `ProvideTransactor`, mirroring cashus's shape) — add it now, not before. |
| Session cache | Minimal in-process `SessionCache` (`adapters/core/service` or a small `internal/core/sessioncache` package — a mutex/`sync.Map`-backed map with TTL eviction, satisfying `authkit.SessionCache`'s 3-method interface) | MVP runs one Railway instance; no multi-instance cache-consistency requirement yet that would justify Redis. `SessionCache` is a tiny interface (`Get`/`Delete`/`Shutdown`) — hand-rolling it is less work and less infra than adding a dependency for a single-process app. Revisit if the backend ever scales beyond one instance. |
| Schema | New goose migration (`internal/adapters/db/postgres/migrations/`) creating `users`, `sessions`, `refresh_tokens` — replaces the ticket-01 bootstrap placeholder's "no schema changes yet" state | The bootstrap migration's own comment says to replace it once real schema exists; this is the first ticket that needs any. PG18 native `gen_random_uuid()` for PKs (per ticket 01's decision, no pgcrypto extension). |
| Auth config | New `internal/core/config/auth_config.go` (`Auth` struct, prefix `AUTH`): `JWTSecret`, `JWTIssuer` (default `cardstack`), `JWTDuration` (default `15m`), `RefreshTokenTTL` (default `168h`), `CookieDomain`, `CookieSecure` (default `true`), `CookieSameSite` (default `Lax`) — added to the top-level `config.Config` struct alongside `App`/`DB`/`OTel` | Mirrors how `App`/`DB`/`OTel` are already loaded (`envconfig.Process(prefix, &struct)` in `config.Load()`); auth secrets/TTLs are config, not hardcoded. |
| `GET /auth/me` | Add a small secured endpoint returning the current session's user (id, email) — **not** explicitly listed in the ticket checklist, added as necessary connective tissue | Access/fingerprint/refresh cookies are `HttpOnly` by design (`CookieTransport`) — frontend JS cannot read them to decide "am I logged in." Without some endpoint to probe, "frontend redirects unauthenticated users away from protected routes" (an actual checklist item) has no way to determine auth state on page load/refresh. `/auth/me` is the standard shape for this and reuses the same session-guard middleware as every other secured route — no new auth logic. |
| Frontend cookie transport | Extend `frontend/orval.config.ts`'s fetch output with a `mutator` (or an equivalent wrapper module under `src/lib/`) that sets `credentials: "include"` on every generated call, and attaches `X-CSRF-Token` (read from the non-`HttpOnly` `csrf_token` cookie) on mutating requests | The current generated client (`src/generated/endpoints/health/health.ts`, ticket 01) calls bare `fetch()` with no `credentials` option — cookies are never sent cross-origin (frontend on Vercel, backend on Railway in prod; different ports in dev) without this. This has to land now, the first ticket where any endpoint relies on cookies, or every generated auth call silently fails. |

## Backend layout (new/changed files)

```
backend/
  internal/
    core/
      config/
        auth_config.go          Auth struct (JWT secret/issuer/duration, refresh TTL, cookie domain/secure/samesite) — wired into config.Config + config.Load()
    domain/
      service/
        auth_service.go          AuthService interface: Register/Login/Logout/RefreshToken/Me — thin wrapper the handler calls, matching HealthService's shape
    adapters/
      core/
        service/
          auth_service.go         AuthService impl: holds *authkit.AuthKit, calls its methods, maps authkit sentinel errors (ErrUserExists, ErrInvalidCredentials, ErrSessionNotFound, ...) to the project's HTTP error conventions
          session_cache.go        hand-rolled authkit.SessionCache (sync.Map + TTL eviction + background sweep, Shutdown stops the sweeper)
      repository/
        user_repository.go        implements authkit.UserStore over GORM
        session_repository.go     implements authkit.SessionStore over GORM
        refresh_token_repository.go  implements authkit.RefreshTokenStore over GORM
      http/
        auth/
          transport.go            re-exports/wraps authgin.CookieTransport construction from config.Auth (or a hand-rolled equivalent — see Key decisions)
          middleware.go            SessionGuard (ports authgin.AuthMiddleware) + CSRFGuard (ports authgin.CSRFMiddleware), both as func(huma.Context, func(huma.Context))
          claims.go                 typed accessors (UserID(ctx), SessionID(ctx)) over the context values SessionGuard stashes
        handler/
          auth_handler.go          Register/Login/Logout/Refresh/Me — mirrors health_handler.go's Routes() []endpoint.Registrable shape; Logout/Refresh/Me use Secured: true (Refresh's "auth" is its own refresh-cookie check inside authkit, not the access-token guard — see below) plus the CSRF middleware on Logout/Refresh via Endpoint.Middlewares
        huma/
          config.go                 EDIT: "BearerAuth" → cookie-based apiKey scheme (see Key decisions)
        routes/
          register_routes.go        EDIT: mount authHandler.Routes() alongside health
      db/
        postgres/
          migrations/
            <timestamp>_auth_schema.sql   users / sessions / refresh_tokens tables — replaces the bootstrap placeholder's "no schema changes yet" note
    endpoint/
      endpoint.go                  EDIT: rename bearerAuthSecurity → cookieAuthSecurity (see Key decisions); no structural change
    provider/
      repository_provider.go       NEW: RepositorySet — provides the three GORM repositories + crud.Transactor (go-crud, added to go.mod now)
      auth_provider.go             NEW: builds authkit.Config from config.Global.Auth, authkit.Deps from the repository providers + session cache, constructs *authkit.AuthKit
      service_provider.go          EDIT: Services gains Auth service.AuthService
      wire.go / wire_gen.go        EDIT: regenerate (`make wire`) to include RepositorySet + auth provider
  go.mod                            ADD: github.com/itsLeonB/go-authkit, github.com/itsLeonB/go-crud
  .env.example                      ADD: AUTH_JWT_SECRET, AUTH_JWT_ISSUER, AUTH_JWT_DURATION, AUTH_REFRESH_TOKEN_TTL, AUTH_COOKIE_DOMAIN, AUTH_COOKIE_SECURE, AUTH_COOKIE_SAMESITE
```

### Route surface

All under `/auth`, registered through the existing `endpoint.Endpoint[Req,Res]`/`NoBodyEndpoint` wrappers (no new registration mechanism needed):

- `POST /auth/register` — `{email, password, passwordConfirmation}` → 201, `{message}`. Unsecured.
- `POST /auth/login` — `{email, password}` → 200, sets access/refresh/fingerprint/csrf cookies via the Output struct's `Set-Cookie` header(s). Unsecured (this *is* the auth step).
- `POST /auth/logout` — no body → 204, clears cookies. `Secured: true` (session guard) + CSRF guard.
- `POST /auth/refresh` — no body, reads `refresh_token` cookie → 200, rotates + resets cookies. `Secured: false` for the access-token guard (the whole point is the access token may be expired), but still runs the CSRF guard; the refresh cookie itself is authkit's real check (`kit.RefreshToken` returns `ErrTokenInvalid`/`ErrTokenExpired` otherwise).
- `GET /auth/me` — → 200, `{id, email}`. `Secured: true`.

Set-Cookie handling: prefer Huma's native `[]http.Cookie` output field tagged `header:"Set-Cookie"` (Huma has built-in multi-cookie support for this) over manually unwrapping to `*gin.Context`/`http.ResponseWriter` — verify the exact tag/type via context7's Huma docs at implementation time; if it doesn't hold up, fall back to `humagin.Unwrap(ctx)` inside the handler.

### Error mapping

`authkit` returns its own sentinel errors (`ErrUserExists`, `ErrInvalidCredentials`, `ErrSessionNotFound`, `ErrTokenInvalid`, `ErrTokenExpired`, `ErrTooManyRequests`, ...) — map these to `huma.Error4xx`/existing error-handling conventions in `auth_service.go` (or a shared `errors.go` in `adapters/core/service`), the same boundary point ticket 01 didn't need since health has no error paths.

## Frontend layout

1. `bun run orval` regenerates `src/generated/` once the backend's `openapi.json` includes `/auth/*` — new `auth` tag directory alongside `health`.
2. Add the `credentials: "include"` + CSRF-header wrapper (see Key decisions) — likely `src/lib/http.ts`, wired in as orval's fetch `mutator` or by having the generated calls import a shared fetch wrapper. Confirm the exact orval mechanism (mutator vs. baseUrl-adjacent option) via context7/the `tanstack-query` skill at implementation time.
3. A small auth/session layer: `useSession()` (TanStack Query hook wrapping `GET /auth/me`, `staleTime`/`retry` tuned so a 401 is treated as "logged out" not an error toast) + login/register/logout mutations using the generated hooks.
4. Register and login forms/routes (`src/routes/register.tsx`, `src/routes/login.tsx`).
5. Route-guard mechanism: a `beforeLoad` (or root-level check) on protected routes that redirects to `/login` when `useSession()`/an auth context says unauthenticated. `src/routes/__root.tsx` is the natural place for the shared auth context/loader if TanStack Router's context mechanism is used — confirm current `__root.tsx` shape before adding to it (not yet read in this planning pass).
6. Logout action clears session client-side (invalidate the `/auth/me` query) after the backend call succeeds.
7. Existing `lint`/`typecheck`/`test`/`build` scripts need no changes — new code just needs to pass them.

## CI

No changes expected — ticket 01's `backend-ci.yml`/`frontend-ci.yml` already provision Postgres, run `go build`/`go vet`/`golangci-lint`/`go test` and lint/typecheck/test/build respectively; new auth code and tests run through the same jobs. Only touch CI files if a new env var needs to reach the test job (e.g. `AUTH_JWT_SECRET` for backend feature tests against real Postgres, per ADR-0005) — add as a job-level env, not a new workflow.

## Testing (per spec.md's testing decisions)

- Repository tests: real local Postgres (user/session/refresh-token CRUD, `FindByEmail`, hashed-token lookup).
- Service tests: mock `authkit`-shaped store interfaces (or mock `*authkit.AuthKit` behavior at the `AuthService` boundary) via `mockery`.
- Feature tests: full HTTP boundary — register → login → cookies set → `/auth/me` returns the user → logout → `/auth/me` 401s → refresh flow. This is the first ticket exercising `endpoint.Endpoint`'s `Secured`/`Middlewares` fields for real, so also a natural place to add a feature test asserting an unauthenticated request to a `Secured: true` route is rejected (a ticket checklist item).
- Frontend: `useSession`/login/register component tests mocking the generated client module (`vi.mock`), per spec.md's frontend testing decision (no MSW).

## Execution routing

Per `docs/agents/orchestration.md`: touches both `./backend` and `./frontend` → multi-component task.

1. `backend-agent` in its own worktree: everything under "Backend layout" above, in this rough order — config → migration → repositories → session cache → `authkit` wiring in `provider/` → middleware/transport → handlers/routes → the `huma/config.go` security-scheme correction → tests.
2. `frontend-agent` in its own worktree: everything under "Frontend layout" above. Blocked on backend's `openapi.json` regenerating with `/auth/*` — sequence backend first (unlike ticket 01, there's no reasonable placeholder spec to stub against here, since the whole point is exercising real cookie auth).
3. Orchestrator (root, directly, no worktree): no dedicated root-only file this ticket (no new CI expected — see above); if a CI env var addition does turn out to be needed, do it directly rather than through a subagent.
4. Each subagent runs its own verification script, self-reviews via `code-review` skill, commits on its own branch (`feat(backend): add registration and login`, `feat(frontend): add registration and login`).
5. Orchestrator runs a cross-component architecture review on the merged diff — pay particular attention to the cookie/CSRF contract actually matching between backend `Set-Cookie` behavior and the frontend fetch wrapper (this is the one place a mismatch would pass both components' own tests but fail end-to-end) — merges worktrees back, pushes after confirmation.

## Open questions worth a quick confirmation before/while implementing

- **CSRF scope on register/login**: this plan applies the CSRF guard only to logout/refresh (routes that ride an existing cookie set), not register/login (which create it). This defends against classic CSRF on state-changing authenticated actions but not "login CSRF" (tricking a victim into authenticating as the attacker). Given MVP is single-owner/personal use (ADR-0004's own framing), this plan treats that as acceptable for now — flag if that framing has changed.
- **`GET /auth/me` naming/shape**: not in the ticket's checklist; confirm the addition (or an equivalent mechanism) is wanted before or during implementation rather than after, since the frontend redirect-guard checklist item depends on it existing in some form.
