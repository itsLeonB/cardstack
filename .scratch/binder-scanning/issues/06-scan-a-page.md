# 06: Scan a Page

**What to build:** From the binder view, the user swipes to a Page and presses scan. They take a photo with the camera, or choose a photo file when the camera is unavailable, and drag four corners onto the outer corners of the pocket grid (the corners start as a centred rectangle). The browser flattens the photo, cuts it into equal cells (the Layout's rows × columns), crops each cell inward by a fixed margin (first value 3 percent per side, one tunable constant), downsizes each to about 1024 pixels on the long side, and sends each as a JPEG to the existing match endpoint. Several matches run at a time and the rest queue. A review grid with the Layout's rows and columns shows each cell as it is matched: a confident Card, an unsure Card with its candidates in a dropdown defaulting to the top one, loading, failed (with Retry), or no good candidate. Every cell defaults to occupied, the user can mark any cell as an empty pocket or Unidentified, and cells that differ from the Page's current Slots are highlighted. A failed cell or a cell with no good candidate defaults to Unidentified. Confirm waits until no cell is loading, then replaces the Page through ticket 05's operation and returns to the binder view on that Page. The review is held in memory only, and leaving after edits asks for confirmation.

Spec: `.scratch/binder-scanning/spec.md` (Page scan). Reuses the Phase 1 frame-source seam and the match endpoint; no new matcher or batch route. This is the largest ticket; if it does not fit one context, split it at the review grid (cutting and matching first, highlighting and confirm second).

**Blocked by:** 02 (binder view), 05 (replace Page operation)

**Status:** resolved

- [x] Scan starts from the viewed Page, and the photo comes from the camera or the file picker through the shared frame source
- [x] Four draggable corners start as a centred rectangle, and the flattened image is cut into the Layout's rows × columns cells with the margin as one named constant
- [x] Each cell is uploaded as a downsized JPEG to the match endpoint, with at most three matches in flight and the rest queued
- [x] The review grid has the Layout's rows and columns and shows confident, unsure (dropdown defaulting to top), loading, failed (with Retry) and no-good-candidate cells
- [x] Failed and no-good-candidate cells default to Unidentified, any cell can be set to empty or Unidentified by hand, and cells differing from the Page's current Slots are highlighted
- [x] Confirm is disabled while any cell is loading, sends the whole Page to the Replace Page operation, shows any Slot stored as Unidentified because the server did not know the Card, and then shows the updated Page
- [x] The review is not stored; refresh discards it, and leaving after edits asks for confirmation
- [x] Frontend feature tests with the generated client mocked, the frame source and the flatten-and-cut step faked, cover capture to review, queueing, the dropdown, failed and Retry, empty and Unidentified marking, highlighting and confirm
- [x] One Playwright spec on the file-picker path with stubbed match responses covers creating a binder, scanning a Page, reviewing and confirming, with cutting running for real in Chromium

## Answer

Implemented on `feat/binder-scanning` (frontend). Scan starts from the viewed Page; the photo comes through the shared frame source; four draggable (and arrow-key nudgeable) corners start centred and are validated; `lib/page-cut.ts` flattens and cuts the photo into the Layout's rows × columns (margin `CELL_MARGIN = 0.03` per side, about 1024 px JPEG cells, pure helpers `cellRegions`, `gridToPhoto` and `cellSize` for ticket 09's offline script; `cutPage` needs `createImageBitmap` and a canvas); cells go through the Phase 1 `MatchQueue` (3 in flight). The review grid has the Layout's rows and columns; failed and no-candidate cells default to Unidentified; changed cells are highlighted; Confirm waits for every cell and sends the whole Page to Replace Page, then lists any `unknown_card` Slots. Leaving after edits asks for confirmation (in-page buttons, router blocker, `beforeunload`). **Unverified:** `e2e/page-scan.spec.ts` could not run in the sandbox (Clerk sign-in times out there, as for `card-scanning.spec.ts`); it needs a run on a machine or CI with Clerk working.
