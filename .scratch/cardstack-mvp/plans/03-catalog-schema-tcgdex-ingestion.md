# Plan: 03 — Card catalog schema + TCGDex (SV) ingestion

**Status:** not started — planning only.

Handoff doc for `backend-agent`. Ticket: `.scratch/cardstack-mvp/issues/03-catalog-schema-tcgdex-ingestion.md`. Blocked by ticket 01 (done). Backend-only — no frontend surface in this ticket (catalog *read* API/UI is ticket 05).

Reference: no local reference backend for TCGDex the way `go-authkit` gave ticket 02 a source tree to read. This plan is grounded in live responses pulled from the real API (`https://api.tcgdex.net/v2`) during planning — exact shapes below, captured 2026-09-22. TCGDex has no versioned client library for Go; verify shapes haven't drifted at implementation time (the `updated` timestamp on a card response is the freshness signal).

## Key decisions

| Area | Decision | Why |
|---|---|---|
| Schema shape | Four tables — `games`, `expansion_sets`, `cards`, `card_variants` — each `crud.BaseEntity` (uuidv7 PK, GORM), one goose migration | Matches ADR-0001's `Game → Expansion Set → Card → Card Variant` shape and ticket 01/02's migration style (`20260921000000_auth_schema.sql`). |
| `expansion_sets` uniqueness | `UNIQUE (game_id, code)` — `code` is TCGDex's own set ID (`"SV1V"`) | Literal ticket wording. |
| `expansion_sets.locale` | Add a `locale` column (e.g. `"id"`) — **not** part of the unique key, just descriptive/source metadata | `CONTEXT.md`'s single most emphasized invariant is that an Expansion Set is never assumed shared across regions. TCGDex set codes already happen to be region-specific in practice (confirmed: `SV1V` only exists under the `id` locale path), so `(game_id, code)` alone is safe for MVP — but recording the source locale as a real column costs one field and makes the invariant structural instead of implicit in "whatever code TCGDex happened to assign," and gives ticket 04 (MA, a different source entirely) something to leave differently. |
| `cards` uniqueness | `UNIQUE (expansion_set_id, local_id)` — `local_id` is TCGDex's `localId` (`"008"`, zero-padded string, not an int) | Literal ticket wording. Stored as `TEXT`, not `INT` — TCGDex's own values are zero-padded strings and some sets carry non-numeric local IDs (promos); parsing to int would lose the padding and break on non-numeric ones for no benefit. |
| `cards.names` | `JSONB` map, e.g. `{"id": "Spidops ex", "ja": "ワナイダーex", "zh-tw": "操陷蛛ex", "th": "วาไนเดอร์ex"}` — not four fixed columns | TCGDex has no single endpoint returning all locale names together (confirmed: each locale is a fully separate path, `/v2/{locale}/cards/{id}`, differing only in `name`) — ingestion makes one call per locale per card and merges. A JSONB map handles that a card may only have *some* locales' data (English SV lookups for `SV1V-008` 404 entirely), and ticket 05's own search decision ("matches card name in any locale the source data carries") already implies variable, not fixed-cardinality, locale sets — ticket 04's MA scrape may only ever populate one key. |
| `cards.attributes` | `JSONB` — HP, types, attacks, abilities, weaknesses, retreat, stage, suffix, regulationMark, dexId, illustrator, category (see shape below) | ADR-0001: game-specific attributes attach per-Game, not as fixed `cards` columns — Riftbound's attributes won't be HP/types/attacks. `rarity` and `image_url` stay first-class columns since they're cross-game concerns ticket 05 explicitly filters/displays on. |
| `card_variants.finish` | One row per **true** flag in TCGDex's `variants` map: `normal`, `reverse`, `holo`, `first_edition`, `w_promo` — `TEXT` with a `CHECK` constraint, not a Postgres `ENUM` type | TCGDex's actual flag set is five, not the ticket text's four ("normal, reverse holo, holo, first edition, **etc.**") — `wPromo` is real and appears on genuine promo prints. A `CHECK` constraint extends with a one-line migration later; a Postgres `ENUM` type needs `ALTER TYPE ... ADD VALUE` ceremony for the same change. `UNIQUE (card_id, finish)`. |
| Ingestion CLI | New standalone binary `cmd/ingest-tcgdex/main.go` — **not** folded into `cmd/job` | `cmd/job` runs unconditionally on every deploy (`preview-environments.yml` calls `go run ./cmd/job` as a pipeline step). Putting ingestion there would violate the ticket's explicit "manually triggered only — no scheduler or cron job." A dedicated binary, invoked only by a developer running `make ingest-tcgdex`, is the only way that requirement actually holds. |
| CLI wiring | Reuses `provider.InitializeProviders()` (same as `cmd/job`/`cmd/genspec`) for `DataSources`, but does **not** add catalog repositories to `provider/repository_provider.go` / `ServiceSet` | Nothing else needs these repositories yet — ticket 05 is what wires a real read-facing `CatalogService`. Constructing `crud.Repository[entity.X]` directly from `providers.DataSources.Gorm` inside the ingestion package mirrors `migrate.Setup(providers)`'s existing lightweight pattern (takes `providers.SQL` directly, no wire graph of its own). Avoids adding an unused abstraction layer this ticket doesn't need. |
| Upsert strategy | Find-by-natural-key (`FindFirst` on the unique tuple) then `Insert`/`Update` — same shape as `user_repository.go`'s `upsertProfile` | Makes re-running the command idempotent: a partial failure, a TCGDex data correction, or a deliberate re-seed doesn't duplicate rows or require a manual `DELETE` first. `go-crud`'s `SaveMany` upserts by primary key, which is useless here since the PK is a generated UUID, not TCGDex's natural key. |
| HTTP client | Stdlib `net/http` + `encoding/json`, no third-party client library | A handful of GET calls against a plain JSON REST API — nothing a dependency buys over ~30 lines of stdlib. |
| Concurrency | Bounded worker pool: `golang.org/x/sync/errgroup` (already an *indirect* `go.mod` dependency — promote to direct) + a semaphore channel, cap ~8–16 concurrent requests | A full SV/`id` pull is 23 sets × up to ~180 cards × up to 4 locale name-lookups ≈ several thousand HTTP requests. Serial would make a "quick manual re-run" take many minutes; unbounded concurrency risks hammering TCGDex's public API. A bounded pool is the standard shape for this and costs no new dependency. |
| Game seeding | The ingestion command itself find-or-creates a single `games` row (`slug: "pokemon-tcg"`) at the start of `Run()` — no separate seed migration/fixture | Keeps this ticket self-contained. Ticket 04's MA ingestion finds the same row by slug rather than creating a duplicate — this is exactly the "adding a new Expansion Set/source shouldn't require migrating existing data" property ADR-0001 is arguing for. |
| Skipped: TCGDex's own card `id` (e.g. `"SV1V-008"`) as a stored column | Not stored — `(expansion_set_id, local_id)` is already the ticket's stated natural key and is sufficient for idempotent upsert on its own | Storing it too would be a second, redundant identity column nothing in this ticket or its downstream tickets (05, 08) needs. Add it later if a TCGDex-specific re-sync/debug need actually shows up. |
| Skipped: image URL suffixing | `cards.image_url` stores TCGDex's raw `image` field as-is (e.g. `https://assets.tcgdex.net/id/SV/SV1V/008`, no size/format suffix) | TCGDex requires appending a quality+format suffix (e.g. `/high.webp`) to actually load an image — but that's a *rendering* concern for whichever ticket first displays card images (05 or later), not an ingestion concern. Storing the raw URL keeps ingestion a faithful mirror of the source. |

