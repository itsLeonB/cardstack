# Scan enhancements

Status: implemented on `feat/scan-enhancements` (tickets 01–13 done), not yet merged to `main`

## Problem Statement

The matcher is provisional and nobody can see where it fails. When a user changes what the matcher chose, that knowledge is lost the moment they confirm, so the operator cannot tell which Cards are mismatched often, which photos are handled badly, or whether the confident threshold is right. Two failures are already known. A blank pocket, or a photo with no card in it, still comes back with its five nearest Cards and no way to say "nothing here", so a binder cell defaults to a wrong Card or to Unidentified instead of Empty, and single card scan has no way to say it either. And one artwork printed in several Expansion Sets (a Reprint) can fill all five candidates with the same Card's prints, so the print the user actually holds is never offered, and the user cannot find it however much they scroll the top five.

## Solution

The app records what the matcher said and what the user finally chose, for every Scan and every Page Review cell they confirm. A confirmation records only that it was confirmed. A Correction (a different Card, Empty, Unidentified) also keeps the photo, so the operator can later look at the photos the matcher gets wrong. Nothing about this is visible to the user beyond the result they already see.

The matcher can now say "no card here". Below a configurable, low similarity it returns an empty Detection with no candidates; a binder cell then defaults to Empty, and a single card scan shows that no card was found. Single card scan also gains the two marks a binder cell already has, Empty and Unidentified, worded for a single photo. A marked Scan adds nothing when the Draft Addition is confirmed.

Every candidate menu, in the Draft Addition and in a Page Review cell, gains "Other prints". It lists the Reprints of the Cards the matcher offered, five at a time, with "Load more" while there are more, so a user who cannot find their print among the top five can keep looking among the same Card's other Expansion Sets. It is offered on confident results too.

Detecting a promo stamp and reading the set code from the photo are researched and specified separately; the data this spec collects is what decides whether they are worth building.

## User Stories

