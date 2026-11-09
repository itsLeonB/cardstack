# Trades

Status: ready-for-agent

## Problem Statement

Two Cardstack users who want to swap cards have no shared place to agree the terms. They send screenshots of Collections, agree in a chat app, meet up, hand the cards over, and then each has to hand-edit quantities in the right Collection on both sides — removing what left, adding what arrived — while trying to remember exactly what was in the deal. Nothing records what was agreed, so a miscount is discovered days later with no way to check, and a card swapped into a Collection the receiver had not thought about lands in the wrong place or is forgotten entirely.

## Solution

A user starts a Trade and shares its link. Anyone signed in who opens it can read the offered cards; the first one to join takes the second seat, and from then on each side adds Trade Items — a Card, a quantity, and the Collection it will come from. When both sides accept, the Trade locks so the agreed set cannot drift. After the cards physically change hands, both sides confirm the handover and one exchange applies it: the offered quantities leave each side's Collections and land in the other side's Trade Inbox, a Collection set apart from their own where the received cards wait to be filed.

## User Stories

1. As a signed-in user, I want to start a Trade, so that I have a place to agree a swap with someone instead of doing it in a chat.
2. As a user starting a Trade, I want a link to it, so that I can send it to the person I want to trade with.
3. As a user starting a Trade, I want to add my cards to my side before anyone joins, so that the person I invite sees what I am offering.
4. As a user starting a Trade, I want to pick the quantity and the Collection each offered Card comes from, so that I know where it will leave from.
5. As a user starting a Trade, I want to remove or change my Trade Items while the Trade is negotiating, so that I can refine the offer.
6. As a signed-in user opening a Trade link, I want to read what the other side is offering before joining, so that I can decide whether I want it.
7. As a signed-in user opening a Trade link, I want to take the second seat with one click, so that joining does not require an invitation from inside the app.
8. As a second user who joined a Trade, I want the seat to be mine alone, so that a third person cannot join the deal I am now part of.
9. As a signed-in user who opens a Trade link after someone has joined, I want to be told the Trade is full, so that I stop trying.
10. As a participant, I want to see the other participant's full name, so that I know who I am dealing with.
11. As a participant, I want my email and the other participant's email never shown, so that arranging a trade does not leak contact details.
12. As a participant, I want to see each Trade Item with its Card, quantity, set and rarity, so that I can check the deal against the physical cards.
13. As a participant, I want to add several Trade Items from different Collections, so that one Trade can draw on the whole of what I have.
14. As a participant, I want the same Card to appear twice if it comes from two Collections, so that the app does not stop me expressing what I actually mean.
15. As a participant, I want an empty side to be allowed, so that I can give a card away without inventing a second side to the deal.
16. As a participant, I want the Trade to refuse to move when both sides are empty, so that an empty Trade cannot be executed by accident.
17. As a participant, I want to accept my side, so that the other side knows I agree to the set as it stands.
18. As a participant, I want the whole Trade to lock when both sides have accepted, so that the set the deal is about cannot change after we agreed it.
19. As a participant, I want to see that the Trade is waiting on the other side to accept, so that I know the deal is not yet agreed.
20. As a participant, after the cards change hands, I want to confirm the handover, so that the trade is recorded as done.
21. As a participant, I want the exchange to apply only when both sides have confirmed, so that one side cannot alone make the other side's Collection show cards that never moved.
22. As a participant, I want the exchange applied in one step, so that a failure leaves both sides exactly as they were.
23. As a participant, I want the app to refuse the whole exchange if any of my offered cards is no longer there in the quantity I offered, so that I am never told a deal happened when it did not.
24. As a participant, I want the refusal to name the Trade Item that went short, so that I know what to fix or renegotiate.
25. As a participant, I want the source Collection used first for the deduction and my other Collections used when it no longer holds enough, so that tidying my binders the night before does not break a deal.
26. As a participant, I want the Trade to stay in the room after a refusal, so that we can renegotiate instead of starting again.
27. As a participant, I want to see when the Trade's cards no longer match what was agreed, as soon as the Trade page is opened, so that I find out before meeting instead of at the handover.
28. As a participant offering cards from a binder-style Collection, I want the exchange to empty the right Slots, so that the binder view stays honest.
29. As a receiver, I want every card that arrives to land somewhere I can see, so that a trade never makes cards disappear from my account.
30. As a receiver, I want the arrived cards collected in a Trade Inbox apart from my own Collections, so that I decide where they really belong instead of inheriting a guess.
31. As a receiver, I want one Trade Inbox per Trade that gives me cards, so that I can tell which deal each card came from.
32. As a receiver, I want the Trade Inbox named after the other participant, so that I recognise it at a glance.
33. As a receiver, I want to file an arrived card into a Collection of mine, so that it takes its place among the rest.
34. As a receiver, I want the Trade Inbox to disappear once I have filed its last card, so that empty leftovers do not clutter my Collections.
35. As a receiver, I want the cards in a Trade Inbox counted as mine straight away, so that my totals and Wishlist fulfilment reflect what I already own.
36. As a receiver, I want to browse a Trade Inbox but not edit or delete it, so that the only way out of it is filing the cards somewhere real.
37. As a participant, I want to cancel a Trade that is open, negotiating or locked, so that an abandoned or hijacked deal cannot hold my cards.
38. As a participant, I want my cancelled Trade kept in my history, so that I can see what was called off.
39. As a participant, I want a Trades list in my account, so that I can find a Trade again after closing the tab.
40. As a participant, I want my open Trades shown with how long they have stood, so that I can spot ones going stale.
41. As a user, I want the Trades list to show what each Trade's state is, so that I know which ones need my click.
42. As a user, I want to see only Trades I am a participant in, so that nobody else's deals are discoverable through my account.
43. As a Guest, I want a Trade link to answer as not found, so that a leaked link leaks nothing to people without an account.
44. As a user, I want a Trade page marked unindexed, so that my deals never appear in search results.
45. As a user, I want the same not-found answer for a Trade link that does not exist, is cancelled, or is not mine to read, so that I cannot learn anything from the difference.
46. As a user who made a Trade and shared the link by mistake, I want to cancel it and start a new one, so that a wrong recipient cannot take the second seat.
47. As a user, I want a new Trade to get a new link, so that a cancelled link never comes back to life.
48. As a user with a full wishlist of trades, I want no email, push or in-app notification to be needed, so that the feature works without new delivery infrastructure.
49. As a user, I want to reach my Trades from the account area, so that it sits beside the rest of my account rather than in the catalog.
50. As a user, I want the Trade page to work on my phone at the table, so that reading the agreed list during the handover is easy.

