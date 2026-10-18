# 12: Save a Page Review and resume it

**What to build:** A page-scan review is no longer lost when the user leaves or refreshes. Once the photo is cut, the review is saved in the browser as a pending Page Review, and it is saved again after every match that arrives, every pick and every mark. The user can close the screen, refresh or close the tab, and come back: the Page shows a banner with Resume and Discard, and Resume restores the review as it was. Confirming (a successful Replace Page) or Discarding ends it. Because every change is saved, the "Leave this review?" dialog and the route blocker go away.

Spec: `.scratch/binder-scanning/spec.md` (Pending Page Review (frontend), stories 44, 76, 77, 78, 81, 82, 83, 84, 85, 87 and 88). Glossary term: Page Review. Frontend only, no backend change. This supersedes the old "memory only" rule and the leave dialog from ticket 06.

Decisions already made:
- Stored in the browser's IndexedDB because it holds photos. It holds the cut cell photos, each cell's match result (status, confidence and candidates) and the user's picks and marks. It does not hold the full photo or the corners, so a resumed review offers "Scan again" (back to the photo step) in place of "Adjust corners".
- At most one per signed-in user, binder and Page, keyed by the auth identity's user id, the Collection and the Page number. Another login on the same browser never sees it. Two tabs on the same one: last write wins.
- No expiry. It ends only on a successful Confirm or an explicit Discard.
- On resume, cells that were queued or loading go back in the match queue, finished matches are never re-run, and failed cells keep Retry. The difference ring is computed against the Page's current Slots on every render.
- When IndexedDB is unavailable (private mode, quota), the screen tells the user and falls back to a memory-only review with the leave dialog and blocker on, as before.
- Tests need an in-memory IndexedDB, which jsdom lacks. Adding a dev dependency for that is the implementer's call.

**Blocked by:** None (can start immediately; ticket 10 is resolved)

**Status:** resolved

- [x] After "Cut and match", the review is saved as a pending Page Review for the signed-in user, the binder and the Page, and is saved again after each match result, pick and mark
- [x] Leaving the screen with Back to binder, refreshing, or closing and reopening the tab loses nothing, and no "Leave this review?" dialog or route blocker appears
- [x] A Page with a pending review shows a banner with Resume and Discard; Resume reopens the review with its photos, matches, picks and marks, and Discard removes it after the user confirms
- [x] A successful Confirm removes the pending review; a failed Confirm keeps it
- [x] On resume, queued and loading cells match again, finished matches are not re-run, and a failed cell still shows Couldn't match with Retry
- [x] A resumed review offers "Scan again" in place of "Adjust corners", and "Scan again" starts from the photo step
- [x] Another signed-in user on the same browser does not see the banner or the review
- [x] When IndexedDB cannot be used, the user is told the review is not being saved and it behaves as before, with the leave dialog and blocker on after edits
- [x] The frontend tests (generated client mocked, frame source and cut faked, IndexedDB faked in memory) cover save and restore across unmount and remount, the banner, Resume, Discard, Confirm clearing it, re-queued and finished cells, Retry, user isolation, and the IndexedDB-unavailable fallback
- [x] Build, lint, format and unit tests pass, and the Playwright page-scan spec still passes or is reported as unrun if Clerk sign-in times out in the sandbox

## Answer

Implemented on `feat/binder-scanning` (frontend): `lib/page-review-store.ts` (IndexedDB: save, load, list, remove, rename) keyed by user, Collection and Page. The scan flow saves after the cut and after every match, pick and mark, and clears on a successful Confirm. `PendingReviewBanner` offers Resume and Discard on the Page; a resumed review re-queues unfinished cells and offers "Scan again". The leave dialog and route blocker remain only in the memory-only fallback, entered when a save fails, with a notice. `fake-indexeddb` was added as a dev dependency. `e2e/page-scan.spec.ts` was not run (Clerk sign-in times out in the sandbox).
