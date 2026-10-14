# Catalog admin

Status: ready-for-agent

## Problem Statement

The catalog is filled only by ingesting `asia.pokemon-card.com`, and that site does not carry the Indonesian promo set `SM-P` ("Kartu Promo Matahari & Bulan", about 186 cards). No other source has it in a usable form, and card scanning needs an image of the Indonesian print, which only someone holding the card can supply. So a collector who owns `SM-P` cards cannot record them in a Collection or scan them, and the owner of Cardstack has no way to fix a wrong name or add a missing card without editing the database by hand. Any re-run of the ingester would also silently undo such a hand edit.

## Solution

The owner opens a page nobody else is shown, signs in with an account they have named in configuration, and enters Expansion Sets and Cards by hand, one card at a time, each with a photographed image. The card's details and its image are saved separately: the image goes straight to the image bucket, and the card appears in the catalog once its image is confirmed. Everything entered this way is Curated: the Asia ingester never overwrites it. The first content entered is `SM-P`. Normal users see nothing different except that the new set and cards appear in browse, search and Collections like any other, and become scannable once the owner has run the embedding batch.

## User Stories

1. As the owner, I want to name the people allowed to edit the catalog in configuration, so that no role system or admin account type has to be built.
2. As the owner, I want admin routes to exist only when both the allowlist and the image-bucket settings are configured, so that a deployment without them exposes no admin surface at all.
3. As the owner, I want a signed-in person who is not on the allowlist to receive a not-found response from every admin route, so that the admin surface does not reveal itself.
4. As the owner, I want a guest with no token to receive the same not-found response, so that probing the admin routes learns nothing.
5. As the owner, I want a malformed allowlist entry to stop the API booting with a clear error, so that a typo cannot silently lock me out or let the wrong person in.
6. As the owner, I want the allowlist compared case-insensitively against the verified email in the Clerk token, so that capitalisation differences do not matter.
7. As an Admin, I want an `/admin` page reachable only by typing its address, so that it never appears in navigation, the sitemap or search results.
8. As a normal user, I want the app to load exactly as before, so that the admin code adds nothing to the bundle I download.
9. As a signed-in non-Admin who types `/admin`, I want to see the app's ordinary not-found page, so that the page's existence is not revealed.
10. As an Admin, I want to create an Expansion Set with a name, a code, a release date and a Series chosen from existing Series, so that `SM-P` can exist in the catalog under "Matahari & Bulan".
11. As an Admin, I want to pick an existing Expansion Set, have its name, release date and Series loaded, and save it with the changes, so that mistakes can be corrected without retyping what is right (its code stays as it was created).
12. As an Admin, I want the Series list ordered by derived release date to keep working when I add a set with an older date, so that the catalog's Series order does not change unexpectedly.
13. As an Admin, I want to create a Card in a chosen Expansion Set by its collector number, so that the card has a stable identity within the set.
14. As an Admin, I want the combination of set and collector number to be unique, and creating a card at one that already has a card refused with a conflict rather than treated as an update, so that I cannot enter the same card twice or overwrite one by accident.
15. As an Admin, I want entering a set and number that already has a card to load that card's details into the form, so that I change what I want and save the whole card, and nothing I leave alone is lost.
16. As an Admin, I want to give a Card a name, a Category, an optional Tag and a rarity chosen from existing rarities, so that it matches how ingested cards look.
17. As an Admin, I want choosing Item, Supporter or Stadium to set the Tag and Pokémon or Energy to leave it empty, so that the category code from the source list maps without extra typing.
18. As an Admin, I want the rarity of a new card to default to "Tanpa Tanda", so that every `SM-P` card gets the same promo rarity without choosing it each time (the default applies to the form and to creating, never to replacing a card, which always carries its rarity).
19. As an Admin, I want Illustrator, HP and other attributes left empty unless I fill them, so that nothing is guessed.
20. As an Admin, I want to pick an image file for a Card and see it before uploading, so that I can check I photographed the right card.
21. As an Admin, I want the page to reject a file that is not a JPEG, PNG or WebP, that is too large or that has the wrong dimensions before anything is sent, so that mistakes are caught immediately.
22. As an Admin, I want a Card to be not visible in the catalog until its image is confirmed, so that no card shows that scanning could never match.
23. As an Admin, I want the image upload to go directly to the image bucket through a short-lived signed address, so that image bytes never pass through the API.
24. As an Admin, I want the signed address tied to a key the server chooses and to a fixed content type, so that I cannot overwrite an unrelated object by mistake.
25. As an Admin, I want a confirm step after the upload that reads the object's first bytes and checks for a JPEG, PNG or WebP signature, so that a mislabelled file is caught.
26. As an Admin, I want an object that fails that check deleted and the save refused, so that a bad file never becomes a Card's image.
27. As an Admin, I want a Card's hosted image key set only after the check passes, so that a failed upload never leaves a Card pointing at nothing.
28. As an Admin, I want a "save and next" action that keeps the chosen set and moves to the next collector number, so that entering 186 cards is fast.
29. As an Admin, I want to delete a Card I entered by mistake, after a confirmation that names its set, collector number and name and before which nothing is deleted, so that the catalog is not left with junk and I never delete the wrong card.
30. As an Admin, I want deleting a Card to also remove its hosted image and its embeddings, so that no orphaned data remains.
31. As an Admin, I want to be able to write to any Expansion Set or Card regardless of how it first entered the catalog, so that I can fix a wrong name on an Asia-sourced card.
32. As an Admin, I want every row I write marked Curated, so that the ingester can tell my work from its own.
33. As the owner, I want the Asia ingester to skip any Curated Card or Expansion Set, so that my corrections survive the next scrape.
34. As the owner, I want the ingester to keep filling in the numbers in a set that are not Curated, so that curating one card does not freeze the whole set.
35. As the owner, I want a Curated set to stay mine even if the Asia site later publishes the same code, so that a hand-written set is never silently replaced.
36. As the owner, I want a way to hand a Curated row back to the ingester by editing the database, so that the lock is reversible even without a screen for it.
37. As a collector, I want `SM-P` cards to appear in browse, search and Collections like any other, so that I can record the cards I own.
38. As a collector, I want a Curated Card with no embedding yet to be absent from scan results rather than breaking them, so that scanning keeps working while the owner has not run the embedding batch.
39. As the owner, I want to run `make embed-catalog` for one set after an entry session, so that the new cards become scannable without a code change.
40. As the owner, I want the fact that an image's size cannot be capped by R2 documented, so that I know the frontend checks are a convenience and not a security boundary.
41. As the owner, I want admin traffic to pass through the same edge and rate-limit rules as the rest of the API, so that the admin surface adds no unprotected path.
42. As the owner, I want the glossary and ADR-0019 to describe the feature in the domain's own words, so that a later reader finds the reason.

