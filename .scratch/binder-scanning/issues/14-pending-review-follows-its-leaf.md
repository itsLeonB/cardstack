# 14: A pending review follows its Leaf

**What to build:** Adding or removing a Leaf renumbers the Pages after it, so a pending Page Review saved for "Page 7" must not end up pointing at a different physical sheet, where Confirm would overwrite the wrong Page. When the user adds a Leaf before a pending review's Page, the review's Page number shifts with its Leaf. When the user removes a Leaf before it, the number shifts the other way. A review is discarded, with a one-line message, when its own Leaf is removed, or when the Layout's rows or columns no longer match the review's cells.

Spec: `.scratch/binder-scanning/spec.md` (Pending Page Review (frontend), story 86). Frontend only. Removing a Leaf is already refused while either face holds a card, so only an empty Page's pending review can lose its Leaf; the Layout can only change while every Slot is empty.

**Blocked by:** 12

**Status:** resolved

- [x] Adding a Leaf before or after a Leaf renumbers the pending reviews of every later Page to match, so the banner and the notice show the new Page numbers and Resume opens the same review
- [x] Removing a Leaf renumbers the pending reviews of every later Page to match
- [x] Removing the Leaf that holds a pending review's Page discards that review and tells the user in one line
- [x] A pending review whose cell count does not match the Layout's rows and columns is discarded with a one-line message when the binder opens
- [x] A pending review of another binder or another user is never changed
- [x] The frontend tests (generated client mocked, IndexedDB faked) cover a shift on add, a shift on remove, a discard on remove of its own Leaf, and a discard on a Layout mismatch
- [x] Build, lint, format and unit tests pass

## Answer

Implemented on `feat/binder-scanning` (frontend): `lib/page-review-leaf.ts` plans and applies the shift (a Leaf is two Pages, so later Pages move by two; the highest first on add, the lowest first on remove). Removing a Leaf discards the reviews of its two Pages with a toast; opening the binder discards reviews whose cell count no longer matches rows × columns. The notice and banner refresh after each change. Known limit: a rename that fails partway is swallowed, so a review may sit on the wrong Page number but none is lost.
