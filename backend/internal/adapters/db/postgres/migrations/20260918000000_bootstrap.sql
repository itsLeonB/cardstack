-- +goose Up
-- Placeholder: no schema changes yet. Keeps at least one file here so
-- `//go:embed *.sql` in embed.go has something to embed and the
-- goose/migrator pipeline is provable end to end. Replace once real schema
-- migrations exist; PG18's native uuidv7()/gen_random_uuid() mean no
-- extension (e.g. pgcrypto) is needed for UUID primary keys.

-- +goose Down
