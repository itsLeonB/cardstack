# Card scanning

Status: ready-for-agent

## Problem Statement

A collector who owns a stack of physical cards has to find each one in the catalog by name or set and card number, then type its quantity into a Collection, one card at a time. With hundreds of cards that is slow and error-prone, and the user already has the cards in hand and could just point a phone camera at them. Today there is no way to add cards to a Collection from a photo, and no way to review a batch of additions before they touch the Collection.

## Solution

The user opens one of their Collections and presses "Scan cards". A camera viewfinder with a card-shaped guide appears. Each capture is sent to the backend, which matches the photo against the catalog. A confident match is added to a Draft Addition (a transient, unsaved list of Cards with quantities to add). A match that is not confident shows a few candidate cards to pick from, or the user skips it. The Draft Addition is held in the browser and survives a refresh. When the user is done they review the list, adjust quantities, and press "Add to Collection", which writes the quantities into that Collection's Inventory Entries through the existing bulk update. Cards the Collection cannot take (capacity) stay in the draft with the reason shown.

This spec covers Phase 1 (one card per photo). Phase 2 (photographing a whole binder page, recognising the layout, and storing card positions) is outlined under Out of Scope and gets its own spec after Phase 1 ships.

## User Stories

1. As a collector, I want a "Scan cards" button on a Collection's page, so that I can start adding cards to that Collection from my camera.
2. As a collector, I want scanning to always belong to the Collection I opened it from, so that I never have to say where the cards go when I finish.
3. As a collector, I want a live camera viewfinder with a card-shaped guide, so that I know how to frame the card.
4. As a collector, I want to tap a capture button to photograph one card, so that I control exactly which card is scanned.
5. As a collector, I want the app to crop to the guide and shrink the photo before sending it, so that scans are fast and use little mobile data.
6. As a collector, I want a confident match to be added to my draft automatically, so that scanning a stack of cards takes one tap per card.
7. As a collector, I want to see the card that was just matched (image, name, set, number) right after a capture, so that I can tell at a glance whether the app got it right.
8. As a collector, I want to undo or remove a card that was matched wrongly, so that a bad match does not end up in my Collection.
9. As a collector, I want to see a short list of candidate cards when the app is not sure, so that I can pick the right one with one tap.
10. As a collector, I want to be able to skip a capture that has no good candidate, so that I can retake the photo without polluting my draft.
11. As a collector, I want two lookalike reprints that the app cannot tell apart to be offered as candidates instead of silently choosing one, so that I never record the wrong edition.
12. As a collector, I want scanning the same card twice to raise its quantity in the draft to 2, so that duplicates in a stack are counted.
13. As a collector, I want to raise or lower the quantity of any row in my draft, so that I can correct counts without rescanning.
14. As a collector, I want to remove a row from my draft, so that I can drop a card I scanned by mistake.
15. As a collector, I want to see my draft as a tray of cards with counts while I scan, so that I know how many cards I have captured so far.
16. As a collector, I want a "Review and add" step that lists every card and quantity I am about to add, so that I can check everything before it changes my Collection.
17. As a collector, I want the quantities in my draft to be additions to what the Collection already holds, so that scanning three copies of a card I already own makes my total go up by three and not reset it to three.
18. As a collector, I want the review step to show how many copies of each card the Collection already holds, so that I can see the result of adding.
19. As a collector, I want a running total of the Collection's cards against its capacity limit on the review step, so that I get an early warning before I press Add.
20. As a collector, I want pressing "Add to Collection" to write all rows at once, so that I do not have to save them one by one.
21. As a collector, I want rows that were added to leave my draft after a successful add, so that I do not add them twice.
22. As a collector, I want rows the Collection could not take (over the capacity limit, or a card the server does not know) to stay in the draft with the reason, so that I can fix or drop them.
23. As a collector, I want to be taken back to the Collection's page when everything was added, so that I can see the result.
24. As a collector, I want my draft to survive a page refresh, an accidental tab close, or my phone switching apps, so that I do not lose a long scanning session.
25. As a collector, I want each Collection to have its own draft, so that scanning for one binder never mixes into another.
26. As a collector, I want a draft whose Collection no longer exists to be discarded quietly, so that I am not stuck with an orphan list.
27. As a collector, I want to discard my whole draft, so that I can start over.
28. As a collector, I want a file picker as a fallback when the camera is unavailable, denied, or I am on a desktop, so that I can still scan from a photo file.
29. As a collector, I want a clear message when camera permission is denied, so that I know why the viewfinder is missing and what to do.
30. As a collector, I want a clear message when the match service is unavailable or slow, so that I know to retry and my draft is unaffected.
31. As a signed-in user, I want scanning to require my login, so that nobody can use the match service anonymously.
32. As a Guest, I want not to see the scan feature, so that I am not shown something that needs an account.
33. As the site owner, I want the match endpoint rate-limited per caller and the upload size-capped, so that the match service cannot be abused.
34. As the site owner, I want to switch the scan button on and off with a build-time frontend setting, so that the feature stays hidden in production until the real matcher is ready.
35. As a frontend developer, I want the match endpoint to exist early with its final response shape, so that the scan screen can be built and tested before the real matcher exists.
36. As a developer, I want the early match endpoint to return random cards (sometimes confident, sometimes a candidate list), so that both UI paths can be exercised.
37. As a developer, I want replacing the stub with the real matcher to change no frontend code, so that the swap is low risk.
38. As the site owner, I want the matcher's confidence threshold decided by the server and tunable without a frontend release, so that I can adjust it after seeing real scores.
39. As the site owner, I want a measured accuracy check on real phone photos before the real matcher is built, so that I choose the embedding provider on evidence.
40. As the site owner, I want the catalog's card images embedded once in a batch job that can be re-run, so that a model change or new cards only need a re-run.
41. As the site owner, I want a card whose image failed to host to be skipped by the matcher without breaking it, so that one bad image does not block scanning.
42. As a collector, I want to find a Card by its id in the catalog API, so that the app can show my draft's cards after a refresh.

