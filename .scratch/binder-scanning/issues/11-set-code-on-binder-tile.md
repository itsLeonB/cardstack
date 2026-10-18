# 11: Set code on the binder tile

**What to build:** A filled tile in the binder view's Page grid shows which print the Card is. Today it shows the Card's name and `#localId`; it now shows the name and, on the second line, `SET · #localId`, the same line the page-review pill shows. The tile's accessible name also carries the set name, so a screen reader user hears which print the Slot holds. Both the tile and the review pill take that line from one shared formatter, so they cannot drift apart.

Spec: `.scratch/binder-scanning/spec.md` (Binder view (frontend), stories 74 and 75). Frontend only: the Slot reads already return the expansion set on the card summary, so there is no backend or API change. The Slot panel and the entry list are unchanged.

**Blocked by:** None (can start immediately)

**Status:** resolved

- [x] A filled tile shows the Card's name (truncated) and, on the second line, the set code and local ID as `SET · #localId`, and still fits the narrowest Layout (4×3 and 3×4) on a phone without overflowing
- [x] A filled tile's accessible name carries the position, the Card's name and the set name
- [x] Empty and Unidentified tiles are unchanged
- [x] The tile and the page-review pill format the set code and local ID from one shared helper, and the review pill looks and reads as before
- [x] The binder view tests (generated client mocked) cover the set code on a filled tile and the set name in its accessible name, and the page-scan tests still pass
- [x] Build, lint, format and unit tests pass

## Answer

Implemented on `feat/binder-scanning` (frontend): `cardCode(card)` in `lib/binder.ts` formats `SET · #localId` and both the binder tile (`binder-page-grid.tsx`) and the review pill use it. A filled tile's second line is truncated and its accessible name is `Row N, column M: <name>, <set name>`. Build, lint, format and unit tests pass. The 4×3 and 3×4 fit on a phone was checked by the truncate classes only, not in a browser.
