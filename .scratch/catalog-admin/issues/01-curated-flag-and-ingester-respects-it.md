# 01: Curated flag, and the ingester respects it

**What to build:** Cards and Expansion Sets can be marked Curated, and the Asia ingester never overwrites a Curated row. This is the foundation every admin write relies on (see the Curated glossary entry and ADR-0019). Re-running the ingester over a Curated Card or Expansion Set must leave it exactly as it was, while non-Curated cards in the same set are still created and updated.

**Blocked by:** None (can start immediately).

**Status:** resolved

- [ ] An additive migration adds a non-null `curated` flag, default false, to Cards and to Expansion Sets; existing rows stay non-Curated and the running API keeps working while it is applied
- [ ] The ingester skips a Curated Expansion Set's metadata (name, release date, Series) and a Curated Card's fields, and never clears the flag
- [ ] In a set that mixes Curated and non-Curated cards, the ingester still creates and updates the non-Curated ones
- [ ] If the Asia site publishes a set whose code matches a Curated set, the Curated set's own data is kept
- [ ] An ingester test on real Postgres covers all three cases above (existing ingestion-test style)
- [ ] The rollout order (migration with `make job` first, then API, then frontend) is noted in the deployment docs
