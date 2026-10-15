# Binder scanning

Status: ready-for-agent

## Problem Statement

A collector who keeps cards in a physical binder has two problems the app does not solve. First, a Collection only records how many of each Card the user holds, never where a card sits, so the app cannot answer "which page is this card on?" and cannot show the binder the way it looks in the hand. Second, getting a full binder into the app means scanning cards one at a time (Phase 1) or typing them in, although a binder page holds nine or twelve cards and one photo could cover them all. The collector wants to treat a binder as a binder: flip through its pages, see each pocket, photograph a page and have its cards recognised.

## Solution

A Collection can be given a Layout, which turns it into a binder: rows and columns per Page, and a number of Leaves, each Leaf having a front Page and a back Page. The Collection's page shows the binder as something to flip through, one Page at a time, with a grid of Slots that show the Card in each pocket, empty pockets, and pockets holding a card the app could not identify (Unidentified). The user adds or removes a Leaf at the Page they are looking at.

To fill a Page, the user swipes to it and presses scan. They photograph it, drag four corners onto the outer corners of the pocket grid, and the browser flattens the photo and cuts one image per pocket. Each cell is matched with the existing match endpoint, and a review grid with the same rows and columns as the Layout shows the result, with a candidate dropdown on unsure cells. Confirming replaces that Page: each Slot becomes empty, a Card, or Unidentified, and anything the app could not decide becomes Unidentified rather than blocking. The user can also place, clear, replace and move cards by hand, and the Collection's entry list shows where each card sits and jumps to it.

In a binder, the Slots are the truth for quantity. A Card's quantity is the number of its Slots, and the Inventory Entries are kept in step with the Slots inside the same transaction, so Master Inventory, entry search and every Phase 1 read keep working. The Phase 1 "Scan cards" flow still works on a binder by placing each added copy in the first empty Slot.

This spec covers Phase 2 of card scanning. Phase 1 (one card per photo, Draft Addition) is `.scratch/card-scanning/spec.md`.

## User Stories

