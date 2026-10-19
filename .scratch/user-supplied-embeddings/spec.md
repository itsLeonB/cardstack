# User-supplied embeddings

Status: needs-triage

Draft: not grilled. Every point below comes from one conversation and is an idea or a known fact, not a settled decision. Run a grilling session before turning this into a spec or tickets. It depends on the scan enhancements spec (`.scratch/scan-enhancements/spec.md`) and on the admin app extraction spec (`.scratch/admin-app-extraction/spec.md`).

## Idea

When the matcher was not confident and the user switched to another candidate and confirmed, that user's photo is a real photograph of that Card. Embed it as an extra image of the chosen Card, so the next photo of the same print scores higher. A Card can already hold several embeddings, so the matcher can use it. A user photo may only enter the similarity search after an Admin has reviewed it; nothing the user uploads is trusted into the catalog by itself.

## Known facts

- A Card Embedding is stored with the model that made it and a source naming the image that was embedded, and a Card can hold several (GLOSSARY.md, ADR-0018). The table is unique on `(card_id, model, source)` today, with `source` currently always `catalog`, so several rows per source need that key loosened. The matcher already keeps the best row per Card.
- The scan enhancements spec stores each Correction with its photo, the chosen Card, its rank and the Detection. For a binder it keeps the whole Page photo with its grid and corners, so any cell crop can be rebuilt. This spec reads that data and adds nothing to what is recorded.
- Photos live in a private bucket, which the Admin would need read access to.
- The review needs a place to work, which is the separate Admin app.
- Reprints share a name and an artwork, so a user photo of one print is a photo of that print only; which Reprint the user chose matters.

## Questions the grilling must settle

- Which Corrections are eligible: only a user switching to another candidate, or also a Reprint, a placed Card, a confirmation of an unsure result? Is there a minimum, such as a distance from the catalog image, to keep a bad or blurry photo out?
- What does the Admin see and do in a review: approve, reject, crop again, and for a binder cell, adjust the cut? Is a rejection remembered so the same photo is not offered twice?
- What is a user-supplied embedding's source value, and how does it point back to its photo and its Correction? How is an approved one removed again?
- How many user-supplied embeddings may one Card hold, and what happens past the cap?
- Does an approved photo change the matcher's confident rule or its thresholds, since scores for that Card may rise?
- How is an approved embedding made for a binder cell: cut again from the stored Page photo, and with which margin?
- Does the same photo ever become a catalog image, or only an embedding?
- Is there a measurable gain to check before and after, using the tuning script and labelled photos, and what result would justify keeping the feature?
- Privacy: a user's photo is kept for matching other people's cards. What the user sees or is told, and what happens to its embedding if the user's data is deleted.

## Out of scope for now

- Detecting the promo stamp and reading the set code from the photo's corner. They are a separate research note and spec, decided with the data the scan enhancements spec collects.
