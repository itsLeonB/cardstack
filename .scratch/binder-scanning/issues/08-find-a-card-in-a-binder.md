# 08: Find a card in a binder

**What to build:** The Collection's entry list for a binder shows, for each Card, where it sits: every position as Page, row and column ("Page 4 · row 2 · col 3"). Tapping a position opens the binder view on that Page with the Slot highlighted. The lookup is within one binder; searching across all of a user's Collections is not part of this ticket.

Spec: `.scratch/binder-scanning/spec.md` (Slot reads and writes: entry list positions, Binder view).

**Blocked by:** 03 (place, clear, replace and mark Unidentified)

**Status:** ready-for-agent

- [ ] The Collection entries response for a binder includes each Card's Slot positions (Page, row, column), and a Card with two copies lists two positions
- [ ] Unidentified Slots appear in no entry's positions
- [ ] The entry list shows the positions, and tapping one opens the binder on that Page with the Slot highlighted
- [ ] Positions follow Page renumbering after a Leaf is added or removed
- [ ] The entry list of a plain Collection is unchanged
- [ ] Covered by HTTP-boundary tests against real Postgres and frontend feature tests with the API client mocked
