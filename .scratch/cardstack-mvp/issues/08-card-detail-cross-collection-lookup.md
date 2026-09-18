# 08: Card detail: cross-collection lookup

**What to build:** From a card's detail page, a user sees which of their own Collections contain that card, and in what quantity — answering "where is this card physically?" without checking every binder.

**Blocked by:** 05 (Catalog browse & search), 07 (Inventory Entries: record & browse holdings)

**Status:** ready-for-agent

- [ ] A card's detail page/endpoint shows every Collection (belonging to the requesting user) that contains that card, and the quantity in each
- [ ] Only the requesting user's own Collections are considered — no cross-user visibility
- [ ] A card the user owns in zero Collections shows no entries (not an error state)
