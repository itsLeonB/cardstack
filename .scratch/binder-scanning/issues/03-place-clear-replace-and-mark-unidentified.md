# 03: Place, clear, replace and mark Unidentified

**What to build:** A user can edit a binder one Slot at a time. Tapping an empty Slot opens a catalog search and places the chosen Card. Tapping a filled Slot lets them clear it, replace its Card, or mark it Unidentified, and an Unidentified Slot can be given a Card. In a binder the quantity of a Card is the number of Slots holding it, and every Slot write updates the matching Inventory Entry in the same transaction, creating, changing or removing it so that it always equals the Slot count. Unidentified Slots belong to no Card and no Inventory Entry. Master Inventory counts binder cards like any other entry. The quantity inputs on a binder are replaced by these Slot actions.

Spec: `.scratch/binder-scanning/spec.md` (Quantity invariant, Slot reads and writes). ADR-0002 (Master Inventory is computed) stays true. This ticket carries the quantity-invariant tests.

**Blocked by:** 01 (binder foundation), 02 (binder view)

**Status:** ready-for-agent

- [ ] Placing a Card in an empty Slot, replacing, clearing, and marking Unidentified each work through the API and the binder view
- [ ] After every write, each Card's Inventory Entry quantity equals the number of its Slots in that Collection, and a Card with no Slots has no entry
- [ ] Unidentified Slots appear in no Inventory Entry and no Master Inventory total, and count toward the binder's card total
- [ ] Placing a Card the server does not know is rejected, and nothing changes
- [ ] A write on another user's binder is rejected
- [ ] Master Inventory shows the quantities of Cards placed in a binder
- [ ] The binder shows Slot actions in place of quantity inputs
- [ ] Covered by HTTP-boundary tests against real Postgres (including the invariant after each operation) and frontend feature tests with the API client mocked
