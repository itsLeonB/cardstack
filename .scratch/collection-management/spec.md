# Collection management: sharing, moving, a Wishlist and a Scan Filter

Status: ready-for-agent

## Problem Statement

A user who keeps their cards in Cardstack can see them only for themselves. They cannot show a Collection to a friend or a trading group without taking screenshots, and a person they want to trade with has no way to see what they own or want. When a user reorganises, there is no way to take a group of cards out of one Collection and put it in another: they must remove each card and add it again, one by one, and in a big Collection that means hundreds of edits. A user also has no place to record the cards they are looking for, so a missing card lives in their head or in another app, and there is no way to see which cards of a set they are still missing. Finally, the scanner searches the whole catalog every time, so a user scanning a binder of one or two sets is offered Cards from sets they know are not in it, and the right Card is easier to miss.

## Solution

Four connected changes, each using the terms in GLOSSARY.md.

A user can make a Collection Shared. Sharing is off by default and is the owner's explicit choice: a Share button (a share icon) in the Collection header opens a dialog with a switch, a read-only link with a Copy button, and a warning that the owner's full name appears in the link and on the page. Anyone with the link, signed in or not, sees the Collection read-only: the owner's name, the title, the description, the cards with their quantities and, for a binder, its Pages and Slots, with the same filters the owner has. Turning sharing off makes the link stop working at once.

A user can Move cards between Collections. A Select button on a Collection's cards turns on selection mode, where cards are ticked one by one or all at once, including every card that matches the current filter across pages, such as every MA5 card in a Collection that holds MA5 and MA6. A Move to sheet lists the destination Collections, a confirm dialog states the count, and the cards, with every copy of each, go to the destination.

A user has a Wishlist. It is a page of its own beside Inventory and Collections, filtered like them. Cards are added by hand from the catalog with a quantity, or by a Wishlist Preset, which fills the Wishlist once with every Card of chosen Expansion Sets that the user owns none of yet. Each item shows how many the user owns of how many they want. The Wishlist can be Shared the same way as a Collection. Anyone who opens a Shared Wishlist sees the owner's progress on each item and can filter by it. A signed-in viewer also sees how many of each Card they own themselves, which shows where a trade is possible.

A user can limit what a scan looks for. A Limit search row above the camera, on both scan modes, opens a sheet where Expansion Sets and rarities are chosen. The matcher then offers only Cards inside that Scan Filter, and so do the other-prints suggestions.

## User Stories

Sharing a Collection

1. As a Collection owner, I want a Share button with a share icon on my Collection, so that I can find sharing from the page I am already on.
2. As a Collection owner, I want sharing to be off until I turn it on, so that nothing I own becomes visible by accident.
3. As a Collection owner, I want to be told before I first turn sharing on that my full name appears in the link and on the page, so that I agree to that knowingly.
4. As a Collection owner, I want a read-only link with a Copy button once sharing is on, so that I can send it in one tap.
5. As a Collection owner, I want a confirmation that the link was copied, announced to a screen reader too, so that I know the copy worked.
6. As a Collection owner, I want turning sharing off to ask for confirmation and then make every existing link stop working at once, so that I can take a Collection back.
7. As a Collection owner, I want the link to keep working when I rename the Collection while it is Shared, so that links I have already handed out do not break.
8. As a Collection owner, I want sharing again after turning it off to give me a fresh link built from the current title, so that an old link I revoked never comes back.
9. As a Collection owner, I want the Share button on a Collection's page to show whether it is Shared right now, so that I never have to guess.
10. As a Collection owner, I want a Collection I never shared to stay private to me, so that sharing one does not expose the others.
11. As a Collection owner, I want deleting a Shared Collection to end its link, so that nothing of it stays reachable.

Viewing a Shared Collection

