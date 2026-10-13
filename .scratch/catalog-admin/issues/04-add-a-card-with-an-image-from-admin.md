# 04: Add a Card with an image from admin

**What to build:** An Admin can add or correct a Card in any Expansion Set, with a photographed image uploaded separately from its details, from the `/admin` page. The image goes straight from the browser to R2 through a presigned PUT, and a confirm step checks the object's first bytes before the Card's image key is set. A Card with no confirmed image never appears in the catalog. Every write marks the Card Curated (ADR-0019, glossary entries Curated and Admin).

**Blocked by:** 01, 02.

**Status:** resolved

- [ ] `POST /admin/cards` creates a Card at (Expansion Set, collector number) with name, Category, optional tags and a rarity chosen from existing rarities, rejecting unknown references with a validation error; a set and number that already has a Card is a 409, never an update
- [ ] `PUT /admin/cards/{id}` replaces the Card's name, Category, tags, rarity and illustrator with the request's and keeps nothing the request leaves out (the rarity is required); the admin Card response lists all its `tags`, and the same array is accepted back on the PUT
- [ ] Choosing Item, Supporter or Stadium sets the Tag; Pokémon and Energy leave it empty; rarity defaults to "Tanpa Tanda" in the form and on create only; Illustrator and other attributes stay empty unless filled
- [ ] The card form looks the Card up by set and collector number: one found is loaded into every field and saved with a PUT of all values, one not found is created with a POST
- [ ] An upload operation returns a short-lived signed PUT address for a key the server derives from the Card's id, with a signed content type; the client never supplies the key
- [ ] A confirm operation reads the object's first bytes and accepts only a JPEG, PNG or WebP signature; on a mismatch it deletes the object and refuses; on a match it sets the hosted image key and marks the Card Curated
- [ ] A Card without a confirmed image is never visible in browse, search or Collections, however the implementation avoids image-less rows (writing the row at confirm, or hiding empty keys)
- [ ] The card form's image section uploads with a PUT and then confirms, for the Card found by set and collector number; the browser can reach R2 (bucket CORS allows the frontend origin to PUT)
- [ ] Route tests on real Postgres with the object-store mock cover create, a duplicate (409), replace-by-id, validation errors, a passing and a failing signature (object deleted, key unset), and the Curated flag
- [ ] The deployment docs describe the bucket CORS rule and the write credential scope
- [ ] Manual end-to-end acceptance: one real image uploaded to a development bucket, confirmed, and visible on the catalog card, since presigned PUT and CORS cannot be exercised by mocks

## Comments

Implemented on `feat/catalog-admin`. The last checklist item (manual end-to-end acceptance: a real image uploaded to a development bucket, confirmed, and visible on the catalog card, with the bucket CORS rule in place) is still open for the owner, since mocks cannot exercise a presigned PUT.
