# 08: Find a card in a binder

**What to build:** The Collection's entry list for a binder shows, for each Card, where it sits: every position as Page, row and column ("Page 4 · row 2 · col 3"). Tapping a position opens the binder view on that Page with the Slot highlighted. The lookup is within one binder; searching across all of a user's Collections is not part of this ticket.

Spec: `.scratch/binder-scanning/spec.md` (Slot reads and writes: entry list positions, Binder view).

**Blocked by:** 03 (place, clear, replace and mark Unidentified)

**Status:** resolved

- [x] The Collection entries response for a binder includes each Card's Slot positions (Page, row, column), and a Card with two copies lists two positions
- [x] Unidentified Slots appear in no entry's positions
- [x] The entry list shows the positions, and tapping one opens the binder on that Page with the Slot highlighted
- [x] Positions follow Page renumbering after a Leaf is added or removed
- [x] The entry list of a plain Collection is unchanged
- [x] Covered by HTTP-boundary tests against real Postgres and frontend feature tests with the API client mocked

## Answer

Implemented on `feat/binder-scanning` (backend and frontend). The Collection entries response carries `positions` (Page, row, column) per binder entry, one per copy, excluding Unidentified Slots and computed on read so they follow Leaf renumbering. The entry list shows "Page N · row R · col C"; tapping one opens the binder on that Page with the Slot highlighted (the highlight clears on a Leaf add or remove).
