# 07: Phase 1 "Scan cards" on a binder

**What to build:** The Phase 1 "Scan cards" flow keeps working on a binder Collection. The user scans single cards into a Draft Addition and reviews it as before, but pressing "Add to Collection" places each added copy in the first empty Slots in Page order, one Slot per copy, instead of calling the bulk quantity update. Copies that find no empty Slot stay in the draft with the reason "not enough empty Slots", the same way a capacity decline stays today, and the Collection's page shows the binder afterwards. The review step shows the binder's empty-Slot count instead of a quantity-based capacity warning.

Spec: `.scratch/binder-scanning/spec.md` (Phase 1 flow on a binder). Phase 1 spec: `.scratch/card-scanning/spec.md`. The placement must keep Inventory Entries equal to the Slot counts (ticket 03).

**Blocked by:** 03 (place, clear, replace and mark Unidentified)

**Status:** ready-for-agent

- [ ] Adding a draft to a binder fills the first empty Slots in Page order, one Slot per copy, and updates the Inventory Entries to match
- [ ] Two Scans of the same Card place two copies in two Slots
- [ ] When there are fewer empty Slots than copies, the placed copies leave the draft and the rest stay with the reason "not enough empty Slots"
- [ ] Applied rows leave the draft, the draft clears only when nothing stayed, and a full clear returns to the Collection's page
- [ ] The review step shows the binder's empty-Slot count and does not show quantity inputs that would change the Slots
- [ ] The Phase 1 flow on a plain Collection is unchanged
- [ ] Covered by HTTP-boundary tests against real Postgres for the placement and frontend feature tests with the API client mocked for the review step