12. As anyone with the link, signed in or not, I want to open a Shared Collection without an account, so that a friend does not have to sign up to see my cards.
13. As a viewer, I want to see who shares the Collection, its title and its description, so that I know what I am looking at.
14. As a viewer, I want the cards and their quantities in the same layout the owner sees, read-only, so that nothing looks different from the app I may use myself.
15. As a viewer of a binder, I want to see its Pages and Slots, so that I see how it is laid out.
16. As a viewer of a binder, I want Unidentified Slots shown as a count, so that I see that the binder holds cards that have no identity yet.
17. As a viewer, I want to filter the cards by name, Expansion Set, rarity, category and tag, including as a Guest, so that I can find a card in a large Collection.
18. As a viewer, I want to see no edit controls and no email address, so that the page is safe to share.
19. As a viewer, I want a link that does not work to show the same not-found page whether the Collection is private, no longer Shared or never existed, so that nothing about someone's Collections leaks.
20. As an owner, I want a shared page to be left out of search engines, so that my name does not turn up in a search.
21. As anyone, I want cards from unpublished Expansion Sets to stay hidden on a shared page, so that it follows the same rule as everywhere else.

Moving cards between Collections

22. As a user, I want a Select button on a Collection's cards, so that I can choose several cards to act on.
23. As a user, I want a checkbox on every card in selection mode and an action bar that says how many are selected, so that I always know what a Move will act on.
24. As a user, I want to select all the cards on the page and then every card matching the current filter across pages, so that I do not tick hundreds one by one.
25. As a user, I want to filter a Collection by Expansion Set and move the whole result, so that I can move all MA5 cards out of a Collection that holds MA5 and MA6.
26. As a user, I want filters, search and selection to work together, so that I can narrow down and then pick.
27. As a user, I want a Move to sheet listing my Collections, so that I choose the destination in one place.
28. As a user, I want the Collection I am moving from, and every binder, shown disabled with the reason, so that I understand why I cannot pick them.
29. As a user, I want a confirmation that states how many cards and copies move from where to where, so that I do not move the wrong thing.
30. As a user, I want every copy of a chosen card to move, so that a card does not end up split across two Collections by accident.
31. As a user, I want a card the destination already holds to have its quantity added to, so that no duplicate entry is created.
32. As a user, I want a Move that would pass the destination's maximum card-count limit to be refused whole, with an error naming the limit and by how much it is over, so that I never end up with half a Move.
33. As a user, I want to move cards out of a binder into a Collection without a Layout, so that I can take cards out of a binder.
34. As a user, I want the Slots that held the moved cards to become empty, so that the binder still matches what it physically holds.
35. As a user, I want Unidentified Slots to stay where they are when I move a binder's cards out, so that nothing unknown is lost.
36. As a user, I want a Move into a binder to be impossible, so that no Slot gets a card nobody chose a place for.
37. As a user, I want the action bar on mobile to sit at the bottom of the screen clear of the safe area, so that I can reach it with my thumb.
38. As a user, I want leaving selection mode to clear the selection, so that a stale selection cannot be moved later.

The Wishlist

39. As a user, I want a Wishlist page beside Inventory and Collections, so that wanting a card has a home of its own.
40. As a user, I want the Wishlist filtered like Inventory by name, Expansion Set, rarity, category and tag, so that I find things the way I already do.
41. As a user, I want to add a Card to my Wishlist from the catalog card page and from catalog search results, with a quantity that starts at 1, so that I can add a card the moment I see it.
42. As a user, I want adding a Card that is already on the Wishlist to replace its wanted quantity, so that I change a number rather than create a duplicate.
43. As a user, I want to change a wanted quantity or remove an item, so that the list stays true to what I want.
44. As a user, I want each item to show how many I own of how many I want, taken from my Master Inventory, so that I see my progress.
45. As a user, I want an item to show as fulfilled once I own as many as I want, with the word and not colour alone, so that I see it whatever my vision.
46. As a user, I want an item I have fulfilled to stay on the list until I clear it, so that finishing a card is something I see and not something that silently disappears.
47. As a user, I want a Clear fulfilled action, so that I can tidy up in one step.
48. As a user, I want to filter the Wishlist by All, Still needed and Fulfilled, so that I see only what I am still looking for.
49. As a user, I want a Fill from sets action with a Wishlist Preset, so that I can fill the Wishlist for a whole set at once.
50. As a user, I want to pick Expansion Sets in the preset sheet from a multi-select with wrapping chips, so that I can see and remove what I picked.
51. As a user, I want to narrow a preset by rarity, category and tag, so that I can ask for only the rare cards of a set.
52. As a user, I want the preset to add Cards of those sets that I own none of in my Master Inventory, so that it fills the gaps and nothing else.
53. As a user, I want to choose the quantity the preset gives each item, so that I can want two of each if I play with playsets.
54. As a user, I want a live preview of how many cards will be added before I confirm, so that a preset never surprises me.
55. As a user, I want a preset to skip Cards already on my Wishlist and leave their quantity alone, so that applying it twice is harmless.
56. As a user, I want a preset applied once and then forgotten, so that later catalog or inventory changes do not rewrite my Wishlist.
57. As a user, I want Wishlist items never to count toward my Master Inventory, so that wanting a card is never confused with owning it.
58. As a user, I want cards from unpublished Expansion Sets to be absent from the Wishlist, from the preset and from its preview, so that it follows the same rule as everywhere else.

