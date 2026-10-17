# 02: Unpublished Expansion Sets are hidden from users and from scanning

**What to build:** An Expansion Set can be unpublished. A caller who is not an Admin never sees an unpublished set or any of its Cards: not in browsing, search, filters, facet counts, card detail, inventory and collection pickers, or as a scan match candidate. Admin routes still see everything. Existing sets stay published and new sets default to published; only the TCG Collector ingester will create unpublished ones, and the operator publishes by SQL (GLOSSARY **Published**).

**Blocked by:** None (can start immediately).

**Status:** resolved

- [ ] A migration adds `expansion_sets.published`, true for every existing row and true by default
- [ ] Series and Expansion Set listings, name search, rarity, category and tag filters, facet counts, pagination totals and card detail exclude unpublished sets and their Cards for a Guest and for a signed-in non-Admin
- [ ] Inventory and collection pickers and any Master Inventory view built from the catalog never show an unpublished Card
- [ ] The scan matcher never returns a Card from an unpublished set
- [ ] Admin routes list and edit unpublished sets and their Cards
- [ ] A Series whose sets are all unpublished does not appear in the Series list, and its Card count excludes them
- [ ] Catalog handler tests cover an unpublished set against Guest, non-Admin and Admin callers across listing, search, filters, counts and detail; a matcher test covers the scan exclusion
- [ ] Backend verification script passes
