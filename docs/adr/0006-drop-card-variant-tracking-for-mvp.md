# Drop Card Variant tracking for MVP; Inventory Entry tracks Card

TCGDex ingestion (ticket 03) is being replaced because its data is outdated; the replacement source (`asia.pokemon-card.com`, chosen over `pokepedia.id`, which is blocked by bot-challenge) doesn't expose print-finish data either, and no other source currently in scope does. Rather than keep a `card_variants`/`finishes` schema no ingestion path can populate, we drop both tables and redefine Inventory Entry/Master Inventory to track quantity against Card directly. Card Variant stays in `CONTEXT.md` as a documented "not modeled in MVP" concept (same treatment as Wishlist), not deleted from the vocabulary — a collector app not distinguishing normal from holo is a real, deliberate MVP gap, not an oversight, and re-adding it later (when a source actually supplies finish data) is additive: a new table plus a foreign key swap on Inventory Entry, not a rename.

## Consequences

ADR-0002 ("Master Inventory is computed... `SUM(quantity) GROUP BY card_variant`") is superseded on this point: the grouping key is now Card, not Card Variant. No code changes as a result — Collection/Inventory Entry aren't implemented yet.
