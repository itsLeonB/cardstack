-- +goose Up

-- The offer revision an accept compares against (spec: two nullable
-- acceptance timestamps beside the status column). It moves only when the
-- offer does — item edits bump it atomically with the acceptances clear —
-- so an acceptance, which updates the row's updated_at, cannot stale the
-- other seat's pending accept. updated_at keeps its record-changed meaning.
ALTER TABLE trades
    ADD COLUMN offer_revision TIMESTAMPTZ NOT NULL DEFAULT now();

-- +goose Down

ALTER TABLE trades
    DROP COLUMN offer_revision;
