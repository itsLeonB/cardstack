# Trade title, actions and item presentation

Status: ready-for-agent

## Problem Statement

A participant opens a Trade and sees a page titled only "Trade". Both sides are headed with personal names, so nobody can tell at a glance which side is theirs. Each offered card carries a long "Remove <card name>" button, and adding a card means filling a form buried in the side panel. Invite stays on screen after the other person has joined, so it invites someone to a seat that is already taken. The Trades list is a stack of similar rows that are hard to scan. When a Trade was made by mistake and nobody joined it, Cancel leaves a cancelled row in history forever and the user has no clean way to remove it. The result is a user who cannot find the right Trade, may invite the wrong person, and cannot discard an empty mistake.

## Solution

The Trade detail page is titled with the counterparty's name once they join, or "New trade" while the Trade is open, with the state kept as a description. Your own side is labelled "You" and the other side keeps its display name. The Remove button reads "Remove" and carries the card name as its accessible name. Adding a Trade Item happens on its own page that reuses catalog search, keeps the source Collection and quantity fields, and returns to the detail page when done. Invite appears only while the Trade is open. Each Trade Item and each row in the Trades list is a card that shows the details needed to tell items apart. A Trade that nobody joined can be Discarded after confirmation: the Trade and its Trade Items are deleted and its link stops working at once. A Trade that someone joined is Cancelled in the existing way and stays in history.

## User Stories

