# User-supplied embeddings

Status: ready-for-agent

Depends on the scan enhancements spec (`.scratch/scan-enhancements/spec.md`), which records each Correction with its photo, and on the admin app extraction spec (`.scratch/admin-app-extraction/spec.md`), which gives the Admin its own app. Decisions are recorded in ADR-0027, which this spec follows and does not repeat in full.

## Problem Statement

The matcher compares a phone photo with one catalog image per Card. A phone photo of a real card differs from the catalog image in light, angle, sleeves and glare, so the right Card sometimes scores low or loses to another print, and the user has to correct it. Each time a user does, Cardstack already keeps the photo and the Card they chose (ADR-0022), but nothing learns from it: the next person who photographs the same print meets the same miss.

## Solution

An Admin reviews the photos kept from Corrections in the Admin app. For each one they see the user's photo (for a binder cell, the cell cut from the stored Page photo) next to the chosen Card's catalog image and the scores the matcher gave. They approve the ones that are a clear photo of that print, and reject the rest. An approved photo is embedded as an extra image of the Card the user chose (a User-supplied Embedding, GLOSSARY.md), and the matcher starts using it at once. A photo the Admin has not approved never reaches the matcher.

Because user photos can lift one print of a Reprint family above its siblings, a result whose top Card has a Reprint is never confident any more, so the user still picks the print. The scan screens tell users that photos of corrected scans may be used to improve matching.

## User Stories

1. As an Admin, I want a queue of the Corrections whose photos are waiting for review, so that I can work through them in one place
2. As an Admin, I want to see the user's photo next to the chosen Card's catalog image, so that I can judge whether they are the same print
3. As an Admin, I want to see the scores the matcher gave for that photo, so that I can see how far it was from a match
4. As an Admin, I want a binder cell shown as the cut the matcher would embed, so that I approve exactly what will be embedded
5. As an Admin, I want a binder cell cut with the same margin the user app used when it matched, so that the stored vector looks like a real query
6. As an Admin, I want to approve a photo, so that the next photo of the same print scores higher
7. As an Admin, I want to reject a blurry, dark, wrong or unrelated photo, so that it never reaches the matcher
8. As an Admin, I want a rejected photo to stay rejected, so that I am not shown it again
9. As an Admin, I want an approved photo to leave the queue, so that I only see work that is left
10. As an Admin, I want to see who reviewed a photo and when, so that a decision can be traced
11. As an Admin, I want the queue to include Corrections recorded before this feature shipped, so that the photos already collected are not wasted
12. As an Admin, I want the queue to hold only Corrections that carry a photo and a chosen Card, so that I never review something I cannot approve
13. As an Admin, I want a user who switched to another candidate to be eligible, so that the most common correction teaches the matcher
14. As an Admin, I want a user who picked a print from "Other prints" to be eligible, so that the photo of that exact print counts for that print
15. As an Admin, I want a Card placed where the matcher offered nothing to be eligible, so that a hand-placed Card teaches the matcher once such placements exist
16. As an Admin, I want Corrections marked Empty or Unidentified, and confirmations, left out of the queue, so that I do not review photos with no chosen Card
17. As an Admin, I want a photo approved for one print to teach only that print and not its Reprints, so that the print the user chose is the print that improves
18. As an Admin, I want an approval to fail cleanly when the embedding service fails, so that no half-approved photo is left behind
19. As an Admin, I want to retry an approval that failed, so that a passing outage does not lose the review
20. As an Admin, I want a second Admin who approves the same photo at the same moment to be refused, so that one photo is never embedded twice
21. As an Admin, I want the photos shown through short-lived links, so that the private bucket stays private
22. As an Admin, I want review to be reachable only from the Admin app, so that an Admin's token from the user app cannot review
23. As a user, I want a Card I corrected to be found more easily next time, so that scanning needs fewer corrections
24. As a user, I want my photo to be used only after an Admin has looked at it, so that nothing I upload is trusted into the catalog by itself
25. As a user, I want to be told that photos of corrected scans may be used to improve matching, so that I know what happens to my photos
26. As a user, I want a scan of a Card that has Reprints to ask me which print, so that I am never told a wrong print with confidence
27. As a user, I want a Card with no Reprints to still be matched confidently when it is clearly the top result, so that scanning stays fast
28. As a user, I want my photo never to appear as a Card's picture in the catalog, so that my photo is not shown to others
29. As the operator, I want a user photo to be one extra embedding of a Card and never a catalog image, so that the catalog stays curated
30. As the operator, I want every user-supplied embedding to name the Correction it came from, so that I can find and remove it
31. As the operator, I want to remove every user-supplied embedding with one statement, so that I can roll the feature back
32. As the operator, I want review state kept with the Correction record, so that nothing else has to be joined to know what is pending
33. As the operator, I want the matcher's thresholds and margin to stay as they are, so that this change does not move what the tuning script settled
34. As the operator, I want catalog-only questions, such as how alike two Cards' art is, to ignore user photos, so that a user photo is never taken for a Card's art
35. As the operator, I want the cut photo of every approval kept, so that a future model switch can re-embed without cutting again
36. As the operator, I want photos kept until I delete them by hand, so that behaviour matches ADR-0022

