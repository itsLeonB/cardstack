# Plan: 04 — Unified catalog ingestion via Pokémon Asia scraper

Handoff doc for `backend-agent`. Ticket: `.scratch/cardstack-mvp/issues/04-unified-catalog-ingestion.md`. Blocked by ticket 03 (done). **This ticket adds a new, additive migration** against the schema ticket 03 created — `20260922000000_catalog_schema.sql` is already merged into `main` (PR #7, `263f7c5`), so it is not edited in place; see the "migration is already in `main`" row below and the ticket's own corrected Blocked-by note. Backend-only — no frontend surface (catalog *read* API/UI is ticket 05).

Reference: `docs/adr/0006`, `0007`, `0008`, `0009`. No local reference backend the way ticket 02 had `go-authkit` — this plan is grounded in live requests against `https://asia.pokemon-card.com` made during planning (captured/verified 2026-09-23, including a second pass after a grilling review; the site has no public API or version-pinned client, so verify the HTML shapes below haven't drifted at implementation time).

## Key decisions

| Area | Decision | Why |
|---|---|---|
| Migration is a new additive file, not an in-place edit | `20260922000000_catalog_schema.sql` is already merged into `main`. A new migration, `20260923000000_rework_catalog_schema.sql`, handles the rework (see "Migration shape" below). | The ticket text's own assumption ("never deployed to any persistent DB, so no additive migration is needed") is wrong — merged-into-`main` is the bar for "don't edit in place," not "deployed to a running DB." |
| The rework migration deletes ticket 03's `cards`/`expansion_sets` rows | `DELETE FROM cards; DELETE FROM expansion_sets;` runs before the column changes, before any new `NOT NULL` column (`name`, `category`, `rarity_id`) is added — so no default/backfill value is needed for those. `games`/`locales` rows are kept (the ingester finds the same `pokemon-tcg`/`id` rows by natural key rather than duplicating them). | Since a real Postgres may already have run ticket 03's migration (it's in `main`), this could be real data, not a hypothetical. Ticket 03/04's own framing is that TCGDex's data is outdated and being wholly replaced by this ticket's scrape — column-migrating `cards.names` (JSONB) into `cards.name` (TEXT) would be lossy and pointless busywork for rows about to be superseded anyway; clearing them is more honest than a fake transform, and prevents stale TCGDex rows sitting alongside fresh pokemonasia rows when the ticket's acceptance criterion ("queryable Card records") gets checked. |
| Site is a plain multi-page HTML app, not an SPA | Confirmed: `POST /id/card-search/list/` and `GET /id/card-search/list/?pageNo=N&expansionCodes=...` both return fully server-rendered HTML (no JSON API, no client-side render). Pagination is a GET query string, so the whole flow (enumerate → paginate → detail) can run as plain `net/http` GETs — no session/CSRF token needed. | Matches the ticket's "no headless browser" constraint; confirms it's actually true rather than assumed. |
| Two-phase scrape per Expansion Set: list pages, then detail pages | The results list (`ul.list > li.card > a[href="/id/card-search/detail/{id}/"]`) contains **only** a thumbnail image and a link to a numeric detail-page ID — no name/number/rarity/illustrator/category/tags. All of those fields live only on `GET /id/card-search/detail/{id}/`. | Not what the ticket text implies ("scrapes card-search... extracted and stored") — worth flagging since it roughly doubles the request count (one list-page fetch per ~20 cards, plus one detail fetch per card) versus a single-pass scrape. |
| Enumerate cards per set as 3 regulation-partitioned passes (`regulation=1`, `=2`, `=3`, each `cardType=all`), not one `regulation=all` pass | The form's own default is `regulation=1` ("Standar" only, `checked data-default`) — a naive scrape would silently skip every "Wide"/"Others"-regulation card (most of Sun & Moon and Sword & Shield). Running the 3 buckets separately instead of one `=all` pass costs roughly the *same* total list-page volume (the card count just splits across 3 partitions instead of 1) and gets each card's regulation bucket for free. | Ticket wants "every Indonesian print edition" — a single default-regulation pass would under-ingest three of the four Series almost entirely. The 3-way split is the cheap way to also capture `attributes.regulation` (below) without extra requests. |
| Category derived from detail-page structure, not a stored source field | No card carries an explicit "category" label. Rule (confirmed against Pokémon/Trainer/Energy examples): `h1.pageHeader` has a `span.evolveMarker` (basic/Stage 1/...) → **Pokémon**; else `h3.commonHeader` text is `"Energi Dasar"` or `"Energi Spesial"` → **Energi**; else → **Trainer**, and that same `h3.commonHeader` text (`Item`/`Supporter`/`Stadium`/`Pokémon Tool`) is also the card's Card Tag. Stored category value is the site's own label text (`Pokémon`/`Trainer`/`Energi`), mirroring ADR-0007's "vocabulary is Game-specific." | Matches ADR-0007 exactly (`cardType` is the ticket's single-select radio; its three label values are the closed category vocabulary) while working from data the detail page actually exposes. |
| Card Tag: Trainer/Energy subtype only; Pokémon special markers (ex/V/GX/VMAX/...) go into `attributes`, not Tag | Trainer/Energy subtype is structurally exposed (`h3.commonHeader` text, one value per card) and becomes the Tag. Pokémon's own special markers are **not** separately marked up — `ex`/`V`/`GX`/etc. only ever appear baked into the plain card-name text (e.g. `<h1>...Mega Venusaur ex</h1>`, no wrapping span) — captured into `cards.attributes` via a cheap name-suffix/text heuristic against the known small vocabulary (V/V-UNION/GX/Bercahaya/ex/Evolusi Mega ex, plus the Trainer/Pokémon "Kartu Khusus" markers). | Confirmed live that per-marker query-bucketing (`spPokemon[]=16` narrowed MA1's 184→28 results) *would* give exact data, but costs ~15-20x more list-page requests per set for a field nothing filters on yet — not worth it when the roster is small and a name/text heuristic gets the common cases for free. Revisit (promote to Tag, or switch to query-bucketing) once something actually needs to filter on it. |
| Rarity is a first-class, per-Game lookup table (`rarities`), not `cards.rarity TEXT` | `rarities(id, game_id, code, name)`, unique `(game_id, code)`, `cards.rarity_id` FK — populated by find-or-create during ingestion as `span.alpha` codes are seen (not seeded from the source's own 22-value `rarity[]` filter list, though that list is captured below for reference/grounding). See ADR-0009 for the full rationale (raw codes are illegible without a name; Game-scoped not Series/Set-scoped because the roster *rotates* per print era without any code ever changing meaning). | Raw codes (`SSR`/`BWR`/`MUR`/`FUR`/...) don't "curate" well as bare strings — a lookup table gives every code a readable name once, everywhere it's referenced. |
| `cards.local_id` = the numeric part of `span.collectorNumber` (e.g. `"001/126"` → `"001"`) | Confirmed format via multiple cards. Split on `/`, take the left side; if no `/` is present, store the raw string unchanged (a non-numeric collector number is possible in general — seen on an out-of-scope Taiwan energy card — but not expected within the four in-scope Series). | `(expansion_set_id, local_id)` stays the natural key for idempotent upsert, same as ticket 03. |
| Series/Expansion Set/release_date enumeration: one source, `ul.expansionList` on the paginated plain `GET /id/card-search/?pageNo=N` (~5 pages) | Confirmed: each `li.expansion` carries `span.series` (Series name), `h3.expansionTitle` (Set name), the `expansionCodes=CODE` query param on its own link, and `<time class="relaseDate" datetime="MM-DD-YYYY">` (release date) — all four fields, in one cheap paginated pass. **Supersedes** the `#productSelectorModal`-based enumeration originally planned (ADR-0008's own source): that only has Series name + Set code/name, no release date, and turned out to be the weaker of two sources for the same job. | Originally planned to pull Series/Set from the modal and separately spike a `release_date` source (the modal has none; `/id/products/` was the fallback candidate, requiring per-product archive-page fetches and fragile name-matching). `ul.expansionList` makes that whole spike unnecessary — one source, fewer total requests, no fallback needed. |
| Bounded concurrency + a small per-request delay, not unbounded/serial | 94 Expansion Sets (13 Evolusi Mega + 33 Scarlet & Violet + 33 Pedang & Perisai/Sword & Shield + 15 Matahari & Bulan/Sun & Moon, confirmed via `ul.expansionList`) × (list pages across 3 regulation passes + ~1 detail fetch per card) is roughly 6,000–11,000 HTTP requests (MA1 alone was 184 cards / 10 list pages at `regulation=all`). Unlike ticket 03's TCGDex API (a service built for this), this is a real consumer website. | Bounded worker pool (`golang.org/x/sync/errgroup`, cap ~4-6) plus a small fixed delay between requests. A fully serial run at even 0.5s/request is over an hour; unbounded concurrency risks the run getting IP-blocked mid-way. `golang.org/x/sync` therefore **stays a direct dependency** — ticket's "remove x/sync if nothing else uses it" checkbox is satisfied by this ingester still using it, not by removal. |
| New ingestion package name: `pokemonasia` | `internal/adapters/ingestion/pokemonasia/`, `cmd/ingest-pokemon-asia/` — named after the source, mirroring `tcgdex`'s own naming convention. | Consistency; also makes a future removal of *this* source as clean as removing `tcgdex` was. |
| HTML parsing: stdlib `net/http` + `github.com/PuerkitoBio/goquery` (new dependency) | Ticket explicitly names goquery as the expected choice. Nothing in `go.mod` parses HTML today; stdlib `golang.org/x/net/html` alone is too low-level for the ~5 different selector shapes here (list grid, detail sections, `expansionList`) without hand-rolling a mini jQuery. | Ladder rung 5 (already-installed dependency) doesn't hold — goquery is the ticket's own named example and the standard Go choice for this. |
| `cards.tags` stored as `datatypes.JSONSlice[string]` (JSONB), not a native `text[]` column or a join table | `gorm.io/datatypes` (already a direct dependency, v1.2.7) ships a generic `JSONSlice[T]` Scanner/Valuer — zero new dependencies. A native Postgres `text[]` would need `github.com/lib/pq.StringArray` added as a new dependency purely for this one column. | Ladder rung 5: reuse the dependency already in `go.mod`. JSONB is equally GIN-indexable (`jsonb_path_ops`) for the "filterable" requirement ADR-0007 cites. |
| Upsert strategy: find-by-natural-key then insert/update, for every entity including the new ones | `Game` by `slug`; `Series` by `(game_id, code)` (code = a slugified Series name, per ADR-0008 — locally derived, not source-provided); `Rarity` by `(game_id, code)` (code = the site's own `span.alpha` text); `ExpansionSet` by `(game_id, code)`; `Card` by `(expansion_set_id, local_id)`. | Idempotent re-runs. Reuses the exact `games` row ticket 03's TCGDex ingester created (`slug: "pokemon-tcg"`), per the ticket's own note. |
| CLI wiring | New standalone binary, `provider.InitializeProviders()` for `DataSources`, no wire/`repository_provider.go` changes — mirrors ticket 03 exactly (nothing else needs these repositories yet; ticket 05 wires the read-facing service). | Same reasoning ticket 03 already established. |

## Site shapes (captured live, 2026-09-23, `id` locale)

Base: `https://asia.pokemon-card.com/id`. No auth, no API key, plain GETs (one POST confirmed to behave identically to the equivalent GET query string).

**`GET /card-search/?pageNo={n}`** — the enumeration source for Series, Expansion Set, and release date, all at once:
```html
<ul class="expansionList">
  <li class="expansion">
    <a class="expansionLink" href="/id/card-search/list/?expansionCodes=MA6">
      <div class="rightColumn">
        <div class="seriesBlock"><span class="series">Evolusi Mega</span></div>
        <div class="titleBlock">
          <h3 class="expansionTitle">Booster Pack "30th CELEBRATION"</h3>
          <label class="releaseDateLabel">Tanggal Penjualan</label>
          <time class="relaseDate" datetime="09-16-2026"><span>09-16-2026</span></time>
        </div>
      </div>
    </a>
  </li>
  ...
</ul>
<nav class="pagination">...<a href="/id/card-search/?pageNo=2">2</a>...</nav>
```
`datetime` is `MM-DD-YYYY`. Confirmed ~5 pages, 20/page, covering all four target Series (13 + 33 + 33 + 15 = 94 Expansion Sets) plus any out-of-scope ones (Taiwan/other-locale product lines) to be filtered out by Series name.

**`GET /card-search/list/?expansionCodes={code}&regulation={1|2|3}&cardType=all&pageNo={n}`** — one page of card thumbnails for one Expansion Set/regulation bucket:
```html
<p class="resultNumber">184</p>  <!-- total cards matching, per bucket -->
<p class="resultTotalPages">/ Total 10 halaman</p>
<ul class="list">
  <li class="card">
    <a href="/id/card-search/detail/16488/">
      <div class="imageContainer loading"><img class="lazy" data-original="https://asia.pokemon-card.com/id/card-img/id00016488.png"></div>
    </a>
  </li>
  ...
```
Pagination via plain `<a href="...?pageNo=2&expansionCodes=MA1&regulation=1&cardType=all">`. `pageNo` omitted = page 1.

**`GET /card-search/detail/{id}/`** — one card's full detail (the actual data-bearing page):
```html
<h1 class="pageHeader cardDetail">
  <span class="evolveMarker">Stage 2</span>   <!-- present only for Pokémon; text → cards.attributes.stage -->
  Mega Venusaur ex                             <!-- → cards.name; suffix heuristic ("ex") → cards.attributes.specialMarkers -->
</h1>
...
<div class="skillInformation">
  <h3 class="commonHeader">Serangan</h3>        <!-- Pokémon: literal "Serangan" heading -->
  <!-- Trainer example instead: <h3 class="commonHeader">Item</h3> → category=Trainer, tag="Item" -->
  <!-- Energy example instead: <h3 class="commonHeader">Energi Dasar</h3> → category=Energi, tag="Energi Dasar" -->
</div>
<section class="expansionColumn">
  <span class="alpha">I</span>                  <!-- → find-or-create rarities(game_id, code="I") → cards.rarity_id -->
  <span class="collectorNumber">001/126</span>  <!-- → cards.local_id = "001" -->
</section>
<div class="illustrator">
  <a href="/id/card-search/list/?illustratorName=HYOGONOSUKE">HYOGONOSUKE</a>   <!-- → cards.illustrator -->
</div>
```
Image URL: `https://asia.pokemon-card.com/id/card-img/id{8-digit zero-padded id}.png` (the detail-page id, not `local_id`). `cards.raw` stores the exact response bytes of this GET (per the existing `raw` column convention — see `20260922000000_catalog_schema.sql`'s own comment and the `fix(backend): store cards.raw as exact response bytes` commit).

**Rarity vocabulary** (site's `rarity[]` search-filter checkboxes, captured for reference/grounding only — **not** seeded into the migration, see Key Decisions): `C, U, R, RR, RRR, PR, TR, SR, HR, UR, "Tanpa Tanda" (no mark), K, A, AR, SAR, S, SSR, ACE, BWR, MUR, MA, FUR` (22 total). Confirmed rotation, not collision, across Series: e.g. `MUR` appears on MA1–MA5 but not MA6; `FUR` only starts on MA6; `UR` predates Evolusi Mega entirely. No code has been observed to mean two different things.

`cards.attributes` JSONB continues to hold the Pokémon-only battle mechanics visible on this page when category is Pokémon (HP/type, attacks, weakness/resistance/retreat, evolution stage), plus the two new best-effort fields from this ticket's grilling: `attributes.regulation` (site's own label text — `"Standar"`/`"Luas"`/`"Lainnya"` — from which of the 3 partitioned passes found the card) and `attributes.specialMarkers` (name-suffix heuristic match against the known Pokémon marker vocabulary).

## Backend layout (new/changed files)

```
backend/
  internal/
    domain/
      entity/
        game.go              unchanged
        series.go             NEW: Series{GameID, Code, Name} — crud.BaseEntity
        rarity.go              NEW: Rarity{GameID, Code, Name} — crud.BaseEntity
        expansion_set.go     ExpansionSet gains SeriesID *uuid.UUID, ReleaseDate *time.Time
        card.go                Card: Names(JSONMap) → Name(string); Rarity(string) → RarityID(uuid.UUID);
                                 +Category, +Illustrator (string); +Tags datatypes.JSONSlice[string];
                                 Attributes/ImageURL/Raw unchanged
        card_variant.go      DELETED
        finish.go              DELETED
    adapters/
      db/postgres/migrations/
        20260922000000_catalog_schema.sql   UNCHANGED (already merged into main)
        20260923000000_rework_catalog_schema.sql   NEW — see "Migration shape" below
      ingestion/
        tcgdex/                DELETED (entire directory)
        pokemonasia/            NEW
          client.go              net/http GETs: expansionListPage(ctx, pageNo), resultsPage(ctx, expansionCode,
                                   regulation, pageNo), cardDetail(ctx, id) (doc *goquery.Document, raw []byte, err)
          types.go                expansionListing{Series, Code, Name, ReleaseDate}; cardDetail{Name, Category,
                                   Tag, RarityCode, LocalID, ImageURL, Illustrator, Regulation, Attributes map[string]any}
          mapper.go                pure funcs: parseExpansionListings(doc), parseResultCardIDs(doc),
                                   parseCardDetail(doc) → cardDetail, splitCollectorNumber(s) string,
                                   detectCategory(doc) (category, tag string), detectSpecialMarkers(name string) []string
          ingest.go                Ingester{db *gorm.DB}; Run(ctx) — Game find-or-create → paginate
                                   `/card-search/?pageNo=N` → filter to the 4 target Series → per Series
                                   find-or-create a Series row → per Expansion Set find-or-create (code/name/
                                   release_date now known) → per set: for regulation in {1,2,3}, paginate
                                   `cardType=all` results, collect detail IDs (tagged with that regulation) →
                                   bounded-concurrency (errgroup, cap ~5, small delay) fetch+parse each Card
                                   detail page → find-or-create its Rarity row by code → upsert Card; returns
                                   a summary (series/sets/rarities/cards counts)
  cmd/
    ingest-tcgdex/          DELETED
    ingest-pokemon-asia/
      main.go                  same shape as the deleted cmd/ingest-tcgdex/main.go: logger.Init, config.Load,
                                 otel init, provider.InitializeProviders(), pokemonasia.NewIngester(ds.Gorm).Run(ctx),
                                 logs summary, logger.Fatal on error. No flags needed (ticket scope is fixed:
                                 all four Series, `id` locale) — optionally keep a `-series` debug flag.
  Makefile                    REMOVE ingest-tcgdex target + .PHONY entry + help line;
                                ADD ingest-pokemon-asia target + .PHONY entry + help line
  go.mod                       ADD github.com/PuerkitoBio/goquery (direct); golang.org/x/sync stays direct
                                 (still used, now by pokemonasia instead of tcgdex)
```

## Migration shape (`20260923000000_rework_catalog_schema.sql`, new file)

Up, in order (the `DELETE`s must run before the `NOT NULL` column adds on `cards`, so no default/backfill value is needed for `name`/`category`/`rarity_id`):

```sql
-- +goose Up

-- ticket 03's TCGDex-sourced cards/sets are wholly superseded by this
-- ticket's pokemonasia scraper (TCGDex data was already flagged outdated
-- when this ticket was written); cleared here rather than column-migrated
-- so no stale TCGDex rows survive alongside fresh pokemonasia rows.
-- `games`/`locales` rows are kept — the ingester finds the same
-- `pokemon-tcg`/`id` rows by natural key rather than duplicating them.
DELETE FROM cards;
DELETE FROM expansion_sets;

DROP TABLE card_variants;
DROP TABLE finishes;

CREATE TABLE series (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    game_id UUID NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_series_game_id_code ON series (game_id, code);

-- rarities: per-Game, not seeded — the ingester find-or-creates a row per
-- source rarity code it encounters. See ADR-0009: scoped by Game because
-- the codes are Pokémon-specific (unlike locale/finish, which were shared
-- reference data); not scoped finer than Game because the roster rotates
-- per print era (presence/absence) without any code changing meaning.
CREATE TABLE rarities (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    game_id UUID NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_rarities_game_id_code ON rarities (game_id, code);

ALTER TABLE expansion_sets
    ADD COLUMN series_id UUID REFERENCES series (id) ON DELETE SET NULL,
    ADD COLUMN release_date DATE;
CREATE INDEX idx_expansion_sets_series_id ON expansion_sets (series_id);

ALTER TABLE cards
    DROP COLUMN names,
    DROP COLUMN rarity,
    ADD COLUMN name TEXT NOT NULL,
    ADD COLUMN category TEXT NOT NULL,
    ADD COLUMN illustrator TEXT NOT NULL DEFAULT '',
    ADD COLUMN tags JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN rarity_id UUID NOT NULL REFERENCES rarities (id);
CREATE INDEX idx_cards_category ON cards (category);
CREATE INDEX idx_cards_rarity_id ON cards (rarity_id);
CREATE INDEX idx_cards_tags ON cards USING GIN (tags jsonb_path_ops);

-- +goose Down
ALTER TABLE cards
    DROP COLUMN rarity_id,
    DROP COLUMN tags,
    DROP COLUMN illustrator,
    DROP COLUMN category,
    DROP COLUMN name,
    ADD COLUMN names JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN rarity TEXT NOT NULL DEFAULT '';

ALTER TABLE expansion_sets
    DROP COLUMN release_date,
    DROP COLUMN series_id;

DROP TABLE rarities;
DROP TABLE series;

-- recreate finishes/card_variants exactly as 20260922000000_catalog_schema.sql
-- defined them (omitted here for brevity; copy verbatim from that file).
```

## Testing (per `spec.md`'s testing decisions, same shape as ticket 03)

- **Mapper unit tests** (`mapper_test.go`): table-driven against saved fixture HTML (a Pokémon card, a Trainer card, an Energy card, a card with a non-`/`-delimited collector number, an `expansionList` page fixture) — pure functions, no network, no DB. This is where category-detection, collector-number-splitting, and the special-marker heuristic actually get exercised.
- **Client tests** (`client_test.go`): `httptest.Server` serving the same fixtures, so CI doesn't depend on the live site's uptime or a markup change breaking builds silently.
- **Upsert idempotency** (real local Postgres, per `spec.md`'s convention): re-running find-or-create twice yields one row with fields updated, not two; unique constraints reject duplicate `(game_id, code)` on `series`/`rarities`/`expansion_sets`, duplicate `(expansion_set_id, local_id)` on `cards`. Include a case where a new `rarities` code is find-or-created mid-run (not pre-seeded).
- **No service-layer mock tests** — same reasoning as ticket 03, no business logic here beyond upserts.
- **Manual verification (the ticket's actual acceptance criterion)**: run `make ingest-pokemon-asia` against a real (local or Neon branch) Postgres; confirm via `psql`/Neon MCP that all four Series and their Expansion Sets exist with `release_date` populated, `rarities` has been populated by find-or-create (no seed data), and `cards` spans all four Series (e.g. one card queried from each, with a non-empty `category` and a resolved `rarity_id`). Record row counts in the ticket's Comments as evidence — same as ticket 03. Watch for HTTP 429/503 during the real run; if the site starts throttling, drop concurrency further rather than retry-hammering.

## CI

No changes expected — same reasoning as ticket 03 (`backend-ci.yml` already runs the full build/vet/lint/test suite against the new migration/entities/mapper/client tests). Deliberately do **not** add a CI step running `ingest-pokemon-asia` — same "manually triggered only" reasoning as ticket 03, now doubly true since this hits a real third-party website on every CI run otherwise.

## Execution routing

Per `docs/agents/orchestration.md`: backend-only, but large (schema rework + a full external-site scraper replacing a whole previous ingestion subsystem) — same sizing call as ticket 03, routes through `backend-agent` in its own worktree.

1. `backend-agent`, own worktree, in this order:
   - Remove `internal/adapters/ingestion/tcgdex/`, `cmd/ingest-tcgdex/`, the Makefile target/help line/`.PHONY` entry.
   - Write the new migration `20260923000000_rework_catalog_schema.sql`; update `entity/card.go`, `entity/expansion_set.go`; add `entity/series.go`, `entity/rarity.go`; delete `entity/card_variant.go`, `entity/finish.go`.
   - `go mod tidy` — confirm `golang.org/x/sync` survives (still used) and add `github.com/PuerkitoBio/goquery`.
   - Build `pokemonasia`: `types.go` → `client.go` → `mapper.go` (each with fixture-backed unit tests) → `ingest.go` orchestration + bounded concurrency.
   - `cmd/ingest-pokemon-asia/main.go`, Makefile target.
   - Run it for real against a scratch/dev database; record row counts (per-Series and total, plus how many distinct `rarities` rows got created) in the ticket's Comments as manual-verification evidence.
2. Runs its own verification script (`go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./...`), self-reviews via the `code-review` skill, commits on its own branch (`feat(backend): replace tcgdex ingestion with pokemon-asia scraper`).
3. Orchestrator (root, directly): no root-only files expected — merge the worktree branch back, push after confirmation.
