# 05: Catalog browse & search (API + UI)

**What to build:** A user can browse Expansion Sets and the Cards within them — including cards they don't own — and search the catalog by name, set/number, and rarity, with results distinguishing print variants.

**Blocked by:** 03 (Card catalog schema + TCGDex (SV) ingestion)

**Status:** ready-for-agent

- [ ] API endpoint supports searching cards by name, matching any locale name carried by the source data
- [ ] Search supports filtering by Expansion Set/card number and by rarity
- [ ] Search results are per Card Variant, so different print finishes of the same card number are distinguishable
- [ ] Frontend page lists Expansion Sets and lets a user browse all Cards within one, including cards the user doesn't own
- [ ] Frontend page/component lets a user search and filter the catalog per the above
- [ ] Works correctly against SV-only seed data; MA data appears automatically once ticket 04 lands, with no code change required