Sharing and viewing a Wishlist

59. As a user, I want the Wishlist to have the same Share button, switch, warning and Copy as a Collection, so that sharing works the same everywhere.
60. As a user, I want a Wishlist to be private until I share it, so that nothing is visible by accident.
61. As anyone with the link, I want to see a Shared Wishlist read-only, with the owner's name and each item's wanted quantity, so that I know what they are looking for.
62. As anyone with the link, including a Guest, I want to see the owner's owned and wanted for each item and whether it is fulfilled, so that I know what they still need.
63. As anyone with the link, including a Guest, I want to filter a Shared Wishlist by All, Still needed and Fulfilled, so that I look at what they still need.
64. As a signed-in viewer, I want to see how many of each Card I own myself next to each item, so that I see which of their wants I can fill.
65. As a signed-in viewer, I want to filter a Shared Wishlist by I have it and I do not have it, so that I see only the cards I could trade.
66. As a signed-in viewer, I want my own inventory used only for my own view, so that the owner never sees what I own.
67. As a Guest, I want a Sign in to see what you own banner on a Shared Wishlist, so that I know why the overlay is missing and how to get it.
68. As an owner opening my own Shared Wishlist link, I want to see my own Wishlist with no overlay, so that I am not shown my own cards against myself.
69. As a viewer, I want to see no messaging, no contact details and no offer flow, so that I know trades are arranged outside the app.
70. As an owner, I want my Wishlist link to end the moment I switch sharing off, so that I can take it back.

Limiting a scan

71. As a user, I want a Limit search row above the camera on both scan modes, so that I can set it before I take a photo.
72. As a user, I want the row to read All sets, All rarities when I have set nothing, so that I know the scanner is searching everything.
73. As a user, I want a sheet where I pick Expansion Sets and rarities, so that I can limit by either or both.
74. As a user, I want a Sets in this Collection shortcut in the sheet, so that I can limit a scan to the sets I am already keeping together.
75. As a user, I want my choices shown as wrapping chips, never a clipped row, so that I see everything I picked.
76. As a user, I want a Clear action in the sheet, so that I can go back to searching everything in one tap.
77. As a user, I want the sheet to apply on close, so that there is no separate Save to forget.
78. As a user, I want the matcher to offer only Cards inside my Scan Filter, so that a card from a set I know is not in the binder is never suggested.
79. As a user, I want the other-prints suggestions to stay inside my Scan Filter too, so that the same limit holds for the whole screen.
80. As a user, I want a photo that matches nothing inside my Scan Filter to be marked Unidentified with a note such as Searched MA5, MA6 only, so that I understand why nothing matched.
81. As a user, I want no automatic widening of a limited search and no search-all button, so that my Scan Filter means what it says.
82. As a user, I want a change to the filter mid-session to affect only the photos I take afterwards, so that the results I have already reviewed do not change under me.
83. As a user, I want my last Scan Filter remembered on this device, so that I do not set the same sets again for the next binder page.
84. As a user, I want Corrections to be recorded for a limited scan as for any other, so that the matcher's weak spots stay visible to the operator.

