# 01: Binder foundation

**What to build:** A signed-in user can turn a new or empty Collection into a binder by giving it a Layout, and then read its Pages. The Layout is rows and columns per Page (each 1 to 6) plus a Leaf count (1 to 100, 10 by default). Every Leaf has a front Page and a back Page, and every Page has a grid of empty Slots. Setting a Layout clears the Collection's `max_card_count` to 0, and the binder's capacity is its Slot count. While every Slot is empty the user can change rows and columns, or remove the Layout again. The user can add a Leaf before or after an existing Leaf and remove a Leaf, and the Pages after it are renumbered. Removing a Leaf is refused if either of its Pages holds a non-empty Slot, and the refusal names the Page. A Page can be read as its Slots in grid order, together with the Layout, the Leaf count, the card total and the capacity. The existing bulk quantity update refuses binder Collections with a machine-readable reason. Deleting the Collection deletes its Slots.

Spec: `.scratch/binder-scanning/spec.md` (Layout and binder lifecycle, Slot reads and writes). Vocabulary: Layout, Leaf, Page, Slot in `GLOSSARY.md`.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] A Layout can be set on a new or empty Collection and is refused on a Collection that holds any Inventory Entry
- [ ] Rows and columns outside 1 to 6, and a Leaf count outside 1 to 100, are rejected
- [ ] Setting a Layout sets `max_card_count` to 0, and the Page read shows capacity as Leaves × 2 × rows × columns
- [ ] Rows and columns can be edited, and the Layout can be removed, only while every Slot is empty
- [ ] Adding a Leaf before or after a given Leaf creates two empty Pages and renumbers the following Pages; adding past 100 Leaves is rejected
- [ ] Removing a Leaf removes both Pages and renumbers the following Pages; it is refused, naming the Page, when either Page has a non-empty Slot, and the last Leaf cannot be removed
- [ ] Reading a Page returns its Slots in grid order with the Layout, Leaf count, card total and capacity
- [ ] The bulk quantity update refuses a binder Collection with a machine-readable reason, and still works on a plain Collection
- [ ] Deleting a binder Collection deletes its Slots
- [ ] Another user's Collection cannot be read or changed, and an unauthenticated caller is rejected
- [ ] All of the above is covered by tests at the HTTP boundary against real Postgres
