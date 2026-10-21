# 04: Record scan feedback

**What to build:** A signed-in caller can post a batch of up to 100 feedback rows, one per confirmed Scan or Page Review cell. A new table keeps each row: the user, the source (`card_scan` or `binder_page`), the Collection (no foreign key, so deleting a Collection keeps its rows), and for a binder the Page, row and column. Also kept: the kind (`confirmed`, `switched`, `reprint`, `empty`, `unidentified`, `placed`), the Detection as size-capped JSON exactly as the client sent it, the chosen Card (none for Empty and Unidentified), the chosen Card's rank and its source (`top` or `reprint`), an image key, and for a binder the grid and four corners. A timestamp is set by the server. Before any row is stored, every image key in the batch is checked: it must sit under the caller's own prefix, the first bytes must be a JPEG, and the object must be under the size cap. An object that fails is deleted and the batch refused. Several rows may share a key, because a Page photo is shared by every Correction on that Page.

Spec: `.scratch/scan-enhancements/spec.md` (Recording). Decision: `docs/adr/0022-scan-corrections-are-recorded-with-their-photo-in-a-private-bucket.md`. Vocabulary: Detection, Correction in `GLOSSARY.md`.

**Blocked by:** 03 (private photo storage and upload addresses)

**Status:** done — implemented on `feat/scan-enhancements` (merge `61b2ef5`), review fixes in `294165a`, not yet merged to `main`.

- [ ] A valid batch is stored and each row reads back with every field above
- [ ] A confirmation with an image key is refused; a Correction may carry a key or none (its photo may be gone)
- [ ] A kind that disagrees with its fields is refused, such as `empty` or `unidentified` with a chosen Card, or `switched`, `reprint` or `placed` without one
- [ ] A key outside the caller's prefix, a non-JPEG, and an oversized object are each refused, and the failing object is deleted
- [ ] A batch of more than 100 rows, and a Detection over the size cap, are refused
- [ ] Rows that share a key are accepted when the key passes once
- [ ] No row of a refused batch is stored
- [ ] A Guest or an invalid token is rejected, and one caller cannot read or affect another's rows or objects
- [ ] Deleting a Collection leaves its rows in place
- [ ] With the private bucket not configured the route refuses with the same machine-readable reason as ticket 03
- [ ] The migration adds indexes that serve the questions the operator will ask, by chosen Card and by kind over time
- [ ] The updated API contract is regenerated
- [ ] All of the above is covered by tests at the HTTP boundary against real Postgres, with the generated `ObjectStore` mock
