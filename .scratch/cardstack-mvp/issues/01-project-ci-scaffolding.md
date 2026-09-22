# 01: Project & CI scaffolding

**What to build:** The foundational toolchain round-trip that every other ticket builds on: a Huma-based Go API with a layered (handler/service/repository) skeleton, Postgres wired through GORM and goose, an OpenAPI spec generated from it, and a frontend that consumes that spec via orval-generated zod schemas and TanStack Query hooks. CI runs both components' verification scripts on every PR.

**Blocked by:** None (can start immediately)

**Status:** done

- [x] Backend: layered skeleton (handler/service/repository) with a Huma-based HTTP server boots and serves a health-check endpoint
- [x] Postgres connection configured via GORM; goose migration tooling wired with at least one initial migration
- [x] Backend's OpenAPI spec is generated automatically from the Huma routes
- [x] Frontend: orval configured to generate zod schemas + TanStack Query hooks from the backend's OpenAPI spec, and successfully generates a client for the health endpoint
- [x] Frontend calls the health endpoint via the generated client and renders the result, proving the full toolchain round-trip end to end
- [x] GitHub Actions CI runs backend verification (`go build`/`go vet`/`gofmt -l`/`go test`) and frontend verification (lint/typecheck/test/build) on every PR
- [x] otel instrumentation points are scaffolded in the backend, but tracing is disabled (no collector wired) — enabling it later should require no code changes

## Comments

Implemented and merged (see `main` history: backend scaffold, frontend scaffold, CI workflows, composite actions). Verified: `backend/internal/domain/service/health_service.go`, `backend/internal/core/otel/otel.go`, `frontend/src/routes/health.tsx`, `frontend/src/generated/`, `.github/workflows/{backend-ci,frontend-ci}.yml` all present on `main`.
