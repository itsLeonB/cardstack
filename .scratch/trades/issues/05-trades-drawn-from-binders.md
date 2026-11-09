# 05: Trades drawn from binders

**What to build:** A Trade Item may name a binder-style Collection as its source. Executing the Trade empties exactly the Slots holding the offered copies and leaves the rest of the binder untouched, so the binder's quantity stays equal to its Slots after the cards leave.

**Blocked by:** 04.

**Status:** ready-for-agent

- [ ] When the source Collection is a binder, execution empties the Slots holding the offered Card until the offered quantity has left.
- [ ] Copies of that Card beyond the offered quantity keep their Slots and the Collection's Inventory Entry equals the number of Slots still holding it.
- [ ] A binder holding none of the offered Card at execution is the short case: the whole Trade is refused and nothing moves.
- [ ] Pages, Leaves and the rest of the binder's layout are unchanged by an execution.
- [ ] Master Inventory after an execution agrees with what the binder shows.
- [ ] Routes tests cover removing part of a binder's copies and removing all of them, each leaving the binder consistent with its Slots.
