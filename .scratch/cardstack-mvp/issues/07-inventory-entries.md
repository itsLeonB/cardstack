# 07: Inventory Entries: record & browse holdings

**What to build:** An authenticated user can add, update, and remove Card quantities within one of their Collections, and browse a Collection's full contents. Capacity limits (from ticket 06) are enforced at write time.

**Blocked by:** 02 (Auth: registration & login), 05 (Catalog browse & search), 06 (Collections CRUD)

**Status:** ready-for-agent

- [ ] Authenticated user can add a Card to one of their Collections with a quantity
- [ ] Authenticated user can update the quantity of a Card already in a Collection
- [ ] Authenticated user can remove a Card from a Collection entirely
- [ ] Authenticated user can view the full contents (Card + quantity pairs) of one of their own Collections
- [ ] An add/update that would push a Collection's summed quantity past its capacity limit is rejected, when a limit is set
- [ ] Capacity-limit and other business logic is unit-tested against a mocked repository (mockery), independent of the database
- [ ] A user cannot view or modify Inventory Entries belonging to another user's Collection
