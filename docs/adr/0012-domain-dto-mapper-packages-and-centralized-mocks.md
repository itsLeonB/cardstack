# internal/domain sub-packages: dto and mapper split from service; mocks are centralized

Continuing ADR-0011: within `internal/domain`, plain data-transfer types live in `internal/domain/dto` (a type reusable beyond one feature gets its own file, e.g. `pagination.go`; feature-specific types share a `<feature>_dto.go`, e.g. `catalog_dto.go`), conversion functions between repository/entity types and those DTOs live in `internal/domain/mapper` (e.g. `catalog_mapper.go`), and every mockery-generated mock lands in one top-level `internal/mocks` directory instead of a per-package `mocks/` folder. Decided via PR review on itsLeonB/cardstack#10, after catalog's DTOs and mapper functions were first inlined directly into `catalog_service.go`.

## Considered Options

- Keep DTOs and mapper functions inlined in the service file that uses them (the original approach). Rejected: `PaginationMeta` is used beyond `catalog_service.go`, so leaving it catalog-scoped would force a future consumer to reach into a package named after a different feature for a generic type; inlining the mapper functions as unexported also closed off reuse even where the conversion logic (entity → summary DTO) is genuinely feature-agnostic.
- Leave mocks per-package (mockery's default `{{.InterfaceDir}}/mocks`), next to each interface's own package. Rejected in favor of one `internal/mocks` directory: with the interface count still small, a single directory is easier to scan than hunting through `internal/domain/*/mocks`, and a mock is an internal test seam, not something whose location should be driven by which package happens to declare the interface.

## Consequences

New DTOs default to `internal/domain/dto` (own file if reusable across features, a shared `<feature>_dto.go` otherwise) rather than living inside the service that first needed them. New cross-type conversion logic defaults to `internal/domain/mapper`. `domain/mapper` may import `domain/dto`, `domain/repository`, and `domain/entity`, but never `domain/service` — that's what lets `domain/service` import `domain/mapper` without a cycle. `.mockery.yaml`'s output directory stays a single top-level `internal/mocks` for any newly-mocked interface.
