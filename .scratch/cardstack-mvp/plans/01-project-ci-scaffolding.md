# Plan: 01 — Project & CI scaffolding

Handoff doc for `backend-agent` / `frontend-agent`. Ticket: `.scratch/cardstack-mvp/issues/01-project-ci-scaffolding.md`.

Reference implementation: `/home/leon/Projects/itsLeonB/cashus/backend` (sibling repo, same author, same stack: Huma v2 + Gin + GORM + goose + wire + otel). Cited paths below are read-only references in that repo — open them directly for exact code shape rather than re-deriving from scratch. Do not copy app-specific business logic (auth, debts, monetization) — only the scaffolding patterns listed here.

**Out of scope for this ticket:** the Neon/Railway/Vercel preview-environment GitHub Action moved to ticket 10 (`.scratch/cardstack-mvp/issues/10-preview-environments.md`), blocked on ticket 01. Do not implement it here.

## Key decisions

These were decided in conversation, overriding my own initial (leaner) proposal — follow this column, not "simplest possible":

| Area | Decision | Why |
|---|---|---|
| DI | `google/wire`, from the start | Explicit user call — match cashus's `internal/provider` wire setup even though ticket 01 only wires two providers. |
| Router | Gin + `huma/v2/adapters/humagin` + `kroma-labs/sentinel-go` | Huma has no router of its own (adapts to stdlib/chi/gin/echo/fiber/gorilla) — user chose Gin+sentinel-go, matching cashus's `server.go` exactly. |
| Layout | Hexagonal, mirrors cashus's `internal/{core,domain,adapters,provider,endpoint}` | Explicit user call over a flatter `handler/service/repository` tree. |
| Migrations runner | Embedded goose FS + `cmd/job` (build-tag `job`), not a bare goose-CLI Makefile target | Matches cashus's `internal/adapters/job/migrate` — no goose CLI needed inside the deployed container. |
| Logging | Single zerolog instance for everything (`internal/core/logger`), bridged into GORM's logger via `ezutil/v2/gorm` | Explicit user call — "one logger for all," same as cashus. |
| otel | Ported near-verbatim from `internal/core/otel/otel.go`; `cfg.Enabled` gate, no collector, no `otel/` Dockerfile yet | Agreed as originally proposed. |
| Backend CI lint | `go vet ./...` + `golangci-lint run ./... --timeout=5m` (config ported from cashus's `.golangci.yml`) instead of a bare `gofmt -l` step | Explicit user call; golangci-lint's `gofmt`+`goimports` formatters cover the same ground plus more. |
| Repositories | `github.com/itsLeonB/go-crud`'s generic `crud.Repository[T]` / `crud.Transactor` for anything that's plain CRUD; hand-written interfaces in `domain/repository` only when a query needs more than that | Already the project's own prior decision (see memory / spec.md), confirmed by how cashus's `repository_provider.go` uses it. |

## Backend layout

```
backend/
  cmd/
    api/main.go              HTTP entrypoint: logger.Init → config.Load → otel.InitSDK → http.Setup → srv.ListenAndServe
    job/main.go               build-tag `job`; runs migrate.Run() — mirror cashus cmd/job/main.go
  internal/
    core/
      config/                 envconfig struct (App, DB, OTel), config.Global *Config, Load() — mirror internal/core/config
      logger/                  zerolog wrapper: logger.Global, Init(name), Error(err) — mirror internal/core/logger
      otel/                    InitSDK(ctx, cfg) — port internal/core/otel/otel.go near-verbatim, trim GCP-specific bits cardstack doesn't use
    domain/
      service/                 HealthService interface — the only domain interface this ticket needs
    adapters/
      http/
        handler/               health_handler.go
        huma/                  config.go (huma.DefaultConfig + BearerAuth scheme registered but unused this ticket — mirror internal/adapters/http/huma/config.go), envelope.go (Envelope[T]/NewEnvelope — mirror internal/adapters/http/huma/envelope.go verbatim)
        routes/                register_routes.go — mounts health route
        server.go               gin.New() + humagin + sentinel-go httpserver.Server — mirror internal/adapters/http/server.go
        setup_sentinel.go        port as-is from internal/adapters/http/setup_sentinel.go
      core/service/            HealthService impl (trivial — returns {status: "ok"})
      db/postgres/
        migrations/             goose .sql files
      job/migrate/             Setup(providers)/Run() goose runner — port internal/adapters/job/migrate/migrate.go, adjust import paths
    endpoint/                  port verbatim from internal/endpoint/{endpoint.go,registrable.go}: Endpoint[Req,Res], NoBodyEndpoint, ListEndpoint, RedirectEndpoint, Registrable, RegisterAll. Generic, app-agnostic — no changes needed beyond the import path for httpapi.Envelope.
    provider/
      providers.go             Providers{*DataSources, *Services} — no Repositories/CoreServices struct yet, nothing to put in them
      data_sources_provider.go ProvideDataSource(cfg) — gorm.Open(postgres.Open(dsn)), pool settings (MaxOpenConns/MaxIdleConns/ConnMaxLifetime), Ping — mirror internal/provider/datasource/sql_data_source.go's dsn()/ProvideAndConfigureSQL(), but no sync.Once singleton: wire already guarantees single construction, so drop that guard (this is the one place we diverge from copying cashus outright — it's solving a problem wire doesn't have)
      service_provider.go      ServiceSet — HealthService
      wire.go                  `//go:build wireinject`; ProviderSet = wire.NewSet(DataSourceSet, ServiceSet, wire.Struct(new(Providers), "*")); InitializeProviders()
      wire_gen.go               generated via `go generate ./internal/provider/...` (`make wire`)
  go.mod
  Makefile                    http/http-hot/wire/lint/vulncheck/test/build targets — mirror cashus's Makefile, trim worker/job-specific targets except `job` (needed for migrations)
  Dockerfile                  single image (no Dockerfile.worker/.job split — MVP has one deployable process, `cmd/job` runs migrations via `-tags job` in an init/pre-deploy step, not a separate image)
  .golangci.yml               copy cashus's verbatim: staticcheck all minus SA1019/ST1003/ST1000, gofmt+goimports formatters
  embed.go                    go:embed for migrations FS, passed to goose.SetBaseFS in job/migrate
```

Do **not** create `domain/entity`, `domain/repository`, `adapters/repository`, `adapters/core/service` (queue/mail/etc equivalents) yet — nothing in ticket 01 needs a real entity or a business-logic service beyond health. They get created by whichever ticket (02+) first needs them, same shape as cashus, not pre-scaffolded empty.

### Dependencies to add (go.mod)

`danielgtaylor/huma/v2`, `huma/v2/adapters/humagin`, `gin-gonic/gin`, `kroma-labs/sentinel-go`, `gorm.io/gorm` + `gorm.io/driver/postgres`, `pressly/goose/v3`, `google/wire` (+ `tool github.com/google/wire/cmd/wire`), `kelseyhightower/envconfig`, `joho/godotenv`, `itsLeonB/ezutil/v2` (zerolog + gorm logger bridge), the `go.opentelemetry.io/*` set cashus uses (`contrib/exporters/autoexport`, `contrib/propagators/autoprop`, `otel`, `otel/sdk`, `otel/sdk/metric`, `otel/sdk/log`, `otel/trace`), `github.com/itsLeonB/go-crud` (for when repositories land, but wire it up now since `crud.Transactor` is part of the standard provider shape — see `repository_provider.go`'s `ProvideTransactor`, only add this once ticket 02 actually needs a repository; skip it in ticket 01 if nothing uses it, don't add unused deps).

### Health endpoint

`GET /health` registered through the ported `endpoint.Endpoint[Req,Res]` wrapper (unsecured, `Secured: false`), response body wrapped in `Envelope[HealthResponse]` — this is what proves Huma → OpenAPI → orval end to end.

### Spec bridging

`backend/openapi.json` generated via a small `cmd/genspec` (boots the Huma routes against `httptest`-style setup, dumps the spec, doesn't bind a port) and **committed**. Frontend's orval config reads that committed path. Frontend generated client (`frontend/src/generated/`) is also committed, not regenerated in CI — keeps each component's CI self-contained under its own path filter.

## Frontend layout

1. Add `orval` dev dependency, config pointed at `../backend/openapi.json`, output `src/generated/` (zod schemas + TanStack Query hooks).
2. `bun run orval` (or a `codegen` package.json script) produces the health client + hook.
3. A route calls the generated health hook and renders the result.
4. Existing `lint`/`typecheck`/`test`/`build` scripts in `frontend/package.json` need no changes — just get wired into CI.

## CI (repo root — orchestrator writes these, not delegable to either component agent)

`.github/workflows/backend-ci.yml` (path filter `backend/**`):
- `build`: `go build ./...`
- `test`: `go test ./...` (mirror cashus's `-race`, consistent with ADR 0005's real-Postgres testing — provision a Postgres service container in the job)
- `lint`: `go vet ./...` then `golangci-lint run ./... --timeout=5m`

`.github/workflows/frontend-ci.yml` (path filter `frontend/**`): lint / typecheck / test / build, one job each, mirroring cashus's `frontend-ci.yml` job shape.

Reuse `/home/leon/Projects/itsLeonB/cashus/.github/actions/{setup-go,setup-bun}` composite actions verbatim (copy into `cardstack/.github/actions/`).

## Execution routing

Per `docs/agents/orchestration.md`: touches both `./backend` and `./frontend` → multi-component task.

1. `backend-agent` in its own worktree: everything under "Backend layout" above.
2. `frontend-agent` in its own worktree: everything under "Frontend layout" above. Blocked on backend's `openapi.json` existing — sequence backend first, or have frontend-agent stub against a hand-written placeholder spec and regenerate once backend lands.
3. Orchestrator (root, directly, no worktree): both CI workflow files, the composite actions.
4. Each subagent runs its own verification script, self-reviews via `code-review` skill, commits on its own branch (`feat(backend): scaffold project`, `feat(frontend): scaffold project`).
5. Orchestrator runs a cross-component architecture review on the merged diff, merges worktrees back, commits CI files, pushes after confirmation.
