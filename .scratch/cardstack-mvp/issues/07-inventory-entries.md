# 07: Inventory Entries: record & browse holdings

**What to build:** An authenticated user can add, update, and remove Card quantities within one of their Collections, and browse a Collection's full contents. Capacity limits (from ticket 06) are enforced at write time.

**Blocked by:** 02 (Auth: registration & login), 05 (Catalog browse & search), 06 (Collections CRUD)

**Status:** done

- [x] Authenticated user can add a Card to one of their Collections with a quantity
- [x] Authenticated user can update the quantity of a Card already in a Collection
- [x] Authenticated user can remove a Card from a Collection entirely
- [x] Authenticated user can view the full contents (Card + quantity pairs) of one of their own Collections
- [x] An add/update that would push a Collection's summed quantity past its capacity limit is rejected, when a limit is set
- [x] Capacity-limit and other business logic is unit-tested against a mocked repository (mockery), independent of the database
- [x] A user cannot view or modify Inventory Entries belonging to another user's Collection
- [x] Frontend: a Collection's page lists its contents (Card + quantity), with an empty state when it holds no Cards
- [x] Frontend: user can add a Card to a Collection with a quantity from the UI
- [x] Frontend: user can edit the quantity of a Card in a Collection from the UI
- [x] Frontend: user can remove a Card from a Collection from the UI
- [x] Frontend: a capacity-limit rejection is shown to the user as an error message and the displayed quantities are left unchanged