## Implementation Decisions

**Domain**
- Draft Addition is a new glossary term (already added to the glossary). It is transient and unsaved, targets exactly one Collection, and holds quantities to add, not new totals. It is not stored on the server.
- Inventory Entries, Collections and Master Inventory are unchanged. Nothing about binder pages or slots is added in Phase 1.

**Flow**
- Scanning starts from a Collection and has its own screen scoped to that Collection. After a successful add, the user returns to the Collection's page.
- The browser captures one still per tap, using the camera, or a file picker when the camera is unavailable. Both feed the same pipeline.
- The browser crops to the on-screen guide and downsizes to about 1024 pixels on the long side, then uploads a JPEG. Server-side card detection and cropping are not part of Phase 1.
- Capture adds 1 to the matched card's quantity in the draft, merging rows for the same card.

**Draft (frontend only)**
- A list of (card id, quantity to add) for one Collection, held in memory and mirrored to local browser storage, keyed per Collection. The draft stores only ids and quantities; card details are fetched from the backend when rendering.
- A draft whose Collection fails to load as missing is dropped.
- On review, the frontend reads the Collection's current quantities for the draft's cards using the existing Collection entries lookup (card id filter, at most 100 per request), computes an absolute target per card as current plus added, and sends the existing bulk update. The bulk update is already idempotent on retry and reports applied, removed or declined per item with a machine-readable reason.
- After the response: applied rows are removed from the draft, declined rows stay with their reason, and the whole draft clears only when nothing was declined. The review step shows the Collection's running total against its capacity limit as a warning only; the server remains the authority.

**Catalog API**
- The catalog card search gains a repeatable card id filter (at most 100), mirroring the same filter on the Collection entries search, so a draft's cards can be rendered from ids. Guests keep their existing limits.

**Match endpoint (new, authenticated)**
- A new authenticated POST takes one uploaded image and returns ranked candidates, each with a score and a card summary (id, name, expansion set, card number, hosted image address), plus a `confident` flag.
- The server owns the confidence threshold. `confident` is true only when the top score clears the threshold and the runner-up is clearly behind. A near-tie, such as the same artwork reprinted in two Expansion Sets, is non-confident and is never auto-picked.
- The uploaded image is processed in memory and not stored by Cardstack. The route has a body size cap and the existing rate limiting applies. The catalog holds only Indonesian prints, so only Indonesian cards are matchable and no foreign-edition handling is built or tested.
- Cards without a hosted image have no embedding and are not matchable.
- The endpoint's response schema is final from the first version, so the generated frontend client does not change when the matcher is replaced.

**Stub first, real matcher later**
- The first version of the endpoint is live and returns random Cards from the catalog: about half the calls confident, otherwise three to five candidates with plausible scores. It writes nothing. There is no backend switch; the stub is simply replaced by the real matcher when it is stable.
- The scan button is gated by a build-time frontend environment variable, off in production until the real matcher lands. Flipping it needs a frontend rebuild.