## Implementation Decisions

- **Domain vocabulary:** two new glossary terms, Curated and Admin, are already written. A Curated Card or Expansion Set is one last written by an Admin; ingestion never overwrites it. An Admin is an ordinary Auth Identity the operator has listed, not a stored role.
- **Schema:** `cards` and `expansion_sets` each gain a non-null `curated` boolean, default false. The change is additive, so the running API keeps working while it is applied; rollout order is migration, then API, then frontend, matching the other additive catalog migrations. No other table changes.
- **Authorization:** one allowlist setting holds comma-separated emails. The API refuses to boot on an empty entry or a value that is not an email, naming the entry, as the client-origin setting already does. The check compares the token's `email` claim, which is trustworthy because Clerk has required a verified email since sign-up was set up (ADR-0015). A listed caller passes; any other caller, guest or signed-in, gets a 404 so the surface does not reveal itself. There is no admin role in the database and no new identity system.
- **Route registration:** the admin route group is registered only when the allowlist and the image-bucket settings are both present. When either is missing the routes do not exist, so a preview or CI deployment without them has no admin surface. Admin traffic uses the same edge secret and rate limits as the rest of the API.
- **API contract (admin):** REST operations, none of them an upsert. `POST /admin/expansion-sets` creates a set and `PUT /admin/expansion-sets/{id}` replaces its name, release date and Series (the code is immutable and not part of the PUT body; a Series of another Game is refused). `POST /admin/cards` creates a Card and `PUT /admin/cards/{id}` replaces its name, category, tags, rarity and illustrator (its set and collector number are its identity and are not editable). A create at a code, or at a set and collector number, that already exists is a 409 and changes nothing. A PUT replaces the resource with everything in the request and keeps nothing the request leaves out, so the rarity is required on a PUT and an omitted illustrator or tags list is cleared; "Tanpa Tanda" is the default rarity only on create. Also: `GET /admin/cards` looks a Card up by set and collector number (so an image can be uploaded for a Card that has none yet, and so the form can load a Card to edit), `DELETE /admin/cards/{id}` deletes a Card, and there are operations to request an image upload and confirm an uploaded image. Writes mark the row Curated. Unknown Series, rarity or set references are rejected with a validation error. A Card's category and tags use the existing closed vocabularies, and the admin Card lists all its tags (a `tags` array, accepted back unchanged on a PUT), which only a Trainer Card may carry. Requests for the upload and confirm steps carry the card identity; the server derives the object key from the Card's id (`cards/<card id>`), never from client input.
- **Image upload:** presigned PUT. The API signs a short-lived address for the server-chosen key with a signed content type, using R2 write credentials scoped to the one bucket. These credentials are new on the Railway API, which today holds none; they are stored as a secret and the admin routes are disabled without them. R2 CORS on the bucket must allow the frontend origin to PUT.
- **Image confirmation:** after the PUT the client calls confirm. The API reads the object's first bytes and checks for a JPEG, PNG or WebP signature. On a mismatch it deletes the object and refuses; on a match it sets the Card's hosted image key. The size and dimension limits are enforced only in the admin frontend, because R2 cannot cap a presigned upload. This is documented as a trust-one-editor measure, not a security boundary.
- **Delete:** deleting a Card removes its row, its hosted image object and its embeddings. A Card in an outstanding embedding batch is handled the way the embedding commands already handle a missing Card. The delete is by id; the form looks the Card up by set and collector number first, and its confirmation names the set, number and name. Every delete in the admin UI needs such a confirmation, and nothing is deleted before it.
- **Ingester change:** the Asia ingester skips any Card or Expansion Set already marked Curated, including set-level metadata, while still creating and updating non-Curated cards in the same set. It never clears the flag. This reverses the earlier lean that the Asia site should win if it later publishes `SM-P`.
- **Frontend:** one route-split `/admin` route, linked from nowhere (no nav entry, no sitemap, no search-engine indexing), so it is not in any ordinary user's bundle. A non-Admin who opens it sees the ordinary not-found page. The page has a set form and a card form in two sections. Entering a set and collector number looks the Card up; one that exists is loaded into every field of the details section (set, collector number, name, Category, Tag, rarity, illustrator), the admin changes what they want, and saving PUTs all the values; one that does not is created with POST. The details section saves the Card alone and never changes its image; "Save and next" there keeps the chosen set and advances the collector number. The set form likewise loads an existing set picked from its "Existing set" list and PUTs it by id with all values, and creates a new one with POST. The image section has a file picker and preview, runs the file checks before requesting a signed address, looks up the Card at the set and number in the form, uploads with a PUT, then confirms. The form says that a Card is hidden until its image is uploaded. The upload helper is kept separate from the page so later features can reuse it.
- **Content rules for `SM-P`:** the set is named "Kartu Promo Matahari & Bulan" with code `SM-P` under the existing "Matahari & Bulan" Series, with release date 2019-05-08 and no cover. The card list (number, English name, category code) comes from Bulbapedia's ITCG page as facts only, with an attribution line, because the page is CC BY-NC-SA 2.5 and Cardstack must stay non-commercial to use it. Rarity is "Tanpa Tanda" for every card. Names stay English until corrected from the physical card. Pokémon cards carry no HP, types or attacks; trainer cards map `I`, `Su`, `St` to Item, Supporter, Stadium; Energy gets an empty Tag.
- **Embeddings:** not made on save. After an entry session the owner runs `make embed-catalog ARGS="-set SM-P"`; a Curated Card cannot be scanned until that batch has been collected.
- **Operations docs:** the new settings (allowlist, R2 write credentials on the API, bucket CORS) are added to the deployment docs, and the image-hosting doc stops saying the API never holds R2 credentials.

