-- +goose Up

-- Reprint lookup (docs/adr/0023) finds Cards whose name matches a source's
-- after case and whitespace are folded. The index is on that exact expression,
-- so the query compares the indexed value rather than scanning every card.
CREATE INDEX idx_cards_normalized_name ON cards (btrim(regexp_replace(lower(name), '\s+', ' ', 'g')));

-- +goose Down
DROP INDEX IF EXISTS idx_cards_normalized_name;