## TCGDex API shapes (captured live, `id` locale)

Base: `https://api.tcgdex.net/v2/{locale}`. No auth/API key.

**`GET /{locale}/series/sv`** — series → its sets (used once, to enumerate all SV sets for the `id` locale):
```json
{
  "id": "SV", "name": "Scarlet & Violet",
  "sets": [
    { "id": "SV1V", "name": "Violet ex", "cardCount": { "official": 78, "total": 78 } },
    "... 23 sets total for the id locale ..."
  ]
}
```

**`GET /{locale}/sets/{setId}`** — one set → its cards (brief form):
```json
{
  "cardCount": { "total": 78, "official": 78, "normal": 64, "holo": 14, "reverse": 0, "firstEd": 0 },
  "cards": [
    { "id": "SV1V-001", "localId": "001", "name": "Pineco", "image": "https://assets.tcgdex.net/id/SV/SV1V/001" }
  ]
}
```
`set.name` here is the Expansion Set's display name. `cardCount` is a useful sanity total to log/compare against ingested row counts, not something to persist.

**`GET /{locale}/cards/{cardId}`** — full card detail, fetched once per locale per card (name differs, `id`/`localId`/`rarity`/`variants`/etc. are locale-invariant so only need reading once, from the primary `id`-locale call):
```json
{
  "id": "SV1V-008", "localId": "008", "name": "Spidops ex",
  "category": "Pokemon", "illustrator": "takuyoa", "rarity": "Double rare",
  "image": "https://assets.tcgdex.net/id/SV/SV1V/008",
  "set": { "id": "SV1V", "name": "Violet ex" },
  "variants": { "firstEdition": false, "holo": true, "normal": false, "reverse": false, "wPromo": false },
  "hp": 260, "types": ["Grass"], "stage": "Stage1", "suffix": "EX",
  "abilities": [{ "type": "Ability", "name": "Trap Territory", "effect": "..." }],
  "attacks": [{ "cost": ["Grass", "Colorless"], "name": "Wire Hang", "effect": "...", "damage": "90+" }],
  "weaknesses": [{ "type": "Fire", "value": "×2" }],
  "retreat": 2, "regulationMark": "G", "dexId": [918],
  "legal": { "standard": false, "expanded": false },
  "updated": "2026-09-16T22:52:11.323Z"
}
```
Only `id`/`ja`/`zh-tw`/`th` locale calls are made (per CONTEXT.md/ticket scope — SV/Indonesian only); a locale that 404s for a given card ID is skipped, not an error (e.g. `en` legitimately has no `SV1V-008`).