## Testing Decisions

- A good test drives the system from the outside and asserts observable behaviour (responses, rows, objects, rendered output), not which internal function was called. It must fail if the behaviour breaks and survive a refactor that keeps behaviour.
- **Seam 1 — API route tests on real Postgres**, in the existing route-test style with the existing object-store mock: allowlist behaviour (listed passes, unlisted and guest get 404, routes absent when config is unset, malformed allowlist stops boot), set and Card create (a duplicate is a 409) and replace-by-id (nothing kept from the stored row) with validation errors, the presign and confirm flow with a passing and a failing signature (object deleted, key unset), every admin write marking Curated, and delete removing image and embeddings.
- **Seam 2 — ingester test on real Postgres**, in the existing ingestion-test style: re-running the ingester over a Curated Card or Expansion Set leaves it untouched, while non-Curated cards in the same set are still created and updated.
- **Seam 3 — frontend route test**, in the existing route-test style with a mocked API: the page refuses a wrong type, an oversize file or wrong dimensions before sending anything, "save and next" keeps the set and advances the number, and a non-Admin sees the not-found page.
- Prior art: the existing `*_routes_test.go` files and test environment helper, the Asia `ingest_test.go`, and `-guest-lock.test.tsx`. A manual end-to-end check (a real upload to a development bucket) is part of acceptance, since presigned PUT and CORS cannot be exercised by mocks.