## Implementation Decisions

Sharing

- A Collection is Shared when it has a Slug and private when its Slug is NULL. There is no separate flag on a Collection. A Wishlist has a plain shared flag, off by default, because there is one per user at a fixed path.
- A profile always has a Slug: the owner's name lower-cased and hyphenated, then a hyphen and the last 5 hex characters of the profile's id. It is made when the profile is created, never changes, and is unique. A migration adds the column and backfills every existing profile. A name that slugifies to nothing falls back to `user`.
- A Collection Slug is made the moment sharing is turned on, from the title at that moment and the last 5 hex characters of a fresh random uuid (not the Collection's id, which would reproduce the old Slug and revive a revoked link on re-share). It stays fixed while Shared, is cleared to NULL when sharing is turned off, and is made again from the current title the next time. It is unique within the owner's Collections. A title that slugifies to nothing falls back to `collection`.
- The last 5 characters are used, not the first 5, because ids are time-ordered `uuidv7()` values whose leading characters repeat across rows made close together. The reasoning and the rejected alternatives are in ADR-0025.
- The shared link has two path segments: `/s/<profile slug>/<collection slug>` for a Collection and `/s/<profile slug>/wishlist` for the Wishlist. A Collection Slug always ends in a 5-character tail, so it can never equal `wishlist`.
- The backend adds a public route group, open to Guests and to signed-in callers, with its own rate limit, that resolves a profile Slug and then a Collection Slug (or the Wishlist). The caller's identity is used only to compute the viewer's own owned quantities on a Shared Wishlist. This is the only public surface beyond the catalog preview.
- A shared Collection response carries the owner's display name, the Collection's title and description, its entries with quantities and the same filter parameters and facets as the owner's listing, and, for a binder, its Layout, Pages and Slots with the Unidentified count. It never carries the owner's email or any identifier that lets the viewer reach the owner's other data. A private, unshared or nonexistent target gives one identical not-found response, both in body and in timing as far as is reasonable.
- Guests get the full card filters on shared pages. The catalog's Guest lock on rarity, category and tag filters and on facets does not apply here, because the data is the owner's and not the catalog.
- Cards of unpublished Expansion Sets are left out of every shared response, by the same rule as every other user-facing view.
- Shared pages are served with `noindex` and are in no sitemap. The frontend route sits outside the signed-in layout.
- The Share dialog is one component used for a Collection and for the Wishlist. It has the warning above the switch before first use, the switch ("Anyone with the link can view"), the read-only link and a Copy button with an announced confirmation, and a confirmation when turning sharing off ("Existing links will stop working"). The trigger is a button with a share icon and a text label that also shows the current state.

Moving cards

- Move is a backend operation from one Collection to another that takes either an explicit list of Card ids or a filter (the same shared card-filter parameters used by the listings), plus the destination. With a filter it acts on every Inventory Entry of the source that matches, across all pages. The two forms are mutually exclusive.
- It moves whole Inventory Entries, every copy of each. A Card the destination already holds has its quantity added to the existing entry.
- The source may be a plain Collection or a binder. When it is a binder, the Slots holding the moved Cards are emptied in the same transaction, through the one shared Slot-write path that ADR-0020 requires, so the entries stay equal to the Slot counts. Unidentified Slots are not touched.
- The destination must be a plain Collection. A binder as destination is refused with a machine-readable reason, since no Slots were chosen. The source and destination must differ, and both must belong to the caller.
- If the result would pass the destination's maximum card-count limit, the whole Move is refused before anything changes, and the error says what the limit is and by how many copies it would be exceeded. A Move is otherwise all-or-nothing in one transaction.
- The response says how many cards and how many copies moved. There is no undo.
- The frontend adds a selection mode to the Collection's entry list: a Select button, a checkbox on every card, a sticky action bar with the count, Move to and Cancel, and a banner offering Select all N matching this filter when the filter matches more than one page. Choosing it switches the request from a list of ids to the filter. The Move to sheet lists the user's Collections with the source and every binder disabled and the reason shown, then a confirm dialog states the cards and copies and the two Collection names. On mobile the action bar sits at the bottom clear of the safe area. Leaving selection mode clears the selection.

Wishlist

- A Wishlist is one per user, created on first use. A Wishlist Item is a Card and a wanted quantity of at least 1, unique per Card within the Wishlist. It is a separate record from an Inventory Entry and never counts toward Master Inventory.
- Reads return each item with its Card, the wanted quantity and the owner's owned quantity from Master Inventory. A status (fulfilled when owned is at least wanted) is derived on read and not stored. The listing takes the shared card-filter parameters and adds a status filter of All, Still needed or Fulfilled.
- Items are added or changed by setting the wanted quantity for a Card (the same set-quantity behaviour as Inventory Entries) and removed one by one. A Clear fulfilled action removes every fulfilled item in one call.
- A Wishlist Preset is a one-time request. It takes one or more Expansion Sets, optional rarity, category and tag narrowing and a quantity, and adds one item per matching published Card of which the user owns none in Master Inventory. It skips Cards already on the Wishlist without changing their quantity. A separate preview call returns the count that would be added, using the same query, so the sheet can show it live. Nothing about the preset is stored.
- A public Wishlist read shows each item with the owner's owned and wanted and the status, and supports the status filter for everyone. For a signed-in viewer it also returns the viewer's own owned quantity per item and supports an I have it or I do not have it filter. The viewer's data is read only for that request and never returned to the owner. A Guest gets no overlay and no I have it filter. The owner opening their own Shared link is shown the plain Wishlist, with no overlay.
- The frontend adds a Wishlist page as a top-level navigation item at `/wishlist` beside Inventory and Collections, reusing the Inventory list layout and the shared filter panel with the status filter. Add to Wishlist with a quantity stepper that starts at 1 sits on the catalog card page and on catalog search results. Fill from sets opens a sheet with a multi-select of Expansion Sets shown as wrapping chips, optional narrowing filters, a quantity stepper and a live count of cards to be added. The status of an item is always written out as a word, not shown by colour alone, and a count that changes is announced as one status message.
- The preset's phrase in the sheet is I own none yet of these sets, followed by the picker.

Scan Filter

- The scan match request gains optional Expansion Set and rarity restrictions, and so does the reprint lookup, so the nearest-Card search and the other-prints suggestions both stay inside them. Empty means no restriction. Unpublished sets stay excluded as before.
- The matcher never widens a restricted search by itself. When the restricted search yields nothing it reports the same Empty or Unidentified answer it gives today, and the frontend adds a note naming the sets and rarities searched, such as Searched MA5, MA6 only. There is no search-all-sets action.
- How the matcher's confidence judgement behaves on a restricted candidate pool needs checking in the existing scoring before it is changed: the absolute-score rule must still hold, so that restricting the pool cannot make a poor match look confident. The restriction narrows the pool and must not lower the bar.
- The Scan Filter belongs to one scan session (a Draft Addition or a Page Review). It is remembered per user on the device, in local storage, as the last choice, and is not stored by the backend. Changing it mid-session affects only photos taken afterwards.
- The frontend adds a Limit search chip row above the capture area on both scan modes. It reads All sets, All rarities by default and opens a sheet with two multi-select groups, a Sets in this Collection shortcut and a Clear action. Selected values show as wrapping chips, and the sheet applies on close.
- Corrections and the scan feedback rows are recorded exactly as before for a limited scan; the filter changes only which Cards the matcher may offer.

Schema and contract changes

- A slug column on profiles (backfilled, unique), a nullable slug column on Collections (unique within the owner), a Wishlist table with a shared flag, and a Wishlist Item table (Wishlist, Card, quantity, unique per Card). The API contract (`openapi.json`) gains the shared read routes, the Move, the Wishlist routes and the preset and preview calls, and the match and reprint calls gain their restriction parameters; the frontend client is regenerated from it.
- GLOSSARY.md already holds the terms Shared, Slug, Move, Wishlist, Wishlist Preset and Scan Filter, and ADR-0025 records the sharing design. The conventions in `docs/agents/conventions/` apply as for any change.

## Testing Decisions

- A good test checks behaviour through the interface a caller or a user has, and nothing about how the code inside is arranged. It would still pass after a refactor that changes no behaviour, and it fails only when what someone sees or gets back changes.
- There are two seams, one per component, both already in use.
- Backend: the HTTP boundary against a real Postgres, per ADR-0005, extending the existing handler tests and their `support` helpers. No repository-level or service-level tests are added for these features, because the HTTP calls exercise them. This seam covers: sharing and unsharing a Collection and a Wishlist, with Slug creation, its stability while Shared, its clearing and its regeneration; the profile Slug backfill; the public routes as Guest, as a signed-in viewer and as the owner, including the identical not-found for private, unshared and missing targets and the absence of the email; Move with the binder source (Slots emptied, entries equal to Slot counts, Unidentified Slots untouched), the binder destination refused, merging into an existing entry, the over-capacity refusal that changes nothing, the same-Collection and wrong-owner refusals, and selection by filter versus by ids; the Wishlist (set quantity, replace on re-add, clear fulfilled, derived status and its filter, never counting toward Master Inventory); the preset and its preview (own-none rule, skipping existing items, narrowing, unpublished sets excluded, the preview count equal to what is added); the public Wishlist overlay and I have it filter for a signed-in viewer and their absence for a Guest; and the scan match and reprint restrictions, including that a restricted pool does not make a poor match confident.
- Frontend: route and screen tests with the network mocked, following the existing `-collection-page.test.tsx` and `-scan.test.tsx` pattern with the generated client's calls stubbed. This seam covers selection mode and the Select all N matching banner, the Move to sheet with its disabled destinations and the confirm dialog; the Share dialog (warning, switch, Copy, confirmation to turn off); the Wishlist page, Add to Wishlist and the preset sheet with its live count; the shared Collection and Wishlist pages in Guest, signed-in and owner views including the not-found page; and the Limit search sheet, its chips and the Searched note.
- One new Playwright spec: a Guest opens a Shared Collection link, because it is the only unauthenticated page outside the catalog. It follows the guest specs in the existing e2e folder, against the seeded backend, and needs no Clerk credentials beyond what the guest specs already need. No other new end-to-end specs are added.
- Prior art: `collection_handler_test.go`, `inventory_handler_test.go` and `catalog_handler_test.go` with the real Postgres helpers; the binder tests that assert the ADR-0020 invariant after each Slot write; `-collection-page.test.tsx` and `-scan.test.tsx`; `catalog-guest-lock.spec.ts` and `catalog-browse.spec.ts` for guest e2e.

## Out of Scope

- Messaging, contact details, offers or any trade flow between users. Trades are arranged outside the app.
- Showing a viewer which of the owner's cards match the viewer's own Wishlist.
- A live Wishlist Preset that keeps itself up to date, and presets built on rules other than I own none yet.
- Several Wishlists per user.
- Moving cards into a binder, partial-quantity Moves, copying instead of moving, and undoing a Move.
- Changing or regenerating a profile Slug after it is made, for example when the name is edited.
- Making shared pages discoverable: search engine indexing, a directory or search of Shared Collections.
- Storing the Scan Filter on the server or sharing it between devices.
- Any change to how the matcher scores or ranks Cards beyond restricting the pool it searches.
- The Collection type label (binder, box or deck) that GLOSSARY.md mentions and no code implements.

## Further Notes

- The profile name is a single string written once when the profile is created, so the Slug is built from it whole; there is no first or last name to split.
- The shared link shows the owner's real name, which is why sharing is opt-in, the warning comes before the first switch and the pages carry `noindex`. A later change that lets an owner pick a display name would need to revisit ADR-0025.
- Existing profiles have no Slug today, so the migration that adds the column must backfill it before the column can be made required.
- The interaction designs follow the general UI/UX guidance for a bulk action bar, a confirmation before an irreversible action, wrapping filter chips, a Sheet for side panels and a single status message for a changing count. The search found no entry for sharing a link, so the Share dialog is designed from general defaults and not from a database match.
- The tickets are in `issues/`, 13 vertical slices numbered in dependency order, each with its blockers listed.