## Implementation Decisions

- Domain vocabulary is fixed by `GLOSSARY.md`: Trade, Trade Item, Trade Inbox. The exposure model and the Trade Inbox exception are ADR-0028; this spec uses them and does not restate them.
- A Trade is a new domain aggregate owned by the trades service, with two tables: one Trade row and one Trade Item row per offered card. The second seat is a nullable participant reference on the Trade, claimed atomically by the first authenticated join.
- Trade states are `open`, `negotiating`, `locked`, `executed`, `cancelled`, held in one status column as the single source of truth. Two nullable acceptance timestamps sit beside it only to answer "who still has to accept"; they never decide the state.
- The Trade's link uses the Slug machinery from ADR-0025 with a difference: a Trade's slug is permanent for the Trade's life (a new Trade draws a new one) and its page requires an Auth Identity, unlike a Shared page which answers a Guest.
- Read and write are separate calls by design: reading the page never claims the seat, joining does. Joining is a single conditional update that succeeds only while the second seat is empty, so simultaneous joins resolve to one winner with no lock held by the reader.
- A Trade Item carries a Card, a quantity, and the source Collection it is meant to leave. One Trade Item per Card and source Collection per side is allowed; the same Card may appear on both sides.
- Execution deducts against a snapshot of both sides taken before any write, then adds, inside one transaction. Same-Card-on-both-sides therefore nets to nothing and is recorded as it happened, with no netting branch.
- Deduction takes the quantity from the named source Collection first and then from the caller's other Collections holding the same Card, refusing the whole Trade only when the participant's total is short. Every refusal names the Trade Item that went short.
- Refusals follow the Move precedent: the whole exchange is refused and nothing moves, rather than clamping to what is left.
- Execution reads and writes through the existing transaction helper, locking the Trade row before checking anything, so two confirmations landing together cannot both execute.
- The Trade Inbox is an ordinary Collection carrying a nullable reference to its Trade. That one column does two jobs: it flags the Collection, and it names the Trade the received cards came from. The Collections list omits any Collection with it set.
- A Trade Inbox is created at execution, only for a side that receives at least one card, and is deleted when its last Inventory Entry leaves. Filing is the only way a card leaves it: the inbox page offers browse and file, no rename, quantity edit or delete.
- Trade routes live under one prefix with the usual split: create, read by slug, join, list mine, add item, remove item, accept, cancel, confirm, and the inbox file operation. Each keeps its Huma input type separate from the request DTO it builds.
- All trade failures the client branches on get stable codes in `apperr` and are documented on the error model: seat taken, trade not open for this operation, the caller is not a participant, an offered quantity is short, both sides empty, and the Trade is already executed.
- New index support is needed for the queries the feature adds: the Trade by slug, Trades by participant, and Trade Items by Trade.
- The frontend adds a Trade page reached by slug (readable before joining), a Trades list in the account area, and a Trade Inbox page. The inbox page reuses the existing Collection item listing.

