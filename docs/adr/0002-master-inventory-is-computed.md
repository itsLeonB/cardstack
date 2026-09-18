# Master Inventory is computed, not stored

A user's total holdings across all Collections could be a denormalized table kept in sync on every write, or computed on read from Inventory Entries. We chose computed: `SUM(quantity) GROUP BY card_variant` across a user's Inventory Entries, no separate table. This avoids a whole class of dual-write bugs (an Inventory Entry changes but the summary table doesn't), and the aggregation is cheap at personal-collection scale. Revisit only if read performance actually becomes a measured problem.
