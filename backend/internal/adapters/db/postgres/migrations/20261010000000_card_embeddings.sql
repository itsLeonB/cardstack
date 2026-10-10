-- +goose Up
-- pgvector stores the image embeddings the card scan matches against. The
-- extension must be installable by the migrating role on the target database.
CREATE EXTENSION IF NOT EXISTS vector;

-- One row per card: the embedding of its hosted image, made by the model named
-- in model. A query only compares rows of one model, and re-embedding with
-- another model replaces the row in place. vector(1536) is the largest size
-- Gemini recommends that stays under pgvector's 2000-dimension index limit.
-- Truncating to a smaller size later needs an L2 re-normalise first.
CREATE TABLE card_embeddings (
    card_id UUID PRIMARY KEY REFERENCES cards (id) ON DELETE CASCADE,
    model TEXT NOT NULL,
    embedding vector(1536) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_card_embeddings_embedding ON card_embeddings USING hnsw (embedding vector_cosine_ops);

-- +goose Down
DROP TABLE IF EXISTS card_embeddings;
DROP EXTENSION IF EXISTS vector;
