# 06: Draft Addition keeps resolved Scans with their photos

**What to build:** A user who refreshes during single card scan keeps their resolved Scans, and each Scan still has its photo and the Detection the matcher gave. Today only ids and a quantity survive in local storage and photos are in memory. The resolved Scans move to IndexedDB, keyed by user and Collection, reusing the pattern of the Page Review store, so a Correction made before a refresh can still upload its photo after it. Pending and failed Scans stay in memory as today. The saved data is removed when the Draft Addition is confirmed or cleared and when a Scan is removed. A second user on the same browser never sees the first user's saved Scans, and a browser without IndexedDB falls back to memory without breaking the flow.

Spec: `.scratch/scan-enhancements/spec.md` (Frontend, Persistence). Decision: `docs/adr/0022-scan-corrections-are-recorded-with-their-photo-in-a-private-bucket.md`. Vocabulary: Draft Addition, Scan, Detection in `GLOSSARY.md`.

**Blocked by:** None (can start immediately)

**Status:** done — implemented on `feat/scan-enhancements` (merge `9a2a294`), review fixes in `294165a`, not yet merged to `main`.

- [ ] After a refresh a resolved Scan shows its chosen Card, quantity and other candidates as before, and keeps its photo and Detection
- [ ] Pending and failed Scans are not persisted and behave as today
- [ ] Confirming, clearing the Draft Addition and removing a Scan each delete the saved data they cover
- [ ] Saved Scans are keyed by user and Collection; another user, and another Collection, see none of them
- [ ] Existing Draft Additions saved in local storage before this change still load, and are moved without losing a Scan
- [ ] With IndexedDB unavailable the Draft Addition works from memory and nothing throws
- [ ] The Detection is stored exactly as the match response was received, so fields the API adds later are kept
- [ ] All of the above is covered by tests on the scan route with the generated API client mocked and IndexedDB faked in memory
