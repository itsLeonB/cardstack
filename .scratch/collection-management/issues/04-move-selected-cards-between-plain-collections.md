# 04: Move selected cards between plain Collections

**What to build:** a user ticks cards in a Collection, chooses a destination and confirms, and those cards, with every copy of each, move to the destination (GLOSSARY.md: Move). This ticket covers a plain Collection as the source and as the destination, and the refusals around it.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] A Move operation takes an explicit list of Card ids, a source and a destination Collection, both owned by the caller and different from each other. It moves whole Inventory Entries, every copy of each. A Card the destination already holds has its quantity added to the existing entry.
- [x] A Move that would pass the destination's maximum card-count limit is refused whole, before anything changes, and the error says what the limit is and by how many copies it would be exceeded.
- [x] A binder as the destination is refused with a machine-readable reason, since no Slot was chosen. The response says how many cards and copies moved.
- [x] A Select button turns on selection mode, with a checkbox on every card and a sticky action bar showing "N selected", "Move to…" and "Cancel", clear of the mobile safe area. Leaving selection mode clears the selection.
- [x] "Move to…" opens a sheet of the user's Collections with the source and every binder disabled and the reason shown. A confirm dialog then states the number of cards and copies and the two Collection names. There is no undo.
- [x] HTTP-boundary tests cover the move, the merge, the capacity refusal that changes nothing, the binder destination refusal, and the same-Collection and wrong-owner refusals. Screen tests cover selection mode, the sheet and the confirm dialog.
