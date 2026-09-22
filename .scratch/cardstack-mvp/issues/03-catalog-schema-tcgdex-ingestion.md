# 03: Card catalog schema + TCGDex (SV) ingestion

**What to build:** The card catalog schema (Game → Expansion Set → Card → Card Variant, generalized across games per ADR-0001, region-scoped identity per the corrected domain glossary) and a manually-triggered CLI command that populates it with Indonesian Scarlet & Violet data from TCGDex.

**Blocked by:** 01 (Project & CI scaffolding)

**Status:** ready-for-agent

**Plan:** `.scratch/cardstack-mvp/plans/03-catalog-schema-tcgdex-ingestion.md`

- [ ] Schema exists for Game, Expansion Set, Card, and Card Variant
- [ ] Expansion Set is unique per (game, set code); Card is unique per (expansion set, local number) — identity is never assumed shared across regions/languages
- [ ] Card Variant models distinct print finishes (normal, reverse holo, holo, first edition, etc.) per Card
- [ ] A CLI command fetches TCGDex's Asia-region SV data and populates the schema for the Indonesian locale
- [ ] Running the CLI command against a real database results in queryable SV Card and Card Variant records
- [ ] The command is manually triggered only — no scheduler or cron job
