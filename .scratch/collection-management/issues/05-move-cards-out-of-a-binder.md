# 05: Move cards out of a binder

**What to build:** a binder-style Collection can be the source of a Move. The Slots holding the moved Cards become empty in the same transaction, so the binder still matches what it physically holds (ADR-0020).

**Blocked by:** 04 (Move selected cards between plain Collections).

**Status:** ready-for-agent

- [ ] Moving Cards out of a binder moves their whole entries to a plain Collection and empties every Slot that held them, through the one shared Slot-write path ADR-0020 requires, so each remaining entry's quantity still equals its Slot count and an entry with no Slots does not exist.
- [ ] Unidentified Slots are untouched, and the move is all-or-nothing in one transaction together with the capacity check.
- [ ] The selection mode and the Move to sheet work on a binder's card list, and the confirm dialog says that the Slots will be emptied.
- [ ] HTTP-boundary tests assert the Slot and entry invariant after a move out of a binder, that Unidentified Slots remain, and the refusal case. A screen test covers the confirm wording.