## Out of Scope

- Any role or permission model beyond the single allowlist; no Cardstack admin account type.
- Contributions from users other than named Admins, and any review or approval flow.
- Bulk upload or CSV import of cards or images.
- Cover images for Expansion Sets, Series logos, and editing Series.
- A UI to hand a Curated row back to ingestion, or to browse what is Curated.
- Embedding on save, or any change to the embedding commands.
- Indonesian card names, a name-language field or a translation table.
- Translating the source list's Promotion text or per-card prose into the catalog.
- Other missing sets; `SM-P` is the first content, not a commitment.
- A server-side cap on upload size beyond what the frontend checks; R2 cannot enforce one on a presigned PUT.
- Moving to a hosted or off-the-shelf CMS (ADR-0019 considers and rejects it).

## Further Notes

- The decisions here are recorded in `docs/adr/0019-catalog-content-is-curated-through-an-in-app-admin.md` and the glossary entries Curated and Admin; both are written but not yet committed.
- Source research found no automated source of Indonesian `SM-P` images or rarity: Bulbapedia and Pokepedia are bot-blocked, TCGdex has no Indonesian copy, and the Hong Kong and Taiwan `SM-P` sets are different cards. A Wayback copy of Bulbapedia's ITCG page is the only complete card list (186 rows, 001 to 186).
- The `SM-P` content run depends on the owner photographing the cards; until then the set can be created, and any Card entered without an image stays hidden from the catalog.
- Before going live: confirm the R2 CORS rule and the R2 write credential on Railway, and run `make job` for the migration against production first.
- Tickets were intentionally not written; split this spec into backend and frontend tickets when ready.