## Testing Decisions

A good test here asserts what a caller of the API can observe — status codes, error codes, and the quantities in Collections and Trade Inboxes before and after — and never internal calls, repository specifications or mocked interactions. Anything that can only be seen by reading state inside the service is not worth a test at this seam.

Backend: one seam, the routes package, which already runs the real schema, the real services and real bearer tokens. It is the highest seam in the codebase, and `inventory_move_routes_test.go` is the prior art — it proves refuse-whole, quantity merging, destination limits and empty-source behaviour for Move. The trade tests cover, through the API only:

- The state machine advancing and refusing to skip states, including the guard that nothing may be added, removed or accepted once locked.
- Two joins racing: exactly one takes the seat, the other gets the seat-taken code.
- A Guest reading a Trade page gets not found, as does a participant of no Trade.
- Cancelling from each state, and that a cancelled Trade's slug never resolves again.
- Execution with a short Trade Item: one refusal, the named Card in the message, and both sides byte-identical afterwards.
- Execution where the source Collection is gone but the participant still owns the Card elsewhere: the exchange goes through from the fallback.
- Execution where the same Card sits on both sides: both sides end with their original quantity, and the Trade is marked executed.
- A one-sided Trade: the receiver gets a Trade Inbox holding exactly the offered cards, the giver gets none.
- Filing: cards move from the Trade Inbox into a chosen Collection, and the Trade Inbox disappears with its last card.
- A binder-style source Collection: the Slots holding the offered Card are emptied, and the binder's quantity stays in step with its Slots.

Handler-level tests are deliberately not added: they stub the service, and the routes seam already drives the real one. Service unit tests for the state machine are deliberately not added either: they would mock two generic repositories and assert specifications, which is testing the implementation the routes tests observe from outside.

Frontend: one end-to-end spec, signed in, against the real Clerk instance and the real API, driving two browser contexts so the second seat is taken by a genuinely different user. Prior art is `wishlist-preset.spec.ts` for a signed-in flow and `shared-collection.spec.ts` for a slug-addressed page reached by client navigation. It covers the happy path — start, share, join, both sides add, both accept, both confirm, cards appearing in the receiver's Trade Inbox — plus one key failure, a confirmation refused because an offered card was already gone. Component tests are not added for the Trade page: its behaviour is the server's, and a component test would only restate the payload shape.

## Out of Scope

- A public or per-user inventory, and any directory of users or tradable Trades — ADR-0028 settles the exposure model as Trade Items only.
- Reserving offered quantities while a Trade is locked. A locked Trade can go short before execution; the room shows it and execution refuses.
- Notifications of any kind: no email, push or in-app inbox. Trades are polled from the Trades list.
- Automatic expiry of an unexecuted Trade, and any scheduled job.
- Messaging, comments, counter-offer history or any negotiation log inside a Trade.
- Money, price, shipping or any payment step.
- Identity per physical copy: Card Variant is not modelled (ADR-0006), so a Trade Item can never name a condition, a photo, or one specific card.
- Kicking a participant who claimed the seat, or regenerating a Trade's link in place.
- Partial execution, or executing some Trade Items and refusing others.
- Guest access to a Trade page, and any sharing of a Trade other than by its link.
- Admin visibility, moderation or reporting of Trades.
- Reputation, ratings or completed-trade counts on a profile.

## Further Notes

The Trade Inbox is the one place where the flag could be mistaken for a second kind of holding: it is not, and any future code that reads "the user's Collections" has to decide explicitly whether inboxes belong in that set. The answer this spec adopts is that they count for quantity (Master Inventory, Wishlist fulfilment) and not for identity (the user's own list of Collections).

The absence of any reservation is the deliberate risk in this design: a Trade can be agreed, locked, and then fail at execution because a card went elsewhere in the meantime. The mitigation is visibility — the room re-checks and marks a short Trade Item whenever it is opened — not a hold on the quantity. If failed executions are ever observed in practice, the upgrade is a held-quantity concept, and it is a cross-cutting change to Inventory Entries, so it deserves its own ADR rather than a patch.

Nothing here depends on new delivery infrastructure, new app hosting, or a schema change to existing tables beyond one nullable column on Collections, which is what keeps the feature inside the current deployment shape.
