# 04: Unified catalog ingestion via Pokémon Asia scraper

**What to build:** A manually-triggered CLI command that scrapes `asia.pokemon-card.com/id/card-search` (plain server-rendered HTML, reachable via predictable GET/POST query params — no headless browser needed) and populates the reworked catalog schema (`Game` → `Series` → `Expansion Set` → `Card`, no `Card Variant`) with every Indonesian print edition across all four Series: Scarlet & Violet, Sword & Shield, Sun & Moon, Mega Evolution. This ticket removes ticket 03's TCGDex ingestion entirely (its data was outdated) and supersedes this ticket's original MA-only/headless-browser scope — see `docs/adr/0006-drop-card-variant-tracking-for-mvp.md`, `docs/adr/0007-card-category-and-tag-are-first-class.md`, `docs/adr/0008-series-groups-expansion-sets-optionally.md`.

**Blocked by:** 03 (Card catalog schema + TCGDex (SV) ingestion) — done; this ticket adds a new migration against the schema ticket 03 created. Originally planned as an in-place edit of `20260922000000_catalog_schema.sql` on the assumption it was never deployed anywhere — corrected during planning (see Comments): that migration is already merged into `main` (PR #7), so it is **not** edited in place; a new additive migration handles the schema rework instead.

**Status:** ready-for-agent

- [ ] TCGDex ingestion code is fully removed: `internal/adapters/ingestion/tcgdex/`, `cmd/ingest-tcgdex/`, its Makefile target, and the `golang.org/x/sync` direct dependency if nothing else uses it
- [ ] `card_variants` and `finishes` tables are dropped (migration edited in place, not a new migration)
- [ ] `cards.names` (JSONB) becomes `cards.name` (TEXT)
- [ ] `series` table added (`game_id` FK, locally-derived `code`, `name`); `expansion_sets.series_id` is a nullable FK
- [ ] `expansion_sets.release_date` column added
- [ ] `cards.category` (TEXT, exactly one per Card) added
- [ ] `cards.rarity` (TEXT) becomes a first-class `rarities` lookup table (`id`, `game_id`, `code`, `name`), unique on `(game_id, code)`, with `cards.rarity_id` FK — populated by find-or-create during ingestion, not seeded (ADR-0009)
- [ ] A Card Tag mechanism (multi-valued per Card — e.g. an array column or join table) added
- [ ] `cards.illustrator` (TEXT) added
- [ ] Pokémon-specific battle stats (HP, energy type, weakness, resistance, retreat cost, evolution stage, format legality) stay in the existing `cards.attributes` JSONB — not promoted to fixed columns
- [ ] A CLI command scrapes `asia.pokemon-card.com/id/card-search` using stdlib `net/http` + an HTML-parsing library (e.g. `goquery`) — no headless browser dependency
- [ ] Scraping covers all four Series and every Expansion Set listed under each, not a single hardcoded one
- [ ] Card name, number, rarity, image, illustrator, category, tags, and Expansion Set release date are extracted and stored per the reworked schema
- [ ] Running the CLI command against a real database results in queryable Card records spanning all four Series
- [ ] The command is manually triggered only — no scheduler or cron job

## Comments

Planning/grilling session, 2026-09-23 — full plan at `.scratch/cardstack-mvp/plans/04-unified-catalog-ingestion.md`. Key corrections and decisions made against the ticket text above:

- **Migration**: `20260922000000_catalog_schema.sql` (ticket 03) is already merged into `main` — it is a new additive migration, not an in-place edit (see Blocked-by note above).
- **Series/Expansion Set/release date**: all sourced from one place, `GET /id/card-search/?pageNo=N` (`ul.expansionList`), which carries Series name + Set title + `expansionCode` + release date together — not the `#productSelectorModal` originally assumed (that only has Series/Set name/code, no date).
- **Rarity**: promoted to a first-class `rarities` lookup table (`(game_id, code)`, find-or-create, not seeded) rather than a plain `cards.rarity TEXT` column — raw source codes (`SSR`/`BWR`/`MUR`/...) are illegible without a name attached. See ADR-0009 and the new checklist item above.
- **Regulation**: captured into `cards.attributes.regulation` (site's own label text) via 3 partitioned scrape passes (`regulation=1/2/3`) per Expansion Set instead of 1 (`=all`) — same total request volume, gets the field for free. Not filterable/indexed for now.
- **Pokémon special markers** (`ex`/`V`/`GX`/`VMAX`/...): not modeled as Card Tag — the source only bakes them into the card's display name, no structured field. Captured best-effort into `cards.attributes` via a cheap name-suffix heuristic; exact extraction (per-marker query-bucketing) was rejected as ~15-20x the list-page request volume for a field nothing filters on yet.
- `CONTEXT.md` and `docs/adr/0009-...md` updated accordingly; `spec.md`'s Domain shape bullet and ADR count updated.