1. As an operator, I want every confirmed Scan to leave a record of what the matcher said and what the user chose, so that I can see where the matcher is least reliable.
2. As an operator, I want a Scan the user accepted without touching to be recorded as a confirmation, so that I can compute a correction rate per Card and not only a count.
3. As an operator, I want a confirmation to store no photo, so that the bucket holds only the photos worth looking at.
4. As an operator, I want a Scan where the user picked another of the matcher's candidates recorded as a Correction with its photo, so that I can see which Cards the matcher ranks wrongly.
5. As an operator, I want the position the user's choice held in the list the matcher offered, so that I can tell a near miss from a candidate that was barely in the list.
6. As an operator, I want a Scan where the user picked a Reprint from "Other prints" recorded as its own kind, so that I can tell how often the Reprint list finds the right print.
7. As an operator, I want a Scan the user marked Empty, when the matcher saw a card, recorded with its photo, so that I can tune the empty threshold.
8. As an operator, I want a Scan the user marked Unidentified, when the matcher offered candidates, recorded with its photo, so that I can tune detection.
9. As an operator, I want the matcher's whole answer stored as the client received it, scores included, so that I can see what a Correction looked like from the matcher's side.
10. As an operator, I want to see whether the matcher was confident when a Scan was confirmed untouched, so that I can find unsure results that were right.
11. As an operator, I want a record only when a Scan or cell is confirmed, so that changing a choice back and forth, or abandoning a review, records nothing.
12. As an operator, I want a Removed Scan to record nothing, so that dropping a mistaken photo is not mistaken for a matcher failure.
13. As an operator, I want a match that failed with an error and was left Unidentified to record nothing, so that an outage does not read as a matcher judgment.
14. As an operator, I want a Card the user placed after a failed match recorded with its photo, so that I get a labelled photo of a case the matcher never saw.
15. As an operator, I want each record to say whether it came from a single card scan or a binder Page, so that I can analyse the two separately.
16. As an operator, I want a binder record to carry its Page, row and column, so that I can find the cell in the Page photo.
17. As an operator, I want a binder Correction to keep the whole Page photo with the grid and the four corners used to cut it, so that any cell can be cut again and tried against a changed matcher.
18. As an operator, I want one stored Page photo shared by every Correction on that Page, so that a Page with many Corrections uploads once.
19. As an operator, I want a card-scan Correction to keep the photo that was matched, so that I see what the matcher saw.
20. As an operator, I want records and photos kept until I delete them by hand, so that I can study old failures after a model change.
21. As an operator, I want the photos in a bucket that is not served publicly, so that a user's binder is never reachable by address.
22. As an operator, I want records kept when a Collection is deleted, so that deleting a binder does not erase the evidence of what the matcher got wrong.
23. As an operator, I want to read the records with SQL, so that I can ask any question without a screen being built first.
24. As an operator, I want feedback recording to be switched off when the private bucket is not configured, so that a local or preview environment works without it.
25. As a collector, I want confirming my Draft Addition or Page Review never to wait for or fail because of feedback recording, so that analytics can never lose my cards.
26. As a collector, I want a failed upload or a failed feedback call to be silent, so that I am not asked to fix something I cannot see.
27. As a collector, I want my correction photos to be uploaded straight from my browser with a short-lived address, so that a slow upload does not hold up the app.
28. As an operator, I want the API to refuse a photo that is not under the caller's own prefix, is not a JPEG, or is over the size cap, so that a record cannot point at someone else's object or at junk.
29. As an operator, I want an object that fails those checks deleted and its record refused, so that junk does not stay in the bucket.
30. As an operator, I want a batch of records limited to 100, and the calls that mint upload addresses to count against the per-user rate limit, so that one caller cannot flood the bucket.
31. As a collector, I want a binder cell the matcher saw no card in to default to Empty, so that an empty pocket is not filled with a wrong Card.
32. As a collector, I want that Empty default visible on the cell before I confirm, so that a card the matcher missed is not dropped silently.
33. As a collector, I want to change a cell the matcher called Empty to a Card or Unidentified, so that a wrong empty call costs me a tap.
34. As an operator, I want the empty rule to be a setting I can change without a frontend release, so that I can tune it on real photos.
35. As an operator, I want the empty rule off until I set it, so that an untuned guess never drops a real card from a binder.
36. As an operator, I want the empty rule to apply to single card scans and binder cells alike, so that the two behave the same.
37. As a collector scanning single cards, I want a photo with no card in it to say "no card found", so that I do not get five meaningless suggestions.
38. As a collector scanning single cards, I want to Retry or Remove a "no card found" Scan, so that I can take the photo again.
39. As a collector scanning single cards, I want to mark a Scan Empty when the photo shows no card, so that the app learns the matcher should have said so.
40. As a collector scanning single cards, I want to mark a Scan Unidentified when it is a card but none of the candidates is right, so that I can move on and the app learns the matcher missed it.
41. As a collector scanning single cards, I want the Empty and Unidentified marks worded for a single photo, so that I am not shown Slot words that do not apply.
42. As a collector scanning single cards, I want a marked Scan to add nothing when I confirm, so that no wrong Card enters my Collection.
43. As a collector scanning single cards, I want the confirm summary to tell me how many Scans were set aside, so that nothing disappears without my knowing.
44. As a collector, I want a marked Scan to be changeable back to a Card before I confirm, so that a mistaken mark costs nothing.
45. As a collector, I want to open "Other prints" from any Scan or cell, so that I can look for my print among the same Card's other Expansion Sets.
46. As a collector, I want "Other prints" offered even when the matcher was confident, so that a confident wrong print can still be fixed.
47. As a collector, I want the Reprints to be of the same name and the same artwork, so that I am not shown every Card that merely shares a name.
48. As a collector, I want the Reprints of the matcher's top candidate listed first, then those of the next one, so that the most likely Card's prints come first.
49. As a collector, I want the Reprints within one Card to be ordered by closeness of artwork, then newest Expansion Set first, so that the likeliest print leads.
50. As a collector, I want a Reprint that is also in the matcher's top five never to be listed again, so that I do not see the same Card twice.
51. As a collector, I want a Reprint reachable from two candidates to appear once, so that the list is short.
52. As a collector, I want the Reprints five at a time with "Load more", so that a Card with forty prints does not bury the menu.
53. As a collector, I want "Load more" to stay until the Reprints run out and then disappear, so that I know when I have seen them all.
54. As a collector, I want a Reprint I pick to become the choice of that Scan or cell, so that picking one works like picking a candidate.
55. As a collector, I want a Reprint I picked to still be my choice after I leave and come back, so that I do not have to find it again.
56. As a collector, I want "Other prints" to say so plainly when there are none, so that I know the matcher offered everything it had.
57. As a collector, I want Reprints never to include a Card from an unpublished Expansion Set, so that nothing hidden shows up in a menu.
58. As a collector, I want a Card without a hosted image never to be offered as a Reprint, so that I am not offered something the app cannot show.
59. As a collector, I want "Other prints" to load only when I open it, so that scanning stays as fast as it is.
60. As a collector scanning single cards, I want my resolved Scans, with their photos, to survive a refresh, so that a Correction I made is not lost to a reload.
61. As a collector, I want my saved photos kept per signed-in user, so that someone else using the same browser never sees them.
62. As a collector, I want my saved photos removed when I confirm, clear or discard the Draft Addition, so that the browser does not fill up.
63. As a collector using a browser without IndexedDB, I want scanning to keep working from memory, so that a missing store does not break the flow.
64. As a collector scanning a binder Page, I want the whole Page photo kept with my pending review, so that a Correction on a resumed review still has its photo.
65. As a collector, I want a pending review's grid and corners kept with it, so that the cells stay reproducible.
66. As an operator, I want a Correction whose photo is no longer available recorded without an image, so that the choice is not lost because the photo was.
67. As an operator, I want the Reprint art distance to be a setting, so that I can retune it without a release.
68. As an operator, I want an offline script that prints the top similarity of each cell of a set of photos, so that I can set the empty threshold from my empty, sparse and full Page photos.
69. As an operator, I want the same script to print how far apart same-name Cards sit in the catalog, so that I can set the Reprint art distance from real groups.
70. As a collector, I want the confident rule, the candidate count and every existing scan behaviour unchanged for photos that are neither empty nor a Reprint case, so that these enhancements add to scanning and change nothing else.
71. As a collector, I want the Draft Addition and the Page Review to confirm exactly as before, so that I cannot be blocked by anything new.
72. As a collector, I want a Guest or an expired token to be refused by the new routes like the existing scan routes, so that the same rules hold everywhere.

