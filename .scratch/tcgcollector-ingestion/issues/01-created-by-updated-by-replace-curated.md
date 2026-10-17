# 01: created_by and updated_by replace curated

**What to build:** Every Card and Expansion Set records which Source created it and which Source last updated it, and a writer changes a row only while it is the one that last updated it (ADR-0021, GLOSSARY **Source**). The Pokémon Asia ingester keeps updating its own rows on rerun and leaves rows an Admin edited alone, as it does today for curated rows. An Admin write makes the Admin the last updater. The `curated` boolean goes away.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] A migration adds `created_by` and `updated_by` text columns (values `pokemon-asia`, `tcgcollector`, `admin`) to `cards` and `expansion_sets`, backfills every row to `pokemon-asia` in both, maps rows with `curated = true` to `admin` in both, then drops `curated`; a column comment says they hold a Source, not a user
- [ ] The Asia ingester inserts with both columns `pokemon-asia`, updates a Card or Expansion Set only while `updated_by` is `pokemon-asia`, and leaves any other row untouched
- [ ] Admin create and edit of an Expansion Set or Card set `updated_by = admin` (and `created_by = admin` on create), and leave `created_by` alone on edit; the admin behaviour is otherwise unchanged
- [ ] Asia ingester run tests show a row last updated by `admin` or `tcgcollector` is not changed by a rerun and that its own rows still update
- [ ] The existing admin service tests pass with the new columns, and a migration test or check shows an existing curated row ends up `admin`
- [ ] Wherever `curated` was read (ingester, admin responses, images step) the new rule replaces it; no reference to the old column remains
- [ ] Backend verification script passes
