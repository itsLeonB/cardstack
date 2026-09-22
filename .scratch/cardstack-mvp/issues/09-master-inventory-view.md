# 09: Master Inventory view

**What to build:** A user sees their Master Inventory — total quantity owned per Card, aggregated across every one of their Collections — computed on read, never stored (ADR-0002, as amended by ADR-0006).

**Blocked by:** 07 (Inventory Entries: record & browse holdings)

**Status:** ready-for-agent

- [ ] Endpoint/page shows, per Card, the total quantity owned across all of the user's Collections combined
- [ ] The total is computed on read from Inventory Entries — no separate stored/denormalized total
- [ ] The Master Inventory reflects a Collection change immediately, with no manual refresh/sync step required
- [ ] Only the requesting user's own Inventory Entries are aggregated
