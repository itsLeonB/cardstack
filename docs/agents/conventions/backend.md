# Backend code conventions

These apply to every backend change, whether made by the root agent or a subagent. Read `general.md` in this folder as well.

## Errors

- Wrap with `ungerr.Wrap`/`ungerr.Wrapf` at the exact first location an error originates in our own code (e.g. inside the helper that calls `rand.Read`). Callers of our own functions, and of `crud.Repository` (it already wraps with ungerr), return the plain `err` unchanged. The single Huma-level seam (`backend/internal/adapters/http/huma/errors.go`, ADR-0013) classifies and unwraps it once.
- Return known, client-safe failures as `ungerr.XxxError(...)` AppErrors. Handlers return AppError types rather than calling `huma.ErrorXXX(...)` with ad hoc messages (ADR-0013).
- Handle every error. Log a non-blocking one with `logger.Error`/`logger.Errorf` (`backend/internal/core/logger`, whose `Global` is a safe no-op until `Init` runs) and carry on; return the rest. Never discard with `_ =`.

## Layout

- Put a Service in `internal/domain/service/<name>_service.go`, interface and implementation in that one file. Give an entity its own Repository (`internal/domain/repository/<name>_repository.go`, same one-file shape) only when it needs custom GORM calls or raw SQL; otherwise the service holds a plain `crud.Repository[entity.X]`. Reserve `internal/adapters/` for modules with a real second implementation. Read `docs/adr/0011-domain-owns-business-logic-and-data-access-adapters-are-for-interchangeable-infrastructure.md` before placing a new Service or Repository.
- Put DTOs in `internal/domain/dto` and entity-to-DTO conversions in `internal/domain/mapper` (ADR-0012).

## Data and mapping

- Key domain ownership by `profile_id` (`user_profiles.id`); `user_id` identifies the authentication account only.
- Scope a query in the query itself with `crud.Specification`, never by fetching rows and filtering in Go.
- Map slices with `ezutil.MapSlice`, or `ezutil.MapSliceWithErr` when the mapper can fail.

## Testing

- Mock any dependency you'd rather not stand up (a repository, a service behind a handler): add the interface to `.mockery.yaml`, run `make mocks`, and use the generated mock, never a hand-written stub or fake. Mocks land in the single committed `internal/mocks` directory (ADR-0012). Handler tests show the pattern with `mocks.NewMockCatalogService(t)` in `internal/adapters/http/handler/catalog_handler_test.go`.
- Run repository-layer tests (`internal/domain/repository`) and feature tests against a real Postgres, because a mock of the DB/GORM layer would reimplement SQL behavior and drift from it (ADR-0005). Tests that merely sit downstream of a repository mock it instead.
- Have a service in `internal/domain/service` depend on its repository through the single exported interface defined beside the concrete implementation (`repository.CatalogRepository` in `internal/domain/repository/catalog_repository.go`), not an interface narrowed to that one service. Its unit tests mock that interface: see `internal/domain/service/catalog_service_test.go` (`mocks.NewMockCatalogRepository(t)`, `.EXPECT().Method(args).Return(...)`).
- Assert with `github.com/stretchr/testify/assert` (`require` when the test cannot continue), not hand-rolled `if ... t.Fatalf`.
- Unit-test a service that holds a plain `crud.Repository[entity.X]` with the generated generic mock `mocks.NewMockRepository[entity.X](t)` (go-crud's `Repository` entry in `.mockery.yaml`). Assert the exact `crud.Specification` passed, since the query carries the scoping.
- Environment setup for tests (Postgres, seed data) lives in `docs/agents/testing.md`.
