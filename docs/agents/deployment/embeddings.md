# Card embeddings

The scan matcher searches image embeddings of the catalog's hosted card images (ADR-0018). `cmd/embed-catalog` produces them. Read this when the job logs a quota or key error, when `card_embeddings` is empty, or when you change the model.

## Settings

Read only by `cmd/embed-catalog`, so the API boots without them. Local values: `backend/.env.example`.

- `GEMINI_API_KEY`: a Gemini API key from Google AI Studio. Required by the command. The free tier is enough for a seed set, a full first run needs a paid key or about 12 daily reruns (below).
- `EMBEDDING_MODEL`: the model whose embeddings are stored, default `gemini-embedding-2`. It is saved on every row. Changing it makes every Card eligible for re-embedding on the next run, and a query only compares rows of one model. The vector size is fixed at 1536 by the table, so only a model that can return 1536 dimensions works without a migration.
- `IMAGE_BASE_URL`: the command downloads `IMAGE_BASE_URL` plus a Card's hosted key (see `docs/agents/deployment/images.md`). A Card with no hosted key is skipped and counted.

## Running it

```
make embed-catalog                  # every Card with a hosted image
make embed-catalog ARGS="-set MA6"  # one Expansion Set, for a seed run
```

It logs one line per Card, a skipped count (already embedded by the current model, or no hosted image) and a closing summary. A failure on one Card is logged and the run continues. It paces itself under 90 requests a minute, backs off on a 429, and stops the whole run when Gemini reports the daily quota exhausted. Rerun it later: embedded Cards are skipped, so it resumes where it stopped. It exits 0 on success or a quota stop, and 1 if any Card failed or on a fatal error such as a missing `GEMINI_API_KEY` or `IMAGE_BASE_URL`.

The free tier allows about 1,000 requests a day, so the full catalog takes about 12 daily reruns on it. Content sent on the free tier may be used by Google to improve its products; catalog images are public, but see ADR-0018 before sending user photos.

`backend/.env` in a worktree is a copy of the main checkout's file and can point at the production database. Export `DB_HOST`, `DB_PORT` and the other `DB_*` values explicitly when you want a seed run against a local database, and check which database a command reads before running it.

## Enabling pgvector

The migration `20261010000000_card_embeddings.sql` runs `CREATE EXTENSION IF NOT EXISTS vector`. Production is Neon, whose `production` branch lists `vector` as available (0.8.x), so the role that runs `make job` only needs permission to create extensions (the project owner role has it). Apply it like any migration, before deploying anything that reads `card_embeddings`:

1. Run `make job` against the production database.
2. Run `make embed-catalog` against it with `GEMINI_API_KEY` set, and repeat daily until the summary reports nothing left to embed.

The migration's Down drops the table and the extension.

Local and CI Postgres must ship pgvector: use `pgvector/pgvector:pg18` where `postgres:18` was used (`docker run -d -p 5432:5432 -e POSTGRES_USER=cardstack -e POSTGRES_PASSWORD=cardstack -e POSTGRES_DB=cardstack pgvector/pgvector:pg18`). The CI workflows already do.