1. As a collector, I want to give a new or empty Collection a Layout, so that it models my physical binder.
2. As a collector, I want to choose a Layout from presets (2×2, 3×3, 3×4, 4×3) or enter custom rows and columns, so that any common binder size is supported.
3. As a collector, I want rows and columns each limited to 1 to 6, so that the grid stays usable on a phone.
4. As a collector, I want to choose how many Leaves my binder starts with (10 by default), so that the binder matches my real one.
5. As a collector, I want a binder limited to 100 Leaves, so that the screens and requests stay a sensible size.
6. As a collector, I want to change rows and columns until I place the first card, so that I can fix a wrong Layout choice.
7. As a collector, I want rows and columns locked once any Slot is filled, so that placed cards never move because of a grid change.
8. As a collector, I want to remove a Layout while every Slot is empty, so that I can undo a binder I created by mistake.
9. As a collector, I want a Layout to be refused on a Collection that already holds cards, so that I never end up with quantities that no Slot explains.
10. As a collector, I want a binder's `max_card_count` setting replaced by the Layout's Slot count, so that there is one clear capacity.
11. As a collector, I want the binder's card total (Cards plus Unidentified) shown against its Slot capacity, so that I can see how full it is.
12. As a collector, I want to see my binder one Page at a time, so that it feels like turning real pages.
13. As a collector, I want to swipe between Pages, so that I can flip through the binder quickly.
14. As a collector, I want to jump to a Page by number, so that I do not swipe through a hundred Pages.
15. As a collector, I want each Slot to show the Card's image, name and number, so that I can recognise the pocket's content at a glance.
16. As a collector, I want empty Slots to look clearly empty, so that I can see gaps.
17. As a collector, I want an Unidentified Slot to look different from both empty and filled, so that I know a card is there but not named.
18. As a collector, I want to see how many Unidentified Slots the binder holds, so that I can work through them.
19. As a collector, I want to add a Leaf before or after the Leaf I am looking at, so that I can grow the binder at the right position.
20. As a collector, I want adding a Leaf to add both its front and back Page, so that it matches a real binder sheet.
21. As a collector, I want the Pages after an inserted Leaf to be renumbered, so that Page numbers match the physical order.
22. As a collector, I want to remove the Leaf I am looking at, so that I can shrink the binder.
23. As a collector, I want removing a Leaf to be refused when either face holds a card, with the filled Page named, so that I never lose placed cards by accident.
24. As a collector, I want to start a scan from the Page I am looking at, so that the scan knows which Page it is for.
25. As a collector, I want to take the Page photo with the camera, or choose a photo file when the camera is unavailable, so that scanning works on any device.
26. As a collector, I want to drag four corners onto the outer corners of the pocket grid, so that the app finds the pockets without guessing.
27. As a collector, I want the corners to start as a centred rectangle, so that I only have to nudge them.
28. As a collector, I want the app to flatten the photo and cut it into one image per pocket, so that each card is matched on its own.
29. As a collector, I want a small margin cut from every cell, so that the sleeve edge does not confuse the match.
30. As a collector, I want the cell images downsized and sent as JPEG like Phase 1 captures, so that scans stay fast and light on data.
31. As a collector, I want every cell of a Page matched with the same match service as single scans, so that the Page and single scans behave the same.
32. As a collector, I want a review grid with the same rows and columns as my Layout, so that each cell sits where the pocket is.
33. As a collector, I want to see each cell's matched Card in the review grid as soon as it is known, with loading cells marked, so that I can follow progress.
34. As a collector, I want an unsure cell to offer its other candidates in a dropdown that defaults to the top one, so that I can correct it in a couple of taps.
35. As a collector, I want to mark a cell as an empty pocket, so that an empty pocket is not stored as a card.
36. As a collector, I want to mark a cell Unidentified by hand, so that a card the catalog cannot offer is still recorded.
37. As a collector, I want a cell that failed to match or had no good candidate to default to Unidentified without blocking me, so that one bad cell does not stop the Page.
38. As a collector, I want to retry a cell that failed to match, so that a network error does not cost me the Unidentified result.
39. As a collector, I want confirm to wait until every cell has finished matching, so that no cell is left out by accident.
40. As a collector, I want cells that differ from what the Page holds now to be highlighted, so that I can see what confirming will change.
41. As a collector, I want confirming to replace the whole Page, so that re-scanning also corrects earlier mistakes.
42. As a collector, I want a card the server does not know to be stored as Unidentified and reported to me, so that one bad id does not fail the Page.
43. As a collector, I want confirm to be repeatable without side effects, so that a retry after a lost response is safe.
44. As a collector, I want the review to be lost if I leave or refresh, and to be asked before leaving after edits, so that I do not discard work by accident.
45. As a collector, I want to place a card in an empty Slot by searching the catalog, so that I can fill pockets without a photo.
46. As a collector, I want to clear a filled Slot, so that I can remove a card from the binder.
47. As a collector, I want to replace the card in a filled Slot, so that I can correct a wrong card.
48. As a collector, I want to place a Card into an Unidentified Slot, so that I can resolve it.
49. As a collector, I want to mark any Slot Unidentified by hand, so that I can record a card I cannot find in the catalog.
50. As a collector, I want to move a card to another Slot, on this Page or another, so that I can sort my binder.
51. As a collector, I want moving onto a filled Slot to swap the two cards, so that I can reorder without a free pocket.
52. As a collector, I want each Slot change to keep the Card's quantity right, so that totals never disagree with the binder.
53. As a collector, I want the Phase 1 "Scan cards" flow to keep working on a binder by placing every added copy in the first empty Slots, so that single-card scanning is not lost.
54. As a collector, I want that flow to tell me when the binder has too few empty Slots, so that I know to add a Leaf.
55. As a collector, I want quantity inputs on a binder replaced by Slot actions, so that the quantity cannot drift from the Slots.
56. As a collector, I want the Collection's entry list to show each card's positions (Page, row, column), so that I can find a card.
57. As a collector, I want tapping a position to open the binder on that Page with the Slot highlighted, so that I see where it is.
58. As a collector, I want Master Inventory to count binder cards like any other Inventory Entry, so that my totals across Collections stay right.
59. As a collector, I want Unidentified Slots to count toward no Card's quantity and no Master Inventory, so that totals only contain identified cards.
60. As a collector, I want deleting a binder Collection to delete its Slots, so that nothing is left behind.
61. As a signed-in user, I want binder actions to need my login and only act on my own Collections, so that nobody can read or change another user's binder.
62. As the site owner, I want the accuracy of binder-cell matching measured on real Page photos, so that I know whether single-card accuracy holds for pockets.
63. As the site owner, I want the cell margin to be one tunable constant, so that I can adjust it after seeing the accuracy results.
64. As a developer, I want the page-scan screen to reuse the existing match endpoint, so that no new matcher or batch endpoint is needed.

## Implementation Decisions

**Domain**
- New glossary terms, already written: Layout, Leaf, Page, Slot, Unidentified. The Collection entry now says a Collection may carry a Layout.
- One physical binder is one Collection. A Collection without a Layout (a box, a deck) is unchanged and tracks quantity only.
- A Slot is in exactly one of three states: empty, holding a Card, or Unidentified. A Slot holds at most one physical card, so two copies of a Card take two Slots.
- In a binder, the quantity of a Card is the number of Slots holding it. Unidentified Slots belong to no Card, to no Inventory Entry and to no Master Inventory.
- A Leaf has a front Page and a back Page. Leaves are ordered, and Pages are numbered in turning order, so inserting or removing a Leaf renumbers the Pages after it.

