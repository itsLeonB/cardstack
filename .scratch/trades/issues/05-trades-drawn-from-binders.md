# 05: Trades drawn from binders

**What to build:** A Trade Item may name a binder-style Collection as its source. Executing the Trade empties exactly the Slots holding the offered copies and leaves the rest of the binder untouched, so the binder's quantity stays equal to its Slots after the cards leave.

**Blocked by:** 04.

**Status:** done

- [x] When the source Collection is a binder, execution empties the Slots holding the offered Card until the offered quantity has left.
- [x] Copies of that Card beyond the offered quantity keep their Slots and the Collection's Inventory Entry equals the number of Slots still holding it.
- [x] A binder holding none of the offered Card at execution is the short case: the whole Trade is refused and nothing moves.
- [x] Pages, Leaves and the rest of the binder's layout are unchanged by an execution.
- [x] Master Inventory after an execution agrees with what the binder shows.
- [x] Routes tests cover removing part of a binder's copies and removing all of them, each leaving the binder consistent with its Slots.

## Comments

Completed with Pi GPT-6.1 Terra on `feat/trades-drawn-from-binders`. Test commits: `99f1be41`, `315c5414`. Ticket 04 already supplied the production behavior, so this change adds API route tests only. The new tests passed without a production fix.

Coverage: partial removal, full removal, unchanged layout and unrelated Slots, Slot/Inventory Entry counts, Master Inventory, Trade Inboxes, whole-Trade refusal and source-first fallback. The empty-binder short case has no fallback holdings; an empty binder with enough cards elsewhere still succeeds, as ticket 04 and the full Trades spec require.

Verification: focused route tests passed. `scripts/verification/backend.sh` passed build, vet, format, lint and race tests in the worker and again on the merged feature branch. Worker review found two test gaps (fallback and Slot counts), both fixed. Final integration review: Standards 0 findings; Spec 0 findings. No frontend changes, push or deployment.