## Implementation Decisions

**Domain**
- New glossary terms, already written: Detection, Correction, Reprint. The Scan entry now says a Scan can be marked Empty or Unidentified and is then set aside on confirm, and Unidentified now covers a Scan as well as a Slot.
- A Detection is what the matcher returned for one photo: the ranked Cards, whether it was confident, and whether it saw no card. A Detection with no candidates because the matcher found none is a Detection; a failed match is not, it is an error.
- ADR-0022 (corrections are recorded with their photo in a private bucket, and the client supplies the Detection) and ADR-0023 (Reprints are found by name and catalog art, not by the photo's score) record the two decisions that are hard to reverse. This spec follows both and does not repeat their reasoning.

**Match response**
- The match response gains an `empty` flag. When the top similarity is below a new setting `MATCH_EMPTY_THRESHOLD`, the response is `empty: true`, `confident: false` and no candidates.
- The setting defaults to 0, which disables the rule. It must be at least 0 and below the confident threshold; a value outside that fails the boot, like the existing match settings.
- Everything else about the response is unchanged: the confident rule, the margin, a confident result trimmed to its top candidate, and up to five candidates otherwise. No second "unidentified" threshold is added; the stored scores decide later whether one is needed.
- The response is stored by the client verbatim as the Detection, so a field added later is recorded without a schema change.

**Reprints**
- `GET /scan/reprints` takes up to five `cardIds` and an `offset` and returns a page of Card summaries (the existing shape) and `hasMore`. The page size is fixed at five. It sits behind the same authentication and rate limit as the match route.
- The rule is ADR-0023's. For each id, in the order given, the API finds Cards with the same normalized name (case and whitespace folded, otherwise exact), excluding every id sent, whose catalog embedding is within `MATCH_REPRINT_DELTA` (a maximum cosine distance, default 0.05) of the source Card's catalog embedding. When a Card has several embeddings, the nearest one counts.
- The order is the source's position in the ids given, then closeness to the source, then newest Expansion Set first. A Card reachable from two sources appears once, at its first place. The same ids give the same order on every call, so an offset is stable with no server state.
- Cards without an embedding and Cards in unpublished Expansion Sets are left out. Unknown ids are dropped without an error, more than five ids are refused, and a negative offset is refused.
- A lookup by normalized name needs an index on the name expression the query uses.
- A confident result sends its one candidate id. The client sends the ids it holds, so Draft Addition rows and Page Review cells use the route the same way.

**Recording**
- A new table holds one row per confirmed Scan or cell. It records the user, the source (`card_scan` or `binder_page`), the Collection (no foreign key, so deleting a Collection keeps its rows), and for a binder the Page, row and column. It records a kind: `confirmed`, `switched`, `reprint`, `empty`, `unidentified` or `placed`.
- It also records the Detection as JSON (size-capped), the chosen Card (none for Empty and Unidentified), the chosen Card's rank (its position in the list the matcher offered, or its position in the Reprints list counted across pages), a source for that rank (`top` or `reprint`), an image key, and for a binder the grid and the four corners. A timestamp is set by the server.
- The kind is derived on the client from the Detection and the final choice: the Card the matcher put first is a confirmation, whether or not the matcher was confident; another candidate is `switched`; one picked from "Other prints" is `reprint`; marking Empty when the Detection offered a Card is `empty`; marking Unidentified when the Detection offered a Card is `unidentified`; keeping what the Detection already was (Empty on an empty Detection, Unidentified on a Detection with no candidates) is a confirmation; placing a Card where the Detection was empty or offered none, or where there was no Detection, is `placed`. A Scan that is a failed match left Unidentified, a Removed Scan, and anything in a review that is never confirmed, record nothing.
- Only a Correction carries an image key. A Correction whose photo is gone is recorded with no key. The server refuses a confirmation that carries a key and a Correction whose kind and chosen Card disagree, such as `empty` with a Card.
- Two calls serve it. A first call asks for upload addresses and returns, for each requested photo, a server-chosen key under the caller's own prefix, a presigned PUT address that fixes the content type and expires shortly, and its expiry. A second call posts the batch of rows, up to 100, and the API checks every key before storing any row: under the caller's prefix, the first bytes are a JPEG, and the object is under the size cap. An object that fails is deleted and its row refused.
- Several rows in one batch may share a key, because a Page photo is shared by every Correction on that Page.
- Both calls are authenticated and the first is counted against the per-user rate limit. They are separate from the placement call and run after it succeeds. The client never waits on them and ignores their failure.
- Photos live in a new private R2 bucket with its own scoped credential and CORS for the frontend origin. When that bucket is not configured, the upload call refuses clearly and the client records nothing. `ObjectStore` gains a way to read an object's size for the check.
- Records and photos have no expiry and no deletion path in the product; the operator deletes by hand.

**Frontend**
- Single card scan: a Scan gains two marks, worded for a photo: Empty means no card was in the photo, Unidentified means it is a card but none of the candidates fits. A server-flagged empty shows "no card found" with Retry and Remove, and the marks. A marked Scan stays in the list, is not counted into the addition, and the confirm summary says how many were set aside. This applies to the existing candidate picker; the binder's pill and menu keep their layout.
- Binder Page Review: a cell whose Detection is empty defaults to the Empty mark and shows it on its pill. Everything else about the pill, the menu and the difference ring is unchanged.
- "Other prints" is an entry in the candidate picker and in a cell's menu. Opening it fetches the first page; "Load more" fetches the next while `hasMore` holds. A Reprint picked becomes the choice and is remembered as a Reprint with its position, so the kind and rank survive a refresh.
- The Draft Addition's resolved Scans, each with its photo and Detection, are kept in IndexedDB keyed by user and Collection, reusing the Page Review store's pattern. Pending and failed Scans stay in memory. The store is cleared when the addition is confirmed, cleared or a Scan is removed, and the app works from memory when IndexedDB is unavailable.
- A Page Review also keeps the Page photo with its grid and corners, so a resumed review can still upload.
- After the placement call succeeds, the client builds the rows, asks for upload addresses for the Corrections' photos, uploads, and posts the rows. Any failure is dropped.

**Tuning**
- An offline script, built like the Phase 1 accuracy script, runs a folder of photos through the matcher and prints the top similarity of every binder cell, and the pairwise catalog distances inside same-name groups. It is what sets `MATCH_EMPTY_THRESHOLD` from the empty, sparse and full Page photos and `MATCH_REPRINT_DELTA` from real groups. It is not a CI test.

## Testing Decisions

A good test exercises external behaviour only: a request in and a response out at the backend, a rendered screen and user interaction at the frontend. It does not assert how the kind is derived internally, how a key is built, or which helper ordered the Reprints.

Seams, preferring existing ones:
- **Backend: the Huma HTTP boundary against real Postgres** (ADR-0005), as for the match route and the catalog tests. It covers the empty flag (below the setting, at it, the setting disabled, an invalid setting failing the boot), the Reprint route (name folding, the art test, exclusion of the sent ids, cards without an embedding, unpublished sets, ordering across sources, a Card reachable twice, paging and `hasMore`, a stable offset, the id and offset limits, unknown ids), and the feedback calls (kind and field consistency, the batch limit, a key outside the caller's prefix, a non-JPEG, an oversized object deleted and its row refused, shared keys, no key on a confirmation, another user's rows invisible, the rate limit, and the feature off with no bucket). Embeddings are seeded directly and the bucket uses the generated `ObjectStore` mock.
- **Frontend: the scan routes and the binder page-scan screen, rendered with the generated API client mocked** (the existing Vitest and Testing Library pattern), with IndexedDB faked in memory as in the Page Review tests. It covers the new marks and wording on a Scan, a server-flagged empty, the set-aside count on confirm, the Empty default on a binder cell, "Other prints" (first page, Load more until it disappears, none found, a picked Reprint becoming the choice and surviving a remount), the Draft Addition surviving a remount with its photos for a given user only, cleanup on confirm and clear, the no-IndexedDB fallback, and the recording after confirm (the rows built for each kind, no rows for a Removed Scan or a failed match left Unidentified, uploads only for Corrections, a shared Page photo uploaded once, and a failed upload or feedback call leaving the confirmed result untouched).
- **The tuning script is an offline report**, not a test.
- No new Playwright spec; the existing page-scan spec keeps passing with the new response fields stubbed.

Prior art: the Phase 1 match and bulk update tests, the Phase 2 Replace Page and Page Review store tests, the catalog-admin upload tests (presigned PUT and the signature check), and the embedding repository tests.

## Out of Scope

- Detecting the promo stamp and narrowing the search to promo sets, and reading the set code from the photo's corner. They need a model that reads regions of a photo and photos to check it against; this spec's data feeds the decision, and they get a research note and their own spec.
- Extracting the admin from the user app into a separate app, and user-supplied embeddings (an admin reviewing a corrected photo and embedding it as an extra image of the chosen Card). They are separate specs, in the order this spec, then the admin extraction, then user-supplied embeddings.
- Any screen for the recorded data. The operator reads it with SQL.
- A second "unidentified" threshold that defaults a low-scoring cell to Unidentified.
- A notice on the scan screens that photos are kept, expiry or automatic deletion of records and photos, and account deletion. The record and photo deletion is manual until account deletion exists.
- Manual catalog search inside a review, and making pending or failed Scans in the Draft Addition survive a refresh.
- Reprints beyond the Reprints: continuing a list into nearest neighbours of the photo.
- Syncing a Draft Addition or a pending review between devices.
- Fuzzy name matching for Reprints.

## Further Notes

- `placed` is part of the model, but nothing in today's review can produce it: the menu offers only the matcher's candidates, "Other prints" and the two marks, and a failed match is retried, not filled by hand. It becomes reachable when a manual search lands in the review.
- A Correction whose photo is gone is recorded without an image; with the Draft Addition now persisting photos, that happens only without IndexedDB or after the browser cleared it.
- The card-scanning spec and the binder-scanning spec both say photos are processed in memory and never stored. ADR-0022 supersedes those statements; the specs are left as written.
- Reprints that tie are non-confident today (the margin), so the top five already holds one Card's prints in that case; "Other prints" is the fallback for when the right print is not among them and for a confident wrong print.
- The art distance of 0.05 and the disabled empty rule are deliberately provisional, like the confident threshold (Phase 1 ticket 07). The tuning script is how they stop being guesses.
- A catalog change while a user pages through Reprints (a set published or a Card added) can shift the offset; paging is for a few minutes at most, and the user simply sees a Card twice or skips one.