## Backend layout (new/changed files)

```
backend/
  internal/
    domain/
      entity/
        game.go              Game{Slug, Name} — crud.BaseEntity
        expansion_set.go     ExpansionSet{GameID, Code, Name, Locale} — crud.BaseEntity
        card.go               Card{ExpansionSetID, LocalID, Names datatypes.JSONMap, Rarity, ImageURL, Attributes datatypes.JSONMap} — crud.BaseEntity
        card_variant.go      CardVariant{CardID, Finish} — crud.BaseEntity
    adapters/
      db/
        postgres/
          migrations/
            20260922000000_catalog_schema.sql   games/expansion_sets/cards/card_variants + FKs + unique indexes + CHECK on finish
      ingestion/
        tcgdex/
          client.go           GetSeries(ctx, locale, seriesID) / GetSet(ctx, locale, setID) / GetCard(ctx, locale, cardID) — net/http + encoding/json, base URL const
          types.go             seriesResponse / setResponse / cardResponse DTOs matching shapes above
          mapper.go             pure funcs: cardResponse+locale-name-map → entity.Card; variants map → []entity.CardVariant — no I/O, unit-testable from fixture JSON
          ingest.go             Ingester{db *gorm.DB}; Run(ctx, locale, seriesID string) — Game find-or-create → per set find-or-create ExpansionSet → per card (bounded concurrency: fetch id-locale detail + ja/zh-tw/th name-only lookups) → upsert Card → upsert CardVariant rows; returns a summary struct (sets/cards/variants counts) for the CLI to print
  cmd/
    ingest-tcgdex/
      main.go                logger.Init, config.Load, otel init (mirrors cmd/job), provider.InitializeProviders(), optional -series/-locale flags (stdlib flag, default "sv"/"id"), tcgdex.NewIngester(ds.Gorm).Run(ctx, ...), logs summary, logger.Fatal on error
  Makefile                    ADD ingest-tcgdex target: go run ./cmd/ingest-tcgdex
  go.mod                       golang.org/x/sync: indirect → direct (errgroup)
```

## Testing (per spec.md's testing decisions)

- **Mapper unit tests** (`mapper_test.go`): table-driven, fed the exact fixture shapes captured above (a card with a single `holo` variant, a hypothetical multi-variant card, a card missing some locale names) — pure functions, no network, no DB. This is where variant-flag→finish-list and locale-map-merging logic actually gets exercised.
- **Client tests** (`client_test.go`): `httptest.Server` serving canned JSON, so CI doesn't depend on TCGDex's real uptime.
- **Upsert idempotency** (repository-level, real local Postgres per spec.md's convention): running the same find-or-create twice yields one row with fields updated on the second pass, not two rows; unique constraints reject a duplicate `(game_id, code)` / `(expansion_set_id, local_id)` / `(card_id, finish)`.
- **No service-layer mock tests** — there's no interface-driven business logic here (capacity limits, aggregation) the way ticket 02's auth flow has; skip per spec.md's own scoping (mock-based service tests are for business logic, not straightforward upserts).
- **Manual verification (the ticket's actual acceptance criterion)**: run `make ingest-tcgdex` against a real (local or Neon branch) Postgres, then confirm via `psql`/Neon MCP that `cards`/`card_variants` are populated and queryable — e.g. `SV1V-008` exists with `rarity = 'Double rare'` and exactly one `card_variants` row (`finish = 'holo'`). This is inherently a run-it-and-look step, not a Go test — same category as how ticket 01's migration itself is verified by running it, not by a dedicated test.

## CI

No changes expected. `backend-ci.yml` already provisions Postgres and runs `go build`/`go vet`/`golangci-lint`/`go test` — the new migration, entities, and mapper/client tests run through that same job. Deliberately **do not** add a CI step that runs `ingest-tcgdex` — that would silently turn "manually triggered only" into "triggered on every CI run," which is exactly what this ticket's last checklist item rules out.

## Execution routing

Per `docs/agents/orchestration.md`: backend-only, but a new subsystem (schema + external API client + new CLI binary) — large enough to route through `backend-agent` in its own worktree rather than directly, same sizing call as ticket 02's backend half.

1. `backend-agent` in its own worktree, in this order: migration → entities → `ingestion/tcgdex` (types → client → mapper, each with its unit tests) → `ingest.go` orchestration + concurrency → `cmd/ingest-tcgdex` → Makefile target → run it for real against a scratch/dev database and record the row counts in the ticket's Comments as the manual verification evidence.
2. Runs its own verification script (`go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./...`), self-reviews via the `code-review` skill, commits on its own branch (`feat(backend): add catalog schema and tcgdex ingestion`).
3. Orchestrator (root, directly): no root-only files expected (no CI changes) — merge the worktree branch back, push after confirmation.
