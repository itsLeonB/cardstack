-- +goose Up
-- published hides an Expansion Set and its Cards from everyone but an Admin
-- (GLOSSARY.md Published). Existing sets stay published, and so does any new
-- set that does not say otherwise.
ALTER TABLE expansion_sets ADD COLUMN published boolean NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE expansion_sets DROP COLUMN published;
