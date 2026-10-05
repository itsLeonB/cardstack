# Series logos and Expansion Set covers are pre-sized at host time, not at read time

Partly supersedes ADR-0016 (the Expansion Set half; Cards are unchanged).

ADR-0016 serves every hosted image through Cloudflare Image Transformations at read time. That suits Cards, which are many and shown at several sizes up to a detail view where the original is wanted. It suits neither of these: a Series logo is one of four static files shown in one place, and an Expansion Set cover is only ever shown at 64px, with no view that shows it larger. For both we now resize once when hosting and serve the small file directly, so a read is a plain fetch with no transformation quota and no redirect-to-original fallback.

A Series logo comes from Bulbagarden Archives, the only source with Indonesian-region Series logos (the Sun & Moon one is cropped out of the First Impact set logo, and the Pokemon TCG lockup is cropped off the Scarlet & Violet one). A committed shell script turns the four originals into 120px-tall WebP files, committed under `backend/assets/series/`, and a Go command uploads them as `series/<code>.<hash8>.webp` and stores the key on the Series row. An Expansion Set cover is resized from the original already in R2, not re-downloaded from pokemonasia: a Go command shells out to `vips`, uploads `expansion-sets/<id>.<hash8>.webp` (128px on the longest side) and stores it in a new `cover_key`, leaving `image_key` pointing at the original as the source for any later re-size. Both keys carry a content hash because hosted objects are uploaded `immutable` for a year, so a changed file must get a new key.

## Considered Options

- Keep Expansion Sets on read-time transformation (ADR-0016 as written). Rejected: it costs only about 94 transformations a month, but it keeps set covers dependent on Image Transformations and on the redirect fallback for a file that never needs more than one size.
- Encode WebP in pure Go or with a cgo library. Rejected: Go has no pure-Go lossy WebP encoder to our knowledge, and cgo adds a build dependency to the whole backend. The commands that need it are run by hand, never on Railway, so requiring `vips` on the developer machine is acceptable.
- Fixed keys without a content hash. Rejected: re-cropping or re-sizing a file later would be served stale for up to a year.

## Consequences

Changing a display size later means re-running the resize, which writes new keys and leaves the old objects orphaned in R2. The Series logos are Pokémon Company artwork under a contributor's fair-use claim on Bulbagarden, not an open licence; we self-host them with no in-app credit, the same as the pokemonasia-sourced card and set art, and a project-wide credit would be its own decision.
