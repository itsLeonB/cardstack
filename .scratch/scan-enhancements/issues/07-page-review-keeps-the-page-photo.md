# 07: Page Review keeps the Page photo

**What to build:** A pending Page Review keeps the whole Page photo together with the grid and the four corners that were used to cut it, in the same IndexedDB record that already holds the cut cells and their results. A resumed review therefore still has the photo a Correction would upload, and any cell can be cut again from it. Confirming or discarding the review removes the photo with the rest of the record. The photo is the downscaled one the app already prepares for the cut, not the raw camera file, to keep a pending review small.

Spec: `.scratch/scan-enhancements/spec.md` (Frontend, Persistence). Decision: `docs/adr/0022-scan-corrections-are-recorded-with-their-photo-in-a-private-bucket.md`. Vocabulary: Page Review in `GLOSSARY.md`.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] A saved Page Review includes the Page photo, the grid and the four corners
- [ ] After leaving and resuming a review, the photo, grid and corners are still there
- [ ] Reviews saved before this change still resume, with no photo, without error
- [ ] Confirm and Discard remove the photo with the record
- [ ] "Scan again" on a Page replaces the saved photo with the new one
- [ ] A review moved or dropped because a Leaf was added or removed keeps its photo with it
- [ ] With IndexedDB unavailable the review works from memory and nothing throws
- [ ] The stored photo's size stays within what the current upload preparation already produces
- [ ] All of the above is covered by tests on the binder page-scan screen with IndexedDB faked in memory
