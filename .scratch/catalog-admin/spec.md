# Catalog admin

Status: ready-for-agent

## Problem Statement

The catalog is filled only by ingesting `asia.pokemon-card.com`, and that site does not carry the Indonesian promo set `SM-P` ("Kartu Promo Matahari & Bulan", about 186 cards). No other source has it in a usable form, and card scanning needs an image of the Indonesian print, which only someone holding the card can supply. So a collector who owns `SM-P` cards cannot record them in a Collection or scan them, and the owner of Cardstack has no way to fix a wrong name or add a missing card without editing the database by hand. Any re-run of the ingester would also silently undo such a hand edit.

## Solution

The owner opens a page nobody else is shown, signs in with an account they have named in configuration, and enters Expansion Sets and Cards by hand, one card at a time, each with a photographed image. Saving a card uploads its image straight to the image bucket and puts the card in the catalog. Everything entered this way is Curated: the Asia ingester never overwrites it. The first content entered is `SM-P`. Normal users see nothing different except that the new set and cards appear in browse, search and Collections like any other, and become scannable once the owner has run the embedding batch.

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
11. As an Admin, I want to edit an existing Expansion Set's name and release date, so that mistakes can be corrected.
12. As an Admin, I want the Series list ordered by derived release date to keep working when I add a set with an older date, so that the catalog's Series order does not change unexpectedly.
13. As an Admin, I want to create a Card in a chosen Expansion Set by its collector number, so that the card has a stable identity within the set.
14. As an Admin, I want the combination of set and collector number to be unique, so that I cannot enter the same card twice.
15. As an Admin, I want entering an existing set and number to update that card instead of failing, so that corrections and re-entry are one action.
16. As an Admin, I want to give a Card a name, a Category, an optional Tag and a rarity chosen from existing rarities, so that it matches how ingested cards look.
17. As an Admin, I want choosing Item, Supporter or Stadium to set the Tag and Pokémon or Energy to leave it empty, so that the category code from the source list maps without extra typing.
18. As an Admin, I want the rarity to default to "Tanpa Tanda", so that every `SM-P` card gets the same promo rarity without choosing it each time.
19. As an Admin, I want Illustrator, HP and other attributes left empty unless I fill them, so that nothing is guessed.
20. As an Admin, I want to pick an image file for a Card and see it before saving, so that I can check I photographed the right card.
21. As an Admin, I want the page to reject a file that is not a JPEG, PNG or WebP, that is too large or that has the wrong dimensions before anything is sent, so that mistakes are caught immediately.
22. As an Admin, I want a Card to be impossible to save without an image, so that no card exists that scanning could never match.
23. As an Admin, I want saving to upload the image directly to the image bucket through a short-lived signed address, so that image bytes never pass through the API.
24. As an Admin, I want the signed address tied to a key the server chooses and to a fixed content type, so that I cannot overwrite an unrelated object by mistake.
25. As an Admin, I want a confirm step after the upload that reads the object's first bytes and checks for a JPEG, PNG or WebP signature, so that a mislabelled file is caught.
26. As an Admin, I want an object that fails that check deleted and the save refused, so that a bad file never becomes a Card's image.
27. As an Admin, I want a Card's hosted image key set only after the check passes, so that a failed upload never leaves a Card pointing at nothing.
28. As an Admin, I want a "save and next" action that keeps the chosen set and moves to the next collector number, so that entering 186 cards is fast.
29. As an Admin, I want to delete a Card I entered by mistake, so that the catalog is not left with junk.
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
- **API contract (admin):** operations to upsert an Expansion Set by code, upsert a Card by set and collector number, delete a Card, request an image upload, and confirm an uploaded image. Upserts mark the row Curated. Unknown Series, rarity or set references are rejected with a validation error. A Card's category and tag use the existing closed vocabularies. Requests for the upload and confirm steps carry the card identity; the server derives the object key from the Card's id (`cards/<card id>`), never from client input.
- **Image upload:** presigned PUT. The API signs a short-lived address for the server-chosen key with a signed content type, using R2 write credentials scoped to the one bucket. These credentials are new on the Railway API, which today holds none; they are stored as a secret and the admin routes are disabled without them. R2 CORS on the bucket must allow the frontend origin to PUT.
- **Image confirmation:** after the PUT the client calls confirm. The API reads the object's first bytes and checks for a JPEG, PNG or WebP signature. On a mismatch it deletes the object and refuses; on a match it sets the Card's hosted image key. The size and dimension limits are enforced only in the admin frontend, because R2 cannot cap a presigned upload. This is documented as a trust-one-editor measure, not a security boundary.
- **Delete:** deleting a Card removes its row, its hosted image object and its embeddings. A Card in an outstanding embedding batch is handled the way the embedding commands already handle a missing Card.
- **Ingester change:** the Asia ingester skips any Card or Expansion Set already marked Curated, including set-level metadata, while still creating and updating non-Curated cards in the same set. It never clears the flag. This reverses the earlier lean that the Asia site should win if it later publishes `SM-P`.
- **Frontend:** one route-split `/admin` route, linked from nowhere (no nav entry, no sitemap, no search-engine indexing), so it is not in any ordinary user's bundle. A non-Admin who opens it sees the ordinary not-found page. The page has a set form and a card form with a file picker and preview, runs the file checks before requesting a signed address, uploads with a PUT, then confirms. "Save and next" keeps the chosen set and advances the collector number. The upload helper is kept separate from the page so later features can reuse it.
- **Content rules for `SM-P`:** the set is named "Kartu Promo Matahari & Bulan" with code `SM-P` under the existing "Matahari & Bulan" Series, with release date 2019-05-08 and no cover. The card list (number, English name, category code) comes from Bulbapedia's ITCG page as facts only, with an attribution line, because the page is CC BY-NC-SA 2.5 and Cardstack must stay non-commercial to use it. Rarity is "Tanpa Tanda" for every card. Names stay English until corrected from the physical card. Pokémon cards carry no HP, types or attacks; trainer cards map `I`, `Su`, `St` to Item, Supporter, Stadium; Energy gets an empty Tag.
- **Embeddings:** not made on save. After an entry session the owner runs `make embed-catalog ARGS="-set SM-P"`; a Curated Card cannot be scanned until that batch has been collected.
- **Operations docs:** the new settings (allowlist, R2 write credentials on the API, bucket CORS) are added to the deployment docs, and the image-hosting doc stops saying the API never holds R2 credentials.

## Testing Decisions

- A good test drives the system from the outside and asserts observable behaviour (responses, rows, objects, rendered output), not which internal function was called. It must fail if the behaviour breaks and survive a refactor that keeps behaviour.
- **Seam 1 — API route tests on real Postgres**, in the existing route-test style with the existing object-store mock: allowlist behaviour (listed passes, unlisted and guest get 404, routes absent when config is unset, malformed allowlist stops boot), set and Card upsert including idempotent re-entry and validation errors, the presign and confirm flow with a passing and a failing signature (object deleted, key unset), every admin write marking Curated, and delete removing image and embeddings.
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
- The `SM-P` content run depends on the owner photographing the cards; until then the set can be created but has no Cards, because a Card requires an image.
- Before going live: confirm the R2 CORS rule and the R2 write credential on Railway, and run `make job` for the migration against production first.
- Tickets were intentionally not written; split this spec into backend and frontend tickets when ready.