1. As a participant, I want the Trade detail page titled with the other participant's name, so that I can tell which Trade I have open.
2. As an owner of an open Trade, I want the title to read "New trade" until someone joins, so that I am not shown a name that does not exist yet.
3. As a participant, I want the Trade state still shown near the title, so that I know whether the deal is negotiating, locked, executed or cancelled.
4. As a participant, I want my own side labelled "You", so that I can tell my offer from the other person's at a glance.
5. As a participant, I want the other side labelled with that person's display name, so that I know who I am dealing with.
6. As a participant viewing from either seat, I want "You" always attached to my side, so that the label does not depend on which side I hold.
7. As an owner before anyone joins, I want the empty seat to keep its "waiting for someone to join" wording, so that I know the second seat is free.
8. As a participant, I want no email address or full holdings shown in the new labels, so that arranging a Trade does not leak contact details or private inventory.
9. As a Guest, I want the same not-found answer as before, so that the new page cannot be used to discover Trades.
10. As a signed-in non-participant, I want the same not-found answer as before, so that the new page cannot expose someone else's deal.
11. As a participant, I want a short "Remove" button on each of my Trade Items, so that the row stays readable.
12. As a participant using a screen reader, I want the Remove button's accessible name to include the card name, so that I can tell which offer I am removing.
13. As a participant, I want Remove to stay on my own side only, so that I cannot change the other person's offer.
14. As a participant, I want Remove to stop being available once the Trade locks, so that the agreed set cannot drift.
15. As a keyboard user, I want every Remove button reachable and operable, so that editing does not need a pointer.
16. As a participant, I want to add a Trade Item on a page of its own, so that the detail page is not crowded by a form.
17. As a participant, I want to open the add-card page from the Trade detail page, so that I do not have to find a separate entry point.
18. As a participant, I want the add-card page to reuse the catalog search I already know, so that finding a Card works the same way everywhere.
19. As a participant, I want to choose the source Collection on the add-card page, so that the app knows which Collection the offered copies should leave.
20. As a participant, I want to choose the quantity on the add-card page, so that I can offer more than one copy.
21. As a participant, I want to see the result of my addition back on the detail page when I return, so that I can check the offer immediately.
22. As a participant, I want the browser back action to return me to the Trade detail page, so that I can abandon an add without losing my place.
23. As a participant, I want a refused add to tell me why, so that zero quantities, Collections I do not own and duplicate Trade Items stay impossible.
24. As a participant, I want to add the same Card twice from two different Collections, so that I can express an offer drawn from more than one binder.
25. As a participant, I want add to be refused once the Trade is locked or cancelled, so that a finished deal cannot be changed.
26. As a participant, I want the add-card page to show only what I am allowed to offer, so that the app does not expose another user's holdings.
27. As an owner of an open Trade, I want to see Invite, so that I can send the link to the person I intended.
28. As an owner, I want Invite to disappear as soon as someone joins, so that I cannot invite someone to a taken seat.
29. As an owner of a locked, executed or cancelled Trade, I want Invite absent, so that a finished Trade cannot gain a new participant.
30. As a participant who is not the owner, I want no Invite action, so that I cannot share a Trade I do not own.
31. As an owner, I want the existing link dialog, Copy and native share behaviour preserved, so that Invite works as it does today.
32. As an owner, I want no Collection or Wishlist publication controls added to Invite, so that my private holdings stay private.
33. As a participant, I want each Trade Item shown as its own card with the Card image and name, so that I can recognise it quickly.
34. As a participant, I want the set and rarity on each item card, so that I can check the deal against the physical cards.
35. As a participant, I want the quantity and the source Collection on each item card, so that I know how many copies move and from where.
36. As a participant, I want an item that has gone short clearly marked, so that I find out before the handover.
37. As a participant, I want the other side's items presented the same way as mine, so that I can compare the two offers.
38. As a user with many Trades, I want each row in the Trades list to be its own card, so that I can scan the list.
39. As a user, I want each Trade card to show the counterparty name or "New trade", the state and the age, so that I can find the Trade I mean.
40. As a user, I want cancelled Trades still listed as non-links with their state, so that history stays visible.
41. As a user, I want the list's newest-first order, empty state, error state and Retry action preserved, so that the list stays usable.
42. As an owner of an open Trade nobody joined, I want a Discard action, so that I can remove a mistake completely.
43. As an owner choosing Discard, I want a confirmation that names the action as destructive, so that I do not delete a Trade by accident.
44. As an owner confirming Discard, I want the default focus on the safe action, so that a stray Enter key does not delete anything.
45. As an owner of a Discarded Trade, I want the Trade and its Trade Items gone and its link to answer not found, so that a wrong recipient cannot use it.
46. As an owner of a Discarded Trade, I want no cancelled row left in my Trades list, so that a mistake does not clutter my history.
47. As a participant in a negotiating or locked Trade, I want Cancel unchanged, so that a called-off deal stays in history.
48. As a participant in an executed Trade, I want neither Discard nor Cancel offered, so that a completed exchange cannot be undone.
49. As an owner, I want Discard and Cancel visually separated from ordinary actions, so that I never confuse them with Edit or Accept.
50. As a user on a phone or a desktop, I want the new cards and actions laid out cleanly on both, so that presentation does not break at either size.
51. As a user in either theme, I want the new cards and danger action readable, so that contrast never hides a control.
52. As a keyboard user, I want the picker, confirmation and all new actions fully operable, so that no step needs a pointer.
53. As a participant, I want a failed Discard or Cancel reported plainly, so that I know the Trade is still there.
54. As a user, I want no new notifications or emails from these actions, so that the app's poll-only behaviour stays as it is.

## Implementation Decisions

