# 01: Binder foundation

**What to build:** A signed-in user can turn a new or empty Collection into a binder by giving it a Layout, and then read its Pages. The Layout is rows and columns per Page (each 1 to 6) plus a Leaf count (1 to 100, 10 by default). Every Leaf has a front Page and a back Page, and every Page has a grid of empty Slots. Setting a Layout clears the Collection's `max_card_count` to 0, and the binder's capacity is its Slot count. While every Slot is empty the user can change rows and columns, or remove the Layout again. The user can add a Leaf before or after an existing Leaf and remove a Leaf, and the Pages after it are renumbered. Removing a Leaf is refused if either of its Pages holds a non-empty Slot, and the refusal names the Page. A Page can be read as its Slots in grid order, together with the Layout, the Leaf count, the card total and the capacity. The existing bulk quantity update refuses binder Collections with a machine-readable reason. Deleting the Collection deletes its Slots.

Spec: `.scratch/binder-scanning/spec.md` (Layout and binder lifecycle, Slot reads and writes). Vocabulary: Layout, Leaf, Page, Slot in `GLOSSARY.md`.

**Blocked by:** None (can start immediately)

**Status:** resolved

- [x] A Layout can be set on a new or empty Collection and is refused on a Collection that holds any Inventory Entry
- [x] Rows and columns outside 1 to 6, and a Leaf count outside 1 to 100, are rejected
- [x] Setting a Layout sets `max_card_count` to 0, and the Page read shows capacity as Leaves × 2 × rows × columns
- [x] Rows and columns can be edited, and the Layout can be removed, only while every Slot is empty
- [x] Adding a Leaf before or after a given Leaf creates two empty Pages and renumbers the following Pages; adding past 100 Leaves is rejected
- [x] Removing a Leaf removes both Pages and renumbers the following Pages; it is refused, naming the Page, when either Page has a non-empty Slot, and the last Leaf cannot be removed
- [x] Reading a Page returns its Slots in grid order with the Layout, Leaf count, card total and capacity
- [x] The bulk quantity update refuses a binder Collection with a machine-readable reason, and still works on a plain Collection
- [x] Deleting a binder Collection deletes its Slots
- [x] Another user's Collection cannot be read or changed, and an unauthenticated caller is rejected
- [x] All of the above is covered by tests at the HTTP boundary against real Postgres

## Answer

Implemented on `feat/binder-scanning` (backend). Routes: `POST/DELETE /collections/{id}/binder`, `PUT .../binder/layout`, `POST .../binder/leaves` (`{leaf, placement}`), `DELETE .../binder/leaves/{leaf}`, `GET .../binder/pages/{page}`. Leaves are addressed by their 1-based number. `CollectionSummary` carries a nullable `binder` field. Quantity writes (add, update, bulk, remove) on a binder are refused with `binder_quantity_locked` (409), wider than the ticket, to keep ADR-0020's invariant; so Phase 1 confirm on a binder returns 409 until ticket 07. `maxCardCount` is refused on a binder's Collection update. Slots are stored sparsely; ticket 03 and 04 must validate row and column against the Layout. `cardTotal` on the Page read includes Unidentified, unlike `cardCount`.

Verification note: the backend tests need pgvector Postgres; the orchestrator ran one on port 5433 (`DB_PORT=5433`).