**Layout and binder lifecycle**
- A Layout is rows per Page, columns per Page, and a Leaf count. Rows and columns are each 1 to 6, the Leaf count is 1 to 100, and a new Layout starts with 10 Leaves unless the user chooses otherwise. Presets are 2×2, 3×3, 3×4 and 4×3, plus custom.
- A Layout can only be set on a Collection that holds no Inventory Entries. Setting it clears the Collection's `max_card_count` to 0, and that field is not offered on binders. Capacity is Leaves × 2 × rows × columns, and the card total shown against it is Slots holding a Card plus Unidentified Slots.
- Rows and columns can be changed while every Slot is empty and are locked once any Slot is filled or Unidentified. A Layout can be removed only while every Slot is empty.
- Adding a Leaf happens before or after a given existing Leaf, and the new Leaf has two empty Pages. Removing a Leaf removes both of its Pages and is refused if either Page has any non-empty Slot; the refusal names the Page. Growing is bounded by the 100-Leaf limit, and the last Leaf cannot be removed.
- Deleting the Collection deletes its Slots.

**Quantity invariant**
- Inventory Entries stay real rows and are updated in the same transaction as every Slot write, so that an entry's quantity always equals the number of Slots holding the Card and an entry with zero Slots is removed. Master Inventory, the entry search and capacity checks need no change.
- The existing bulk quantity update refuses binder Collections, with a machine-readable reason, because quantity there can change only by placing or clearing Slots. The frontend shows Slot actions instead of quantity inputs on a binder.
- This choice is recorded in an ADR written with this spec (ADR-0020), not in the glossary.

**Slot reads and writes**
- Reading a Page returns its Slots in grid order, each with its state and, for a Card, the same card summary the match endpoint returns (id, name, expansion set, card number, hosted image address). The binder's Layout, Leaf count, Unidentified count and card total are available alongside.
- Single-Slot operations: place a Card, clear, replace, mark Unidentified. Move takes a source and a target Slot, on any Page of the same Collection: an empty target takes the card, a filled target swaps the two. Each operation, including a swap, is atomic and keeps the Inventory Entries in step.
- The Collection entry list returns, for each Card in a binder, its Slot positions (Page, row, column). Cross-Collection "where is this card" is out of scope.

**Replace Page (confirm)**
- One authenticated operation replaces the contents of a whole Page. The request names the Page and carries one state per Slot in grid order: empty, a Card id, or Unidentified. It is idempotent on retry.
- A Card id the server does not know is stored as Unidentified and reported as such for that Slot. A Page that does not exist, or a request whose Slot count does not match the Layout, is rejected as a whole and changes nothing. A capacity failure cannot occur, since the Slots are fixed.
- The response gives a result per Slot (stored as sent, or stored Unidentified with the reason), and the Inventory Entries of every affected Card are updated in the same transaction.

**Phase 1 flow on a binder**
- The Phase 1 review step confirms by placing each added copy of a Card in the first empty Slots in Page order, one Slot per copy, instead of calling the bulk update. If there are not enough empty Slots, the remaining copies stay in the draft with the reason "not enough empty Slots", the same way a capacity decline does today.

**Page scan (frontend)**
- Scanning starts from the Page being viewed, so the Page number is known. The photo comes from the camera or the file picker through the same frame-source seam as Phase 1.
- The user drags four corners onto the outer corners of the pocket grid, not the Page edge. The corners start as a centred rectangle. Automatic corner detection is deferred.
- The browser flattens the quadrilateral to a rectangle, cuts it into equal cells (Layout rows × columns), crops each cell inward by a fixed margin (first value 3 percent per side, one tunable constant), downsizes to about 1024 pixels on the long side, and uploads each as a JPEG.
- Each cell is sent to the existing match endpoint. Several run at a time and the rest queue, as in Phase 1. A Page of 9 to 12 cells is within the existing per-user rate limit (300 a minute, burst 100), so no new endpoint or batch route is added.
- The review grid has the Layout's rows and columns. Each cell is one of: confident Card, unsure Card with candidate dropdown (default top), loading, failed, no good candidate, or marked empty. Every cell defaults to occupied. The user marks empty pockets by hand. A failed cell or a cell with no good candidate defaults to Unidentified at confirm, and failed cells offer Retry. Confirm waits until no cell is loading.
- Cells that differ from the Page's current Slots are highlighted. Confirming sends the Replace Page operation. The review lives in memory only and is not stored; leaving after edits asks for confirmation.

