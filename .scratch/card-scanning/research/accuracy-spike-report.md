# Accuracy spike report: provider pick (ticket 01)

Date: 2026-10-10. Script and raw results: `.scratch/card-scanning/research/spike/` (`results/` is git-ignored; per-model notes in `results/<model>/NOTES.md`). Sources for every API fact: `.scratch/card-scanning/research/accuracy-spike-sources.md`.

## Decision

**Gemini `gemini-embedding-2` is the provider, on the free tier, at 1536 dimensions.** This is an operator decision made on operational grounds, not the result of the ticket's decision rule: the 40-photo accuracy run was skipped. The reasoning, stated by the developer:

- Operational limits are fixed by the provider choice, while accuracy can be tuned later (ticket 07).
- Self-hosting (DINOv2) is out: it would need infrastructure the team would have to run, whereas Gemini provides it with a free tier.
- Voyage's free trial (3 RPM, 10K TPM) makes the catalog batch unworkable without billing, and it has no paid batch for this model.

The decision rule's tiebreak (equal models: prefer Gemini, then Voyage, then local) points the same way, but accuracy was not measured, so no model is eliminated or confirmed against the 90 percent top-1 and 98 percent top-5 floor. That check moves to ticket 07 (tune the matcher on the full catalog) and must run on real photos before the matcher ships. If Gemini misses the floor there, the fallback is the OCR hybrid from the ticket's rule 5.

## What was measured (free tier, one machine, i7-1255U, 12 logical CPUs)

Query images for latency were catalog images resized like the frontend (1024 px long side, JPEG), a proxy, because no phone photo set exists. Rate-limit waiting is excluded from latency.

| Model | Dim | Latency p50 / p95 | Catalog (199 images) | Notes |
| --- | --- | --- | --- | --- |
| Gemini `gemini-embedding-2` | 1536 | 1.33 s / 1.56 s | 197 s wall, 1.01 images/s (5 x 429, 24.9 s waiting) | 258 tokens per image |
| Voyage `voyage-multimodal-3.5` | 1024 | 1.47 s / 3.24 s | 2,321 s wall, 0.086 images/s (2,022 s waiting, 0 x 429) | 1,878 tokens per original; free trial needs about 39 hours for 12,000 cards |
| DINOv2 base (CPU) | 768 | 0.58 s / 0.76 s | 2.17 images/s | 761 MB peak RSS, 3.84 CPU s per image |
| DINOv2 small (CPU) | 384 | 0.19 s / 0.28 s | 5.69 images/s | 514 MB peak RSS, 1.24 CPU s per image |

Catalog nearest-neighbour cosine (a crowded gallery makes a tight `confident` margin): Gemini p50 0.926, p95 0.966; Voyage p50 0.907, p95 0.973; DINOv2 base 0.797 / 0.886; small 0.792 / 0.899. Scores are not comparable across models.

Local model load (warm cache, excludes download): about 0.6 to 0.7 s in-process, 3.1 to 3.6 s including the torch import.

Smoke test on two images of one card (Sylveon ex, MA6 160, one full-frame photo and one rough crop): Gemini, Voyage and DINOv2 base ranked it first on both; DINOv2 small ranked it third on the full-frame photo. Two images of one card prove the pipeline, not accuracy. The runner-up for Gemini and Voyage was the other Sylveon ex (MA6 073), a near-duplicate, which is the tie case the `confident` margin must handle.

## Free-tier limits and the 12,000-card batch

- **Gemini:** developer-read AI Studio limits 100 RPM, 30K TPM, 1,000 RPD (not on a public page, not guaranteed). Observed: five 429s during the catalog run, cleared within about 20 s, so per-minute, but the body does not say which quota. Whether a `batchEmbedContents` call counts as one request or N is **not established**. At one request per image, 1,000 RPD means about 12 days for 12,000 cards; do not run the full catalog on the free tier. With the daily cap lifted, about 2 to 3 hours of wall time. Paid cost estimate (arithmetic from the pricing page, not measured): about $1.39 standard, about $0.70 on the Batch tier.
- **Voyage:** free trial 3 RPM and 10K TPM confirmed by the 429 body and by observation. A single 11.3K-token request was accepted; a 30K-token one was rejected. Throughput is bound by the 3 RPM window, about 39 hours for the full catalog.
- Privacy: free-tier content may be used by Google to improve its products; the spec already accepts this (`spec.md` Privacy section).

## Not done (open)

- Phone-photo accuracy: top-1, top-5, precision with coverage and the provisional `confident` rule, for all four models. Needs 40 photos of distinct MA6 cards in `spike/photos/` and `spike/labels.csv` (`photo_filename,set_code,local_number`). The commands are in `results/*/NOTES.md`; `uv run spike.py <model> photos` then `uv run evaluate.py <model>`.
- The paid pass for Voyage and Gemini (needs billing), including the Gemini Batch API turnaround. `providers/gemini_batch.py` exists but is untested against the API.
- Voyage `input_type` experiment, DINOv2 preprocessing comparison (`DINO_PREPROC=hf`), and Gemini 768 versus 1536 versus 3072 dimensions: all need photos.
- Latency on real phone photos and from the deployment environment (Railway is out of scope).
- Reprint tie rate on the full catalog (ticket 07).

## Reproducing

- Catalog CSV: `ma6-cards.csv` was exported by the developer from the `cards` table for set code `MA6` (199 rows; columns `id, local_id, attributes, created_at, updated_at, name, category, illustrator, tags, rarity_id, image_key`). The exact SQL was not recorded; recreate with a `SELECT` over `cards` joined to `expansion_sets` where `code = 'MA6'` and `image_key` is not empty.
- Image base address and keys live in `spike/.env` (git-ignored): `IMAGE_BASE_URL`, `GEMINI_API_KEY`, `VOYAGE_API_KEY`. No secret, photo or `labels.csv` is committed.
- Run: `cd .scratch/card-scanning/research/spike && uv sync && uv run spike.py <voyage|gemini|dino-base|dino-small> <catalog|latency|photos>`.

Next step: ticket 05 (embed the catalog) with Gemini, and ticket 07 for the accuracy check on real photos.
