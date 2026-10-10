# Card image embeddings come from the hosted Gemini API and live in pgvector

Card scanning matches a phone photo to a Card by nearest-neighbour search over image embeddings of the catalog's hosted card images. We embed each Card's hosted image with Gemini `gemini-embedding-2` at 1536 dimensions and store the vector in Postgres with the pgvector extension, in a `card_embeddings` table with one row per Card (`card_id` primary key, cascading from `cards`), the `model` that produced the vector, and an HNSW index on cosine distance. A query compares only rows of one model, and `cmd/embed-catalog` re-embeds every Card whose row is missing or was made by another model, so changing `EMBEDDING_MODEL` re-embeds the catalog. The provider call sits behind a small `Embedder` interface (an adapter, since Voyage and a self-hosted model were real alternatives) that tests fake; CI never calls Gemini.

The choice is an operator decision on operational grounds, not the outcome of the accuracy spike (`.scratch/card-scanning/research/accuracy-spike-report.md`): the 40-photo accuracy run was skipped, so no model was measured against the 90 percent top-1 and 98 percent top-5 floor. Operational limits are fixed by the provider while accuracy can be tuned, so ticket 07 must check Gemini on real phone photos before the matcher ships. If it misses the floor, the fallback is the OCR hybrid from the spec.

1536 is the largest size Gemini recommends (768, 1536, 3072) that stays under pgvector's 2000-dimension limit for an HNSW index on `vector`. Smaller sizes stay possible without re-embedding, because the model's output is Matryoshka-truncatable, but a truncated vector must be L2 re-normalised first. Ticket 07 tunes the final dimension on the full catalog.

## Considered Options

- Voyage `voyage-multimodal-3.5` (1024 dimensions). Rejected: its free trial allows 3 requests a minute and 10K tokens a minute, which needs about 39 hours for the 12,000-card catalog, and it has no paid batch for this model.
- A self-hosted DINOv2 model service. Rejected: it needs infrastructure we would have to run, while Gemini is hosted with a free tier. It stays the fallback if a hosted provider cannot meet the accuracy floor.
- A vector database outside Postgres. Rejected: the catalog is about 12,000 vectors, Neon already hosts our data and offers pgvector, and one store keeps the card join and the backup story simple.

## Consequences

Gemini's free tier is about 100 requests a minute and 1,000 a day (developer-read from AI Studio, not guaranteed), so the first full embedding of the catalog needs about 12 days of daily reruns on the free tier or a paid key. The job paces itself under the per-minute limit, stops cleanly when the daily quota is exhausted, and a rerun resumes because embedded Cards are skipped.

Content sent on the free tier may be used by Google to improve its products. Catalog images are public, so embedding them is fine, but a paid key is needed before real user photos go to the provider at request time (ticket 06); the spec accepts the free-tier terms for the owner's own scans and says to revisit if anyone else starts scanning.

The migration runs `CREATE EXTENSION IF NOT EXISTS vector`, so the migrating role needs permission to create it and the Postgres used by CI, end-to-end runs and local tests must ship pgvector (`pgvector/pgvector:pg18` instead of `postgres:18`). Neon's production branch lists `vector` as available (0.8.x). Settings and the run book are in `docs/agents/deployment/embeddings.md`.
