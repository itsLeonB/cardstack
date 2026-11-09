# Trades expose only their offered items, and received cards land in a flagged Collection

A trade between two users shows each side only what the other put into it: Cardstack gets no per-user public inventory, and the Shared link of a Collection stays the only way a user shows their cards to someone. What a side receives lands in a Trade Inbox — an ordinary Collection carrying a flag, hidden from the owner's own Collections and shown on its own page — rather than in a staging table outside Collections, so that quantity keeps living in an Inventory Entry everywhere in the app.

## Considered Options

- A per-user public inventory flag, exposing the whole Master Inventory with reciprocal publication required before asking for a trade. Rejected: Master Inventory is computed across every Collection, so the flag would publish the binders and decks its owner never trades from, with nothing excluded; and its one advantage, mutual visibility, is already delivered by Trade Items. It also forces the asking user to publish everything in order to ask once.
- A per-card "for trade" mark. Rejected: quantity spans Collections, so the mark would need a table of its own and would still not say which Collection an offered copy leaves.
- A staging table for received cards. Rejected: staged cards become a second kind of quantity, absent from Master Inventory, Wishlist fulfilment and Collection counts unless every read is taught about them, and filing them needs a second implementation of Move. The flag keeps the one invariant instead.
- Reserving offered quantities while a Trade is locked. Rejected for MVP: a held-quantity concept on Inventory Entries reaching into Move, binder Slots and Wishlist Presets, to prevent a case the room already shows before anyone meets — it marks an item short as soon as it is opened. Execution refuses the whole Trade when an item is short, the same rule Move uses.

## Consequences

- There is no delivery mechanism: the API has no email, push or inbox, so a Trade is poll-only and the Trades list is the only way back to one.
- A Trade deals in a Card and a quantity, never a specific copy: Card Variant is not modelled (ADR-0006), so condition, photo and "this exact card" cannot be part of a Trade.
- Any signed-in holder of a Trade link can read what is offered until someone joins; only the second seat is exclusive, claimed atomically by the first join. A wrongly claimed room is fixed by cancelling it and making a new one, which draws a fresh slug.
- Cards waiting in a Trade Inbox count toward Master Inventory, so a Wishlist Item can show as fulfilled before the receiver files anything.
- Execution deducts from every Trade Item against a pre-trade snapshot of both sides, so a Card offered on both sides nets to nothing and is recorded as it happened, with no special case.
