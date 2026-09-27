# 04: Unified catalog ingestion via Pokémon Asia scraper

**What to build:** A manually-triggered CLI command that scrapes `asia.pokemon-card.com/id/card-search` (plain server-rendered HTML, reachable via predictable GET/POST query params — no headless browser needed) and populates the reworked catalog schema (`Game` → `Series` → `Expansion Set` → `Card`, no `Card Variant`) with every Indonesian print edition across all four Series: Scarlet & Violet, Sword & Shield, Sun & Moon, Mega Evolution. This ticket removes ticket 03's TCGDex ingestion entirely (its data was outdated) and supersedes this ticket's original MA-only/headless-browser scope — see `docs/adr/0006-drop-card-variant-tracking-for-mvp.md`, `docs/adr/0007-card-category-and-tag-are-first-class.md`, `docs/adr/0008-series-groups-expansion-sets-optionally.md`.

**Blocked by:** 03 (Card catalog schema + TCGDex (SV) ingestion) — done; this ticket adds a new migration against the schema ticket 03 created. Originally planned as an in-place edit of `20260922000000_catalog_schema.sql` on the assumption it was never deployed anywhere — corrected during planning (see Comments): that migration is already merged into `main` (PR #7), so it is **not** edited in place; a new additive migration handles the schema rework instead.

**Status:** done — implemented (commit `0981352`), merged into `feat/pokemon-asia-scraping`, not yet merged to `main`.

- [x] TCGDex ingestion code is fully removed: `internal/adapters/ingestion/tcgdex/`, `cmd/ingest-tcgdex/`, its Makefile target, and the `golang.org/x/sync` direct dependency if nothing else uses it (survives — still used by the new ingester's `errgroup`)
- [x] `card_variants` and `finishes` tables are dropped (via new additive migration `20260923000000_rework_catalog_schema.sql`, not an in-place edit — see Blocked-by note and Comments)
- [x] `cards.names` (JSONB) becomes `cards.name` (TEXT)
- [x] `series` table added (`game_id` FK, locally-derived `code`, `name`); `expansion_sets.series_id` is a nullable FK
- [x] `expansion_sets.release_date` column added
- [x] `cards.category` (TEXT, exactly one per Card) added
- [x] `cards.rarity` (TEXT) becomes a first-class `rarities` lookup table (`id`, `game_id`, `code`, `name`), unique on `(game_id, code)`, with `cards.rarity_id` FK — populated by find-or-create during ingestion, not seeded (ADR-0009)
- [x] A Card Tag mechanism (multi-valued per Card — e.g. an array column or join table) added
- [x] `cards.illustrator` (TEXT) added
- [x] Pokémon-specific battle stats (HP, energy type, weakness, resistance, retreat cost, evolution stage, format legality) stay in the existing `cards.attributes` JSONB — not promoted to fixed columns
- [x] A CLI command scrapes `asia.pokemon-card.com/id/card-search` using stdlib `net/http` + an HTML-parsing library (e.g. `goquery`) — no headless browser dependency
- [x] Scraping covers all four Series and every Expansion Set listed under each, not a single hardcoded one
- [x] Card name, number, rarity, image, illustrator, category, tags, and Expansion Set release date are extracted and stored per the reworked schema
- [x] Running the CLI command against a real database results in queryable Card records spanning all four Series
- [x] The command is manually triggered only — no scheduler or cron job

## Comments

Planning/grilling session, 2026-09-23 — full plan at `.scratch/cardstack-mvp/plans/04-unified-catalog-ingestion.md`. Key corrections and decisions made against the ticket text above:

- **Migration**: `20260922000000_catalog_schema.sql` (ticket 03) is already merged into `main` — it is a new additive migration, not an in-place edit (see Blocked-by note above).
- **Series/Expansion Set/release date**: all sourced from one place, `GET /id/card-search/?pageNo=N` (`ul.expansionList`), which carries Series name + Set title + `expansionCode` + release date together — not the `#productSelectorModal` originally assumed (that only has Series/Set name/code, no date).
- **Rarity**: promoted to a first-class `rarities` lookup table (`(game_id, code)`, find-or-create, not seeded) rather than a plain `cards.rarity TEXT` column — raw source codes (`SSR`/`BWR`/`MUR`/...) are illegible without a name attached. See ADR-0009 and the new checklist item above.
- **Regulation**: captured into `cards.attributes.regulation` (site's own label text) via 3 partitioned scrape passes (`regulation=1/2/3`) per Expansion Set instead of 1 (`=all`) — same total request volume, gets the field for free. Not filterable/indexed for now.
- **Pokémon special markers** (`ex`/`V`/`GX`/`VMAX`/...): not modeled as Card Tag — the source only bakes them into the card's display name, no structured field. Captured best-effort into `cards.attributes` via a cheap name-suffix heuristic; exact extraction (per-marker query-bucketing) was rejected as ~15-20x the list-page request volume for a field nothing filters on yet.
- `CONTEXT.md` and `docs/adr/0009-...md` updated accordingly; `spec.md`'s Domain shape bullet and ADR count updated.

Implementation, 2026-09-23 — `backend-agent` in worktree, commit `0981352` (`feat(backend): replace tcgdex ingestion with pokemon-asia scraper`), merged into `feat/pokemon-asia-scraping`. `go build/vet/gofmt/test ./...` clean; mapper/client/idempotency tests pass against real local Postgres; architecture-level Standards + Spec review (two-axis, scoped to the full commit diff) found no hard issues.

Manual verification (real run of the built `ingest-pokemon-asia` binary against local Postgres, `cardstack-test-pg`, ~2.5 hours, ~17k HTTP requests, no 429/503):

- 4 series, 94 expansion sets (13 Evolusi Mega + 33 Scarlet & Violet + 33 Pedang & Perisai + 15 Matahari & Bulan) — matches the plan's live-captured counts exactly, all with `release_date` populated.
- 10 `rarities` rows, all created by find-or-create (no seed data).
- 12,119 distinct `cards` spanning all four Series, each with a non-empty `category` and resolved `rarity_id`.

One real-world deviation from the plan found and fixed during this run: the site's `regulation=1/2/3` query param doesn't cleanly partition results as assumed (a set's `regulation=1` and `=2` passes can return the identical full card list) — fixed by deduping card IDs across the 3 passes (lowest-numbered bucket wins) before dispatch, with a regression test. No data was lost by this (idempotent upsert still caught every card) — it only affected some `attributes.regulation` values from this specific run being unreliable; `regulation` is explicitly a best-effort, non-required attribute (not in the extraction list above), so a second full re-run was judged not worth the ~2.5h cost.
