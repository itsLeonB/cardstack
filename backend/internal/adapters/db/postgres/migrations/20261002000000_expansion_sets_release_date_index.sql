-- +goose Up

-- Added proactively for the most-recent-first release date ordering of
-- Expansion Set listings (the Series ordering sorts on a derived MIN and the
-- Card search on a join, so they may not use it directly).
CREATE INDEX idx_expansion_sets_release_date ON expansion_sets (release_date DESC NULLS LAST);

-- +goose Down
DROP INDEX IF EXISTS idx_expansion_sets_release_date;
