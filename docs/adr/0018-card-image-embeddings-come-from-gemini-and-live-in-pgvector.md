# Card image embeddings come from the hosted Gemini API and live in pgvector

Card scanning matches a phone photo to a Card by nearest-neighbour search over image embeddings of the catalog's hosted card images. We embed each Card's hosted image with Gemini `gemini-embedding-2` at 1536 dimensions, through the official Go SDK (`google.golang.org/genai`) and its Batch API on a paid key, and store the vectors in Postgres with the pgvector extension.

The `card_embeddings` table holds one row per Card, model and source (`UNIQUE (card_id, model, source)`): the vector, the `model` that produced it and a `source` naming the image that was embedded (`catalog` today). A Card can therefore hold several embeddings, for example of more than one image of the same card, which a later change can add without a migration. There is an HNSW index on cosine distance. A query compares only rows of one model, and the matcher takes the best row per Card (ticket 06). Changing `EMBEDDING_MODEL` makes every Card eligible again, because the pending query looks for a row of the configured model.

The Batch API is asynchronous (a job can take up to a day), so the work is two re-runnable commands. `cmd/embed-catalog` submits: it picks Cards with a hosted image that have no embedding for the model and are not already in an outstanding batch, uploads their images in chunks of 500, and records each batch job in `embedding_batches` and `embedding_batch_cards`. `cmd/collect-embeddings` collects: for each outstanding batch it reads the job state, stores the vectors of a finished job (matched to Cards by key, never by position), marks a failed, cancelled or expired job failed so its Cards are submitted again, and marks a batch it cannot read for 72 hours failed too. The provider sits behind a small `BatchEmbedder` interface (an adapter, since Voyage and a self-hosted model were real alternatives) and the image download behind `ImageFetcher`; tests fake both and CI never calls Gemini.

The choice is an operator decision on operational grounds, not the outcome of the accuracy spike (`.scratch/card-scanning/research/accuracy-spike-report.md`): the 40-photo accuracy run was skipped, so no model was measured against the 90 percent top-1 and 98 percent top-5 floor. Ticket 07 must check Gemini on real phone photos before the matcher ships. If it misses the floor, the fallback is the OCR hybrid from the spec.

1536 is the largest size Gemini recommends (768, 1536, 3072) that stays under pgvector's 2000-dimension limit for an HNSW index on `vector`. Smaller sizes stay possible without re-embedding, because the model's output is Matryoshka-truncatable, but a truncated vector must be L2 re-normalised first. Ticket 07 tunes the final dimension on the full catalog.

## Considered Options

- Voyage `voyage-multimodal-3.5` (1024 dimensions). Rejected: its free trial allows 3 requests a minute and 10K tokens a minute, which needs about 39 hours for the 12,000-card catalog, and it has no paid batch for this model.
- A self-hosted DINOv2 model service. Rejected: it needs infrastructure we would have to run, while Gemini is hosted. It stays the fallback if a hosted provider cannot meet the accuracy floor.
- A vector database outside Postgres. Rejected: the catalog is about 12,000 vectors, Neon already hosts our data and offers pgvector, and one store keeps the card join and the backup story simple.
- The free tier with per-image `embedContent` calls, paced under the rate limit. Superseded: it allows about 1,000 requests a day, so the full catalog needs about 12 daily reruns plus pacing and quota-stop code. A paid key with the Batch API removes that code and costs roughly a dollar for the whole catalog (arithmetic from the pricing page, not measured).
- A hand-written REST client. Superseded by the official SDK, which owns authentication, uploads and retries.

## Consequences

The batch commands need a Gemini key from a project with billing enabled: the Batch API refuses a free-tier key (a text-only test batch on the spike's free-tier key failed with `FAILED_PRECONDITION` on both embedding models). A paid key also means content sent is not used by Google to improve its products, which removes the free-tier privacy caveat for user photos at request time (ticket 06); check the current terms before relying on it.

A real run on 11 cards of Expansion Set MA6 confirmed the batch design: an embeddings batch accepts PNG image parts, the uploaded JSONL input and the result lines parse as written, and the job finished in about three minutes. The vectors came back 1536-dimensional and unit length, and matched the earlier per-image `embedContent` results for the same cards. The output parsing stays isolated in one function (`resultLine` in `internal/adapters/embedding/gemini.go`) because the Go SDK marks `CreateEmbeddings` experimental and the result format is not documented for images.

Two concurrent `embed-catalog` runs would submit the same Cards twice (no rows are duplicated, the work and cost are). There is one operator, so no lock is taken.

The migration runs `CREATE EXTENSION IF NOT EXISTS vector`, so the migrating role needs permission to create it and the Postgres used by CI, end-to-end runs and local tests must ship pgvector (`pgvector/pgvector:pg18` instead of `postgres:18`). Neon's production branch lists `vector` as available (0.8.x). Settings and the run book are in `docs/agents/deployment/embeddings.md`.
