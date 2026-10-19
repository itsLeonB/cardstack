# 04: Move and swap Slots

**What to build:** A user can reorganise a binder. They tap a filled Slot, then tap a target Slot on the same Page or any other Page of the same binder. If the target is empty the card moves, and if it is filled the two cards swap. An Unidentified Slot can be moved or swapped like any other. The operation is atomic, so a failure leaves both Slots as they were, and Inventory Entries stay equal to the Slot count (a move or swap changes no quantity).

Spec: `.scratch/binder-scanning/spec.md` (Slot reads and writes: move and swap).

**Blocked by:** 03 (place, clear, replace and mark Unidentified)

**Status:** resolved

- [x] Moving a card to an empty Slot on the same Page and on another Page leaves the source empty and the target holding the card
- [x] Moving onto a filled Slot swaps the two Slots, including when one of them is Unidentified
- [x] Moving a Slot onto itself changes nothing
- [x] A move whose source is empty, or whose target is outside the Layout or in another Collection, is rejected and changes nothing
- [x] Every Card's Inventory Entry quantity is unchanged by a move or swap
- [x] The binder view lets the user pick a source, swipe to another Page, and pick the target
- [x] Covered by HTTP-boundary tests against real Postgres and frontend feature tests with the API client mocked

## Answer

Implemented on `feat/binder-scanning` (backend and frontend). The backend move/swap route is in `backend/openapi.json` (grep "move"); it runs in one transaction with the Collection locked and never changes quantities. The Slot panel has a Move action: pick a source, turn to any Page, tap the target; Escape or tapping the source cancels, taps are ignored while a move is in flight, and a pending move resets when the Collection changes or a Leaf is added or removed.
