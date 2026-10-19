# 13: Record Page Review feedback

**What to build:** When a Page Review is confirmed and the Replace Page call succeeds, the client records what happened to every cell, silently, with the same kind rules as ticket 12. Each row also carries the Page, row and column and, for a Correction, the grid and four corners. The whole Page photo is uploaded once, whatever the number of Corrections on the Page, and every Correction row points at that one key. A review resumed after leaving still records correctly, because the photo, grid, corners and Detections were kept with it.

Spec: `.scratch/scan-enhancements/spec.md` (Recording, Frontend). Decision: `docs/adr/0022-scan-corrections-are-recorded-with-their-photo-in-a-private-bucket.md`. Vocabulary: Correction, Detection, Page Review in `GLOSSARY.md`.

**Blocked by:** 04 (record scan feedback), 07 (Page Review keeps the Page photo), 09 (empty default on binder cells), 11 (other prints in the Page Review menu), 12 (record card-scan feedback)

**Status:** ready-for-agent

- [ ] Confirming sends one row per cell with the right kind, using the rules and the shared recorder of ticket 12
- [ ] A cell the matcher called Empty and the user left Empty is a confirmation; changed to a Card or Unidentified it is a Correction
- [ ] Each row carries the Page, row and column, and a Correction row carries the grid and corners
- [ ] A Page with several Corrections uploads the Page photo once and every such row has the same key
- [ ] A Page with no Corrections uploads nothing
- [ ] A failed cell left Unidentified records nothing; a Card picked after a failed cell's Retry is judged against the new Detection
- [ ] A review resumed after leaving, or after a refresh, records correctly
- [ ] Recording runs after the Replace Page call succeeds and never delays or changes it; any failure is silent
- [ ] The saved review, with its photo, is removed on confirm whether or not recording succeeded
- [ ] All of the above is covered by tests on the binder page-scan screen with the generated API client mocked and IndexedDB faked in memory
