# 04: Unified catalog ingestion via Pokémon Asia scraper

**What to build:** A manually-triggered CLI command that scrapes `asia.pokemon-card.com/id/card-search` (plain server-rendered HTML, reachable via predictable GET/POST query params — no headless browser needed) and populates the reworked catalog schema (`Game` → `Series` → `Expansion Set` → `Card`, no `Card Variant`) with every Indonesian print edition across all four Series: Scarlet & Violet, Sword & Shield, Sun & Moon, Mega Evolution. This ticket removes ticket 03's TCGDex ingestion entirely (its data was outdated) and supersedes this ticket's original MA-only/headless-browser scope — see `docs/adr/0006-drop-card-variant-tracking-for-mvp.md`, `docs/adr/0007-card-category-and-tag-are-first-class.md`, `docs/adr/0008-series-groups-expansion-sets-optionally.md`.

**Blocked by:** 03 (Card catalog schema + TCGDex (SV) ingestion) — done; this ticket edits the same migration in place (never deployed to any persistent DB, so no additive migration is needed)

**Status:** ready-for-agent

- [ ] TCGDex ingestion code is fully removed: `internal/adapters/ingestion/tcgdex/`, `cmd/ingest-tcgdex/`, its Makefile target, and the `golang.org/x/sync` direct dependency if nothing else uses it
- [ ] `card_variants` and `finishes` tables are dropped (migration edited in place, not a new migration)
- [ ] `cards.names` (JSONB) becomes `cards.name` (TEXT)
- [ ] `series` table added (`game_id` FK, locally-derived `code`, `name`); `expansion_sets.series_id` is a nullable FK
- [ ] `expansion_sets.release_date` column added
- [ ] `cards.category` (TEXT, exactly one per Card) added
- [ ] A Card Tag mechanism (multi-valued per Card — e.g. an array column or join table) added
- [ ] `cards.illustrator` (TEXT) added
- [ ] Pokémon-specific battle stats (HP, energy type, weakness, resistance, retreat cost, evolution stage, format legality) stay in the existing `cards.attributes` JSONB — not promoted to fixed columns
- [ ] A CLI command scrapes `asia.pokemon-card.com/id/card-search` using stdlib `net/http` + an HTML-parsing library (e.g. `goquery`) — no headless browser dependency
- [ ] Scraping covers all four Series and every Expansion Set listed under each, not a single hardcoded one
- [ ] Card name, number, rarity, image, illustrator, category, tags, and Expansion Set release date are extracted and stored per the reworked schema
- [ ] Running the CLI command against a real database results in queryable Card records spanning all four Series
- [ ] The command is manually triggered only — no scheduler or cron job