## Implementation Decisions

**Domain**
- User-supplied Embedding is already in GLOSSARY.md, and Correction now says an Admin can approve its photo as one. ADR-0027 records the decisions that are hard to reverse: approved photos become embeddings, the source value carries no foreign key, and a Card with a Reprint is never confident. This spec follows it and does not repeat its reasoning.
- Eligible Corrections are those of kind `switched`, `reprint` or `placed` that have an image key and a chosen Card. A confirmation stores no image, and Empty and Unidentified have no chosen Card. There is no minimum score and no automatic filter; the Admin's eye is the gate.

**Review state**
- `scan_feedback` gains three nullable columns: a review status (`approved` or `rejected`), the reviewer's profile and a review time. NULL means pending. No other table is added, and nothing already recorded is changed.
- The queue is the rows of an eligible kind with an image key and no status, oldest first. Rows recorded before this ships are included. Approve and reject move a row only out of pending; a row already reviewed answers conflict, so two Admins cannot both act on one photo.
- A row whose chosen Card has since been deleted cannot be approved and can only be rejected.

**Embedding**
- A user-supplied embedding is a row in the existing embeddings table with the configured model, a vector from the approved image, and `source` set to `correction:` followed by the `scan_feedback` row's id. One source per Correction keeps the unique key on Card, model and source as it is, so no migration loosens it.
- There is no cap on user-supplied embeddings per Card and no Remove action in the Admin app. The operator rolls back with a SQL delete on that source prefix.
- Queries about the catalog itself must read catalog rows only: the Reprint art distance (ADR-0023), the tuning script's reports and the pending-embedding query. Any other query joins embeddings by model alone today and is changed to filter the source. The nearest-neighbour search stays on all sources, because it is the point of the feature.
- Deleting a user's data does not cascade to their embeddings; this known gap is recorded in ADR-0027, and account deletion must close it.

**Matcher**
- The nearest-neighbour search, the best row per Card, the threshold, the margin and the candidate count are unchanged. User-supplied embeddings take effect as soon as they are approved, with no setting.
- The confident rule gains one condition: a result is never confident when its top Card has a Reprint, whatever the margin says. The Reprint lookup is made only when the result would otherwise be confident. A Card with no Reprint behaves as before.

**Photo access and approval (backend)**
- New admin routes, registered like the other admin routes and under the admin-only gate of ADR-0024, so a user-origin token gets the same not-found as a non-Admin. They list the queue with a page of items, mint a short-lived presigned GET for an item's photo, approve an item and reject an item.
- An item carries what the Admin needs to decide: the Correction's kind, source (`card_scan` or `binder_page`), the chosen Card with its catalog image, the Detection's scores, and for a binder the Page row and column, the grid and the four corners.
- The Admin app uploads its cut with a presigned PUT under a server-chosen key `reviewed/<scan_feedback id>.jpg` in the private bucket, with a signed content type and a short expiry. A card-scan photo needs no cut and is uploaded as it is stored.
- Approve names that key. The API checks that it sits under the reviewed prefix of that item, that the first bytes carry a JPEG signature and that the object is under the size cap, and refuses it otherwise. It then embeds the object synchronously with the embedder the matcher uses for query photos, inserts the embedding and sets the status and reviewer in one transaction. If the embedding fails, nothing is stored and the item stays pending.
- Reject sets the status and reviewer and nothing else. Photos are never deleted by the app: a binder Page photo is shared by every cell cut from it, and ADR-0022 keeps photos until the operator deletes them by hand.