**Real matcher**
- Matching is nearest-neighbour search over image embeddings of the catalog's hosted card images, stored in Postgres with the pgvector extension in a new table keyed by Card. Each row records which embedding model produced it, and a query only compares against rows from the same model; changing the model means re-embedding every card.
- A re-runnable batch job embeds every Card with a hosted image and skips Cards already embedded by the current model.
- Embeddings come from a hosted image-embedding API with a free tier. The provider (Voyage `voyage-multimodal-3.5`, Gemini `gemini-embedding-2`, or a self-hosted DINOv2/SigLIP fallback) is chosen by the accuracy spike. If accuracy is equal, prefer Gemini. Vector dimensions must stay at or below the pgvector index limit of 2000. Details and sources are in `.scratch/card-scanning/research/hosted-image-embedding-apis.md`.
- Accuracy bar for the spike: top-1 of at least 90 percent and top-5 of at least 98 percent on about 30 real phone photos with glare and angle. If a hosted provider misses the bar, fall back to a self-hosted model service, or to the hybrid in which OCR of the printed card number narrows the candidates before embeddings rank them. The confidence threshold is tuned on the spike's score distribution so that a confident result is almost never wrong, accepting that non-confident results are common.
- The embedding provider call is the one new backend seam: it sits behind a small boundary so tests can fake it.
- An ADR (embedding provider, pgvector, hosted vs self-hosted) is written during the real-matcher ticket, after the spike, not before.

**Privacy**
- Users' photos go to a third-party provider. The decision is to accept the provider's free-tier terms, which may allow product-improvement use or human review, and to show no notice on the scan screen. Revisit if anyone other than the owner starts scanning, or if a provider's terms change.

**Delivery order**
1. Accuracy spike (gates 5).
2. Card id filter on the catalog search.
3. Stub match endpoint with the final contract.
4. Frontend scan screen, built against the stub.
5. Real matcher: embeddings table and batch job, embedding provider, confident flag, ADR; replaces the stub.

Tickets 2, 3 and 4 do not depend on the spike.

## Testing Decisions

A good test exercises external behavior only: a request in and a response out at the backend, a rendered route and user interaction at the frontend. It does not assert how the draft is stored internally, which helper computed a quantity, or how a vector index is queried.

Seams, preferring existing ones:
- **Backend: the Huma HTTP boundary against real Postgres** (ADR-0005; the backend feature tests described in the MVP spec). The match endpoint and the card id filter are tested here, end to end. The stub is tested for its contract and for being auth-only. The real matcher is tested with the embedding provider faked, covering the confident, near-tie and unembedded-card cases and the pgvector search against real Postgres.
- **Backend: the embedding provider boundary** is the single new seam on the backend: faked in tests. The real provider is only ever called by the spike and the batch job, never in CI.
- **Frontend: the scan route, rendered with the generated API client mocked** (the existing Vitest and Testing Library feature-test pattern). It covers capture to draft, merging duplicates, the candidate pick and skip, quantity edits, persistence across a re-render from local storage, review, absolute-target conversion, and applied versus declined handling. The new seam here is the frame source (camera or file picker) that yields a JPEG, faked in tests.
- **One Playwright end-to-end spec** on the file-picker path with a stubbed match response, signed in through the existing Clerk e2e setup, covering scan, review and add into a seeded Collection. It does not need a camera.
- **The accuracy spike is an offline script and report**, not a CI test. It is run on demand, outside CI.

Prior art: the bulk update's service and route tests, the catalog and Collection entries search tests, the frontend quantity batching tests, and the existing signed-in Playwright specs.

## Out of Scope

- Phase 2: photographing a binder or album page, recognising its layout (pages, rows and columns), pointing it at an existing Collection and updating that Collection with the layout, and storing each card's position. Positional slots are the intended direction, so a Collection will eventually carry a page layout and slot records pointing to Cards. How a slot relates to quantity (one slot per physical copy) is deliberately undecided until Phase 1 ships. Phase 2 also needs server-side detection of many cards in one frame.
- Live scanning (the browser auto-detecting and capturing cards from video). Phase 1 is tap to capture.
- A server-side Draft Addition table, multi-device draft sync, and drafts that expire.
- Server-side card detection and cropping.
- Foreign print editions: English, Japanese and other regions. They are not in the catalog, so the matcher assumes Indonesian cards only and nothing handles or tests a foreign card. Revisit when another region's Expansion Sets are added.
- Print finish or Card Variant detection (normal, reverse holo, holo), which is not modelled (ADR-0006).
- Condition grading.
- Any scan-screen privacy notice.
- A backend feature switch for the stub.
- Deduplicating a card held in view across several captures. Each tap is one capture.

## Further Notes

- Reprints share artwork across Expansion Sets, which is why an embedding match can tie. The hybrid with OCR of the printed number is the planned fix if candidates turn out to be needed too often; it is not part of Phase 1.
- Free-tier limits for Gemini are unpublished and Voyage's free-trial rate limits are unverified, so the spike should also record real throughput for the batch job.
- The research note lists a half-day spike design (30 real photos against about 200 catalog cards, Voyage, Gemini and a local control) that the spike ticket can reuse.
- `VITE_*` values are baked in at build time, so enabling the scan button in production is a frontend rebuild (see the deployment docs).
