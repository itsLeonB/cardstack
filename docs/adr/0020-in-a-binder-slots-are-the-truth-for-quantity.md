# In a binder, Slots are the truth for quantity and Inventory Entries are kept in step

A Collection with a Layout (a binder) records where each physical card sits, as Slots. We chose to make the Slots the truth for quantity in a binder: a Card's quantity is the number of Slots holding it, and Unidentified Slots belong to no Card. Inventory Entries stay real rows, and every Slot write updates the matching entries in the same transaction, creating, changing or removing them so that an entry's quantity always equals the Slot count and an entry with zero Slots does not exist.

Keeping the entries as real rows means Master Inventory (ADR-0002), the entry search, capacity checks and every Phase 1 read keep working unchanged. The existing bulk quantity update refuses a binder Collection with a machine-readable reason, because quantity there can change only by placing or clearing Slots. A Layout can only be set on a Collection that holds no Inventory Entries, and it clears `max_card_count` to 0, so the Slot count is the one capacity.

## Considered Options

- Quantity stays the truth and Slots are an optional annotation on top. Rejected: the two can disagree (three copies but one Slot), and every read would have to decide which to believe.
- Compute quantity from Slots on read and drop Inventory Entries for binders. Rejected: Master Inventory, the entry search and Phase 1 would each need a second code path for binders.
- Allow a Layout on a populated Collection and place existing quantities in Slots automatically. Rejected for now: the placement order is arbitrary and the user would have to rearrange everything. Out of scope in the spec.

## Consequences

Every Slot write path (place, clear, replace, mark Unidentified, move and swap, Replace Page, the Phase 1 placement) must update the entries inside its transaction, and the HTTP-boundary tests assert the invariant after each one. A bug in one path shows up as an entry that disagrees with its Slots. Master Inventory counts binder cards like any other entry, and Unidentified Slots appear in no total.
