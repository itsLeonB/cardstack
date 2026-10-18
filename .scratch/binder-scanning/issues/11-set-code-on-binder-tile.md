# 11: Set code on the binder tile

**What to build:** A filled tile in the binder view's Page grid shows which print the Card is. Today it shows the Card's name and `#localId`; it now shows the name and, on the second line, `SET · #localId`, the same line the page-review pill shows. The tile's accessible name also carries the set name, so a screen reader user hears which print the Slot holds. Both the tile and the review pill take that line from one shared formatter, so they cannot drift apart.

Spec: `.scratch/binder-scanning/spec.md` (Binder view (frontend), stories 74 and 75). Frontend only: the Slot reads already return the expansion set on the card summary, so there is no backend or API change. The Slot panel and the entry list are unchanged.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] A filled tile shows the Card's name (truncated) and, on the second line, the set code and local ID as `SET · #localId`, and still fits the narrowest Layout (4×3 and 3×4) on a phone without overflowing
- [ ] A filled tile's accessible name carries the position, the Card's name and the set name
- [ ] Empty and Unidentified tiles are unchanged
- [ ] The tile and the page-review pill format the set code and local ID from one shared helper, and the review pill looks and reads as before
- [ ] The binder view tests (generated client mocked) cover the set code on a filled tile and the set name in its accessible name, and the page-scan tests still pass
- [ ] Build, lint, format and unit tests pass
