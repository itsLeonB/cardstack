# 04: Add a Card with an image from admin

**What to build:** An Admin can add or correct a Card in any Expansion Set, with a required photographed image, from the `/admin` page. The image goes straight from the browser to R2 through a presigned PUT, and a confirm step checks the object's first bytes before the Card's image key is set. A Card with no confirmed image never appears in the catalog. Every write marks the Card Curated (ADR-0019, glossary entries Curated and Admin).

**Blocked by:** 01, 02.

**Status:** resolved

- [ ] An admin operation upserts a Card by (Expansion Set, collector number) with name, Category, optional Tag and a rarity chosen from existing rarities, rejecting unknown references with a validation error; re-entering an existing set and number updates that Card
- [ ] Choosing Item, Supporter or Stadium sets the Tag; Pokémon and Energy leave it empty; rarity defaults to "Tanpa Tanda"; Illustrator and other attributes stay empty unless filled
- [ ] An upload operation returns a short-lived signed PUT address for a key the server derives from the Card's id, with a signed content type; the client never supplies the key
- [ ] A confirm operation reads the object's first bytes and accepts only a JPEG, PNG or WebP signature; on a mismatch it deletes the object and refuses; on a match it sets the hosted image key and marks the Card Curated
- [ ] A Card without a confirmed image is never visible in browse, search or Collections, however the implementation avoids image-less rows (writing the row at confirm, or hiding empty keys)
- [ ] The card form uploads with a PUT and then confirms; the browser can reach R2 (bucket CORS allows the frontend origin to PUT)
- [ ] Route tests on real Postgres with the object-store mock cover upsert and idempotent re-entry, validation errors, a passing and a failing signature (object deleted, key unset), and the Curated flag
- [ ] The deployment docs describe the bucket CORS rule and the write credential scope
- [ ] Manual end-to-end acceptance: one real image uploaded to a development bucket, confirmed, and visible on the catalog card, since presigned PUT and CORS cannot be exercised by mocks

## Comments

Implemented on `feat/catalog-admin`. The last checklist item (manual end-to-end acceptance: a real image uploaded to a development bucket, confirmed, and visible on the catalog card, with the bucket CORS rule in place) is still open for the owner, since mocks cannot exercise a presigned PUT.