**Admin app**
- A new review screen in the Admin app, behind its existing gate. It shows the queue, one item at a time or as a list with the selected item large, and for each item the user's photo, the chosen Card's catalog image, the kind, the scores and Approve and Reject.
- The Admin app holds its own copy of the user app's cut logic (ADR-0024 allows no shared package), including the same cell margin, and uses it both to preview a binder cell and to produce the JPEG uploaded on approve. A card-scan photo is shown and uploaded as it is.
- The generated API client for the Admin app gains the new operations by its own generation target; the user app's target leaves them out, so the user bundle does not hold them.

**User app**
- A one-line notice on the scan screens, the card scan and the Page Review, says that photos of corrected scans may be used to improve matching. There is no setting and no opt-out. It is shown wherever a scan can end in a Correction, and nowhere else.

**Tuning**
- The tuning script is not changed. It already reports on catalog rows only once the catalog-only filter is in place. The gain from approved photos is not measured before the feature goes on; that was decided in ADR-0027.

## Testing Decisions

A good test exercises external behaviour only: a request with a token and the response and rows it produces at the backend, a rendered screen and a user interaction at the frontend. It does not assert which helper approves, how the source string is built beyond its observable effect, or the order of internal calls.

Seams, preferring existing ones:
- **Backend: the Huma HTTP boundary against real Postgres** (ADR-0005), through the existing route test environment, with the embedder faked and the object store faked. It covers, from the outside: an Admin on an admin-origin token lists only pending eligible Corrections (including old ones, excluding confirmations, Empty and Unidentified); a user-origin token and a non-Admin get the not-found; approving inserts one embedding with the `correction:` source for the chosen Card and sets the status; approving the same item twice answers conflict and leaves one row; a failed embedding stores nothing and leaves the item pending; a key outside the item's prefix, a non-JPEG and an oversized object are refused; rejecting sets the status and embeds nothing; an approved photo makes the matcher rank that Card higher for the same photo; the confident rule is never met when the top Card has a Reprint, and still is for a Card without one; the Reprint art distance and the pending-embedding query ignore user rows; and a row whose chosen Card was deleted cannot be approved.
- **Frontend, Admin app: rendered screens with the generated client mocked**, the existing Vitest and Testing Library pattern. They cover the queue, a binder cell shown as a cut, a card-scan photo shown as stored, Approve calling the upload then approve in order, Reject, an item taken by another Admin (conflict) and a failed approval staying in the queue. The copied cut logic is tested with the same fixtures as the user app's own cut tests.
- **Frontend, user app: the existing suite.** The notice appears on the card scan and on the Page Review, and on no other screen.
- No new Playwright spec. No new seam.

Prior art: the route tests of the scan feedback and the admin routes, the match route tests with a faked embedder, the Reprint route tests, the migration tests, the Admin gate tests from the extraction, and the cut tests of the user app.

## Out of Scope

- Measuring the gain before and after, a catalog-only mode in the tuning script and a setting to turn the feature off. The matcher uses approved photos from the start.
- A cap on user-supplied embeddings per Card, an eviction rule, and a Remove action in the Admin app. Rollback is SQL.
- Re-cropping or adjusting a binder cut in the review.
- An automatic quality gate such as a minimum distance from the catalog image.
- A command that re-embeds user-supplied photos after `EMBEDDING_MODEL` changes. The cut of every approval is kept so one can be written; until then the rows of the old model are simply unused.
- Deleting a user's embeddings, photos and review state when their data is deleted, and account deletion itself. A known gap in ADR-0027.
- Any opt-out for users, and expiry or automatic deletion of photos.
- Making a user photo a catalog image.
- Detecting the promo stamp and reading the set code from the photo's corner, which have their own research note and spec.

## Further Notes

- The Reprint rule restores what the margin does today for ties. It is needed because the margin was the only thing keeping a Reprint family non-confident, and a user photo can now break a near-tie by more than the margin.
- `placed` cannot be produced by today's review. It becomes reachable when a manual search lands in the review, and this spec needs no change then.
- The scan enhancements spec is implemented on its branch and was not yet merged when this was written; this spec reads the data it records and adds only the three review columns.
- An approved photo of a Card the user chose from "Other prints" teaches that print only. Its Reprints do not gain it, so a photo of a Reprint must be reviewed on its own.
- The cut kept with an approval is the exact image that was embedded, so a future re-embed with a new model compares like with like.