**Binder view (frontend)**
- The binder is shown one Page at a time on every screen size, with swipe and jump-to-Page. A two-Page spread is deferred.
- Add Leaf (before or after) and Remove Leaf act on the Leaf of the Page being viewed.
- The entry list shows positions for each Card. Tapping one opens the binder at that Page with the Slot highlighted.

**Delivery order**
1. Binder foundation (backend): Layout, Leaves, Pages, Slots, add and remove Leaf, the quantity invariant, bulk update refusing binders.
2. Binder view (frontend): create a binder, swipe through Pages, add and remove a Leaf. Depends on 1.
3. Manual Slot editing: place, clear, replace, move, mark Unidentified, backend and frontend.
4. Replace Page operation (backend), then the page-scan screen (frontend), as two tickets.
5. Phase 1 flow on binders.
6. Find a card: positions in the entry list with jump.
7. Binder-cell accuracy check, run alongside ticket 07 of Phase 1 and including the cell margin. It does not block 1 to 6.

**Release**
- There is no feature flag for Phase 2. It shares Phase 1's production blockers (the embedded catalog and the Gemini key).

**Privacy**
- Unchanged from Phase 1: cell photos go to the embedding provider on the terms the Phase 1 spec accepted, and are not stored by Cardstack.

## Testing Decisions

A good test exercises external behavior only: a request in and a response out at the backend, a rendered route and user interaction at the frontend. It does not assert how Slots are stored, how Leaves are renumbered internally, or which helper computed a cell rectangle.

Seams, preferring existing ones:
- **Backend: the Huma HTTP boundary against real Postgres** (ADR-0005), as for the Phase 1 match endpoint and the catalog tests. It covers Layout creation and its limits and refusals, Leaf insert and remove with Page renumbering, Slot placement, clear, replace, move and swap, the Replace Page contract (including Unidentified, an unknown Card id, and a mismatched Slot count), the quantity invariant after every write, the bulk update refusing a binder, and the entry list's positions.
- **Frontend: the binder route and the page-scan screen, rendered with the generated API client mocked** (the existing Vitest and Testing Library pattern). The frame source (camera or file picker) and the flatten-and-cut step are faked in tests, since jsdom has no canvas. This covers swiping and jumping between Pages, adding and removing a Leaf, manual Slot actions, the review grid (loading, failed, unsure dropdown, marked empty, Unidentified defaults, difference highlighting), confirm, and the Phase 1 review step placing copies in the first empty Slots.
- **One Playwright spec** on the file-picker path with stubbed match responses, signed in through the existing Clerk e2e setup: create a binder, scan a Page, review, confirm. The cell cutting runs for real in Chromium.
- **The binder-cell accuracy check is an offline script and report**, not a CI test, built like Phase 1's ticket 07: photos of real binder Pages, cell top-1 and top-5, and the tuned cell margin.

Prior art: the Phase 1 bulk update and match endpoint tests, the catalog and Collection entries search tests, the Phase 1 scan route tests and its Playwright spec, and the catalog-admin form tests.

## Out of Scope

- Server-side detection of cards, pockets or Pages. Phase 2 relies on user-placed corners and a declared Layout.
- Automatic corner detection. It can be added later as a refinement of the corner step.
- A two-Page spread view of an open binder.
- Converting a populated Collection into a binder, and turning a binder back into a plain Collection once it holds cards.
- Reordering Leaves, and non-uniform binders (different grids on different Pages, single-sided sheets).
- Cross-Collection "where is this card" search.
- Persisting a page-scan review across a refresh.
- Binder-wide scanning in one action (a sweep through every Page), and live scanning.
- Print finish or Card Variant detection (ADR-0006), condition grading, and foreign print editions (Phase 1's scope notes still apply).
- Sharing a binder, and a feature flag for Phase 2.

## Further Notes

- The matcher's accuracy on real phone photos is still unmeasured (Phase 1 ticket 07 is open, and ADR-0018 says so). Binder cells are harder than single-card photos because of sleeves, glare and cell edges, so the Phase 2 accuracy check may recommend the OCR hybrid from the Phase 1 spec. Phase 2 is built in parallel with ticket 07 by choice, and its tickets 1 to 6 do not depend on the check.
- Reprints that tie are non-confident by design and appear as an unsure cell with a dropdown, as in Phase 1.
- The Phase 1 spec's Out of Scope section describes Phase 2 in outline. After this spec is accepted, that section should point here, and the two statements about "one slot per physical copy being undecided" are now settled: one Slot per physical copy, with Slots as the truth.
- Setting a Layout clears `max_card_count`, so a Collection's capacity field is hidden on a binder. A future request to cap a binder below its Slot count would reopen this.
