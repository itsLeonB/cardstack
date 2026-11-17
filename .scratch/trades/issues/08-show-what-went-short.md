# 08: Show what went short

**What to build:** A locked Trade can be agreed and then go stale, because nothing reserves the offered quantities while the two people arrange to meet. Opening the room re-checks every Trade Item against what each participant currently holds and marks the ones that no longer match, so the mismatch is visible before the meeting instead of at the confirmation.

**Blocked by:** 05.

**Status:** done

- [x] Reading a Trade re-checks every Trade Item against the current quantities the offering participant holds.
- [x] A Trade Item that can no longer be satisfied is marked short, naming the Card and the offered quantity beside what is actually available.
- [x] The check changes no state and no quantity.
- [x] The marker appears on the same read that a participant uses before confirming, alongside the state and both sides' Trade Items.
- [x] Execution still refuses the whole Trade when a marked Trade Item is confirmed, with the existing short-item failure.
- [x] Routes tests cover a Trade that goes short after locking, and that the read leaves quantities and state untouched. The Trade page shows the marker on the affected Trade Item.