- The Trade detail title is derived from the counterparty display name when one exists, and from the "New trade" wording while the Trade is open. The status stays as the page description; no new status value is introduced.
- The two side labels are derived from the viewer's side of the Trade, not from a fixed position. The viewer's side reads "You"; the other side keeps its existing display name and its existing fallback for an empty seat.
- The Remove control keeps its visible text short and moves the card name into the accessible name. Its existing authorization, side-ownership and "negotiating only" rules do not change.
- A new route hosts the add-card picker. It reuses the existing catalog search results component, and keeps the source Collection and quantity on that page. Submitting adds the Trade Item through the existing add mutation and returns to the Trade detail page. No second confirmation step is added.
- Invite is shown only to the owner while the Trade state is `open`. The existing share dialog, link text, Copy and native share behaviour are kept. No publication or access control is added.
- Discard and Cancel are separate operations. Discard applies only to an `open` Trade: it deletes the Trade and its Trade Items, and the Slug stops resolving. Cancel applies to `negotiating` and `locked` Trades and keeps the existing soft-cancel behaviour, cancelled state, dead Slug and history row. `executed` Trades refuse both.
- The backend gains a delete operation for an `open` Trade, including its Trade Items. The existing status-update path for Cancellation is unchanged. The read and list payloads keep their current shape; the UI derives the new labels from existing fields.
- Trade Items on both sides and rows in the Trades list use the existing card, badge and avatar patterns of the design system. A Trade Item card shows image, name, set, rarity, quantity and source Collection; a Trade card shows the derived name, state and age.
- No new Trade state, filter, pagination, notification or delivery mechanism is added. No existing migration changes the status column.
- ADR-0029 records the Discard/Cancel split. `GLOSSARY.md` defines Cancel and Discard as separate terms. The privacy boundary in ADR-0028 still holds: a picker must never expose a user's whole inventory through a Trade.
- Discard uses the design system's destructive confirmation pattern with danger emphasis, focus on the safe action, and no undo toast, because an `open` Trade has nothing to restore. One overlay implementation is chosen per screen; native dialog semantics and the house dialog wrapper are not mixed.
- Existing access rules stay: Guests and non-participants get not found, Trade pages stay unindexed, and no email or holdings appear in Trade payloads.

## Testing Decisions

- A good test checks external behaviour only: the visible title and labels, the accessible name of Remove, which actions appear in each state, where navigation leads after adding or discarding, the API response, and whether a Trade still exists. No test asserts internal component state, helper calls or implementation details.
- Primary seam: the authenticated frontend end-to-end trade suites. They exercise real Clerk sign-in and the real API, so one seam covers the title, "You", Remove's accessible name, Invite visibility across states, the picker round trip, card presentation and the Discard confirmation. Existing assertions on access, cancel, execute and session behaviour are kept, not weakened.
- Supporting seam: the backend trade route test suite. One new case proves that Discard removes an `open` Trade and its Trade Items and that the old Slug answers not found. The existing cancel-from-each-state cases stay as the proof for joined Trades.
- Prior art: the existing authenticated trade end-to-end suites and the existing HTTP-level trade route suite already cover Trade creation, joining, item edits, acceptance, execution, list scoping and cancellation. New cases extend those suites instead of creating parallel ones.
- The focused component suites for the Trade page and the share dialog are not the primary seam. They are extended only for behaviour that a browser cannot observe.
- Verification uses the repository's stack: the frontend verification script for lint, format, typecheck, unit tests, build and authenticated end-to-end tests, and the backend verification script for build, vet, format, lint and tests. Both are run on final source and their exit status is reported.

## Out of Scope

- No new Trade states, notification channels, filters, pagination or search over Trades.
- No change to joining, accepting, locking, confirming, executing, shortfall marking or Trade Inbox filing.
- No change to Guest access, non-participant access, Slug rules, private indexing or the Share dialog contract.
- No restyling of catalog search itself, and no new design system components.
- No migration or cleanup of existing cancelled Trade rows.
- No new work on Wishlists, Collections, scanning, the Admin app or the mobile UI follow-up tickets other than the Invite and row rules this spec supersedes.

## Further Notes

- This spec supersedes the Invite and Trade-row rules in the completed mobile UI follow-up ticket 03, where Invite was hidden only for executed Trades and participants links read "With <name>". It also supersedes the original Trades spec stories that say an `open` Trade is cancelled and kept in history; the new rule is in ADR-0029.
- The planning session had no access to the annotated screenshots, and several annotation notes were cut off. During implementation, check each decision against the current page before changing code, and stop if the live UI contradicts this spec.
- The domain terms used here follow `GLOSSARY.md`; the privacy boundary follows ADR-0028 and the end-of-Trade rule follows ADR-0029.
