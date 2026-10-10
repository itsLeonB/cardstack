-- +goose Up
-- pgvector stores the image embeddings the card scan matches against. The
-- extension must be installable by the migrating role on the target database.
CREATE EXTENSION IF NOT EXISTS vector;

-- One row per card, model and source: the embedding of one image of the card
-- (source names the image, "catalog" today), made by one model. A card can hold
-- several rows, one per model or source, so a model change adds rows instead of
-- replacing them. The unique key also serves lookups by card_id, which leads it.
-- vector(1536) is the largest size Gemini recommends that stays under pgvector's
-- 2000-dimension index limit. Truncating to a smaller size later needs an L2
-- re-normalise first.
CREATE TABLE card_embeddings (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    card_id UUID NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    model TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'catalog',
    embedding vector(1536) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (card_id, model, source)
);

-- The index covers every model, so a query filtered to one model can return
-- fewer rows than its limit while two models coexist during a switch.
CREATE INDEX idx_card_embeddings_embedding ON card_embeddings USING hnsw (embedding vector_cosine_ops);

-- One row per batch job submitted to the Gemini batch API. state is submitted
-- while the job is outstanding, collected once its vectors are stored, and
-- failed when the job failed or was cancelled, which makes its cards eligible
-- for the next submit again.
CREATE TABLE embedding_batches (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    job_name TEXT NOT NULL UNIQUE,
    model TEXT NOT NULL,
    source TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('submitted', 'collected', 'failed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    collected_at TIMESTAMPTZ
);

-- The cards of each batch. A child table rather than a UUID[] column: the
-- outstanding-batch check is then a plain join, and every row is typed.
CREATE TABLE embedding_batch_cards (
    batch_id UUID NOT NULL REFERENCES embedding_batches (id) ON DELETE CASCADE,
    card_id UUID NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    PRIMARY KEY (batch_id, card_id)
);

-- +goose Down
DROP TABLE IF EXISTS embedding_batch_cards;
DROP TABLE IF EXISTS embedding_batches;
DROP TABLE IF EXISTS card_embeddings;
DROP EXTENSION IF EXISTS vector;
