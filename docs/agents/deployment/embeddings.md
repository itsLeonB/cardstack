# Card embeddings

The scan matcher searches image embeddings of the catalog's hosted card images (ADR-0018). Two commands produce them: `cmd/embed-catalog` submits Gemini batch jobs and `cmd/collect-embeddings` stores their results. `POST /scan/match` embeds the uploaded photo with the same model and searches those vectors. Read this when `card_embeddings` is empty, when a batch is stuck, when you change the model, or when scans fail or are never confident.

## Settings

Local values: `backend/.env.example`.

- `GEMINI_API_KEY`: a Gemini API key from a Google AI Studio project **with billing enabled**. The Batch API refuses a free-tier key with `FAILED_PRECONDITION` (HTTP 400, "Precondition check failed"), and a free-tier key may let Google use users' photos to improve its products. Required by both commands and by the API's scan match. The API still boots without it, because previews and CI have none, but it logs `GEMINI_API_KEY is empty` and every scan answers 500.
- `EMBEDDING_MODEL`: the model whose embeddings are stored and searched, default `gemini-embedding-2`. It is saved on every row. Changing it makes every Card eligible for a new embedding on the next submit, and a query only compares rows of one model, so scans find nothing until the new model's catalog embeddings are collected. The vector size is fixed at 1536 by the table, so only a model that can return 1536 dimensions works without a migration.
- `MATCH_THRESHOLD` (default `0.85`) and `MATCH_MARGIN` (default `0.03`): when a scan is `confident`. The top Card's cosine similarity (1 is identical) must reach the threshold, and the second Card must trail it by at least the margin; otherwise the response lists up to five candidates and the user picks. The API refuses to boot with a threshold outside 0 to 1 (0 excluded) or a margin outside 0 to 1 (1 excluded). A near-tie, such as one artwork reprinted in two Expansion Sets, fails the margin by design. **The defaults are provisional**: ticket 07 measures them on the full catalog and real phone photos and replaces them. If scans are never confident, lower the threshold or margin; if a wrong Card is ever added without asking, raise them. A change needs an API restart, not a frontend release.
- `MATCH_EMPTY_THRESHOLD` (default `0`, off): when the match says a photo shows no card. If the top Card's cosine similarity is below this value, the response is `empty: true`, `confident: false` with no candidates, instead of the five nearest Cards. A top similarity equal to the value is not empty. The API refuses to boot with a value below 0 or at or above `MATCH_THRESHOLD`. **Provisional**: leave it at 0 until it is tuned from real photos of empty slots and backgrounds; a value that is too high reports real cards as empty. A change needs an API restart.
- `MATCH_REPRINT_DELTA` (default `0.05`): the largest cosine distance between two catalog images at which two Cards with the same normalized name count as the same artwork, for "Other prints" (`GET /scan/reprints`, ADR-0023). Between 0 and 1. **Provisional**: tune it from real groups of reprints with the `reprints` report (see "Tuning the empty threshold and the Reprint distance" below). A change needs an API restart.
- `IMAGE_BASE_URL`: `embed-catalog` downloads `IMAGE_BASE_URL` plus a Card's hosted key (see `docs/agents/deployment/images.md`). A Card with no hosted key is skipped and counted.

## How a scan is matched

`POST /scan/match` embeds the uploaded JPEG with one synchronous Gemini call (8 second timeout, under `APP_TIMEOUT`), takes the five nearest stored vectors of `EMBEDDING_MODEL` and returns those Cards best first. The photo is processed in memory and never stored. A Card with no embedding for the model cannot appear. A Gemini failure, quota error or timeout is logged in full and answered as a redacted 500, which the frontend shows as "Could not match this photo". Each scan is one billed embedding call, and the per-user rate limits apply.

Card scanning has no feature flag: the "Scan cards" button and the scan route ship in every frontend build. Before deploying to production, make sure the production catalog is embedded, `GEMINI_API_KEY` is set on the API, and a real photo has matched the right Card on the deployed API, otherwise scans answer 500 or never find a Card.

## Running it

```
make embed-catalog                  # submit: every Card with a hosted image that has no embedding yet
make embed-catalog ARGS="-set MA6"  # submit one Expansion Set, for a seed run
make collect-embeddings             # collect: store the vectors of finished batches
```

Submit downloads each image, uploads them in batches of 500, and records every batch job (`embedding_batches`, one row per job, `embedding_batch_cards` for its Cards). A Card in an outstanding batch is not submitted again, so rerunning submit while jobs are running is safe. Collect checks every outstanding batch: a finished job's vectors are validated (1536 entries, unit length) and stored, a running job is left for the next run, and a failed, cancelled or expired job is marked failed so its Cards are submitted again by the next `embed-catalog`. A batch that stays unreadable for 72 hours is marked failed the same way. A Card that comes back with an error, or with no result at all, is counted as failed and stays pending.

Both commands log one line per batch or Card with its position out of the total, then a closing summary, and exit 1 if anything failed. A Gemini job can take up to 24 hours, so the usual rhythm is submit, wait, collect, and repeat submit until its summary reports nothing left to embed. A batch holds its images in memory while it is built (roughly 0.8 GB for 500 images), so lower `embeddingBatchSize` in `backend/internal/domain/service/embedding_service.go` on a small machine.

