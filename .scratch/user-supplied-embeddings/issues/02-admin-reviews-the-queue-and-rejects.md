# 02: Admin reviews the queue and rejects

**What to build:** An Admin opens a review screen in the Admin app and sees the Corrections whose photos wait for review: the user's photo, the chosen Card's catalog image, the kind and the scores the matcher gave. A binder cell is shown as the cut the matcher would embed, cut in the Admin app with its own copy of the user app's cut logic and the same cell margin. The Admin rejects a photo and it leaves the queue for good. Photos are read through short-lived presigned links; the user app and a non-Admin cannot reach any of it.

Spec: `.scratch/user-supplied-embeddings/spec.md` (Review state, Photo access, Admin app). ADR-0027, ADR-0022, ADR-0024. Vocabulary: Correction, Admin and Admin app in `GLOSSARY.md`.

**Blocked by:** None (can start immediately). The admin app extraction (`.scratch/admin-app-extraction/spec.md`) and the scan enhancements (`.scratch/scan-enhancements/spec.md`) must be merged first.

**Status:** ready-for-agent

- [ ] `scan_feedback` gains a nullable review status (`approved` or `rejected`), reviewer and review time; NULL means pending, and nothing already recorded changes
- [ ] The queue lists rows of kind `switched`, `reprint` or `placed` that have an image key and a chosen Card and no status, oldest first, paged; rows recorded before this ships are included
- [ ] Confirmations, Empty and Unidentified rows are never listed
- [ ] An item carries the kind, source (`card_scan` or `binder_page`), the chosen Card with its catalog image, the Detection's scores and, for a binder, the Page row and column, grid and four corners
- [ ] A presigned GET for an item's photo is short-lived and minted only for a listed item
- [ ] Reject sets the status and reviewer and embeds nothing; a row already reviewed answers conflict
- [ ] The routes accept only an Admin on an admin-origin token; a user-origin token and a non-Admin get the same not-found as on the other admin routes
- [ ] The Admin app review screen shows the queue, the photo, the Card's catalog image, the kind, the scores and Reject; a card-scan photo is shown as stored
- [ ] A binder cell is shown as a cut made with the Admin app's own copy of the cut logic and the same cell margin as the user app; its tests reuse the user app's cut fixtures
- [ ] A rejected photo does not come back, and no object is deleted from the bucket
- [ ] The user app's generated client does not hold the new operations