Run one `embed-catalog` at a time: two runs at once submit the same Cards twice. No rows are duplicated, but the work and the cost are.

`backend/.env` in a worktree is a copy of the main checkout's file and can point at the production database. Export `DB_HOST`, `DB_PORT` and the other `DB_*` values explicitly when you want a seed run against a local database, and check which database a command reads before running it.

## Tuning the empty threshold and the Reprint distance

`cmd/tune-scan` is an offline report that sets `MATCH_EMPTY_THRESHOLD` and `MATCH_REPRINT_DELTA` from data instead of guesses (spec: `.scratch/scan-enhancements/spec.md`, Tuning). It is the Phase 1 accuracy script's successor for these two settings: the accuracy spike (`.scratch/card-scanning/research/spike/evaluate.py`) measured top-1 and top-5 and the `confident` rule, and `tune-scan` measures the empty and Reprint rules on the real catalog. It is run by hand, never in CI or on Railway, and it only reads the database. Point `DB_*` at the database that holds the embeddings and check which one a worktree's `backend/.env` names first.

```
make tune-scan ARGS="cells --photos DIR --grid 3x3 [--threshold 0.80] [--cut-dir DIR]"
make tune-scan ARGS="reprints [--limit 20] [--min-size 2]"
```

Both subcommands fail before spending anything when the catalog holds no embeddings of `EMBEDDING_MODEL`: run `make embed-catalog` and `make collect-embeddings` first.

**`cells`** (needs `GEMINI_API_KEY`, `bun` and the frontend dependencies installed with `bun install` in `frontend/`). `DIR` holds one JPEG per Page photo. The report cuts each photo into the grid with the app's own cutting code (`frontend/tools/cut-pages.ts`, the same function `cutPage` uses, after the same downscale), embeds every cell with Gemini, and prints one line per cell: photo, row, column, the top similarity, and the top Card's name and set code. The top similarity is exactly what `POST /scan/match` reports as the top candidate's score (1 minus the cosine distance to the nearest stored image of the model). Each photo then gets a summary: its lowest and highest top similarity and, with `--threshold`, how many cells fall strictly below it. A photo whose Page corners are not the whole photo takes a `<photo-name>.corners.json` beside it (`[[x,y],[x,y],[x,y],[x,y]]`, each 0 to 1, top-left, top-right, bottom-right, bottom-left); without one the whole photo is the Page, so frame the photo tightly or add the sidecar. `--cut-dir` keeps the cut cells so you can look at them. Each run costs one Gemini call per cell.

To set `MATCH_EMPTY_THRESHOLD`, run it on an empty Page, a sparse one and a full one. Empty pockets score low (nothing in them resembles a catalog image), filled ones score high. Read the highest top similarity among the pockets you know are empty and the lowest among the ones that hold a Card. Set the threshold between the two, closer to the empty side: a value that is too high reports real Cards as empty, and the cost of a too-low value is only that an empty pocket is offered candidates. Rerun with `--threshold` set to your candidate: the "below" count of the empty Page should be every empty pocket and the full Page's should be zero. The value must stay under `MATCH_THRESHOLD`.

**`reprints`** (needs no Gemini key). It groups Cards by the same normalized name the Reprint lookup uses (case and whitespace folded), keeps the Cards that have an embedding of the model, and prints the groups of at least `--min-size` Cards, largest first, at most `--limit` of them. For each group it prints the cosine distance between every pair of Cards (each Card shown as set code and local id), using the nearest pair of their embeddings, which is the distance the Reprint rule compares with `MATCH_REPRINT_DELTA`. The configured delta is printed at the top, and each pair is marked within or beyond it. A group of more than 8 Cards prints pairwise distances for its first 8 only and says so.

To set `MATCH_REPRINT_DELTA`, find groups you recognise: one artwork printed in several Expansion Sets should show small distances, while different artworks that share a name (a Pokemon with several illustrations) should show large ones. Set the delta above the distances of the same-artwork pairs and below those of the different-artwork pairs. If the two kinds overlap, prefer the lower value: a too-high delta offers a different artwork as a Reprint, while a too-low one only hides a real Reprint.

Changing either setting needs an API restart.

## Enabling pgvector

The migration `20261010000000_card_embeddings.sql` runs `CREATE EXTENSION IF NOT EXISTS vector`. Production is Neon, whose `production` branch lists `vector` as available (0.8.x), so the role that runs `make job` only needs permission to create extensions (the project owner role has it). Apply it like any migration, before deploying anything that reads `card_embeddings`:

1. Run `make job` against the production database.
2. Run `make embed-catalog` against it with `GEMINI_API_KEY` set, then `make collect-embeddings` once the jobs finish, and repeat until the submit summary reports nothing left.

The migration's Down drops the three tables and the extension.

Local and CI Postgres must ship pgvector: use `pgvector/pgvector:pg18` where `postgres:18` was used (`docker run -d -p 5432:5432 -e POSTGRES_USER=cardstack -e POSTGRES_PASSWORD=cardstack -e POSTGRES_DB=cardstack pgvector/pgvector:pg18`). The CI workflows already do.
