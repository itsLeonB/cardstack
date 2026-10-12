-- +goose Up
-- curated marks a row written by hand through the admin; ingestion never
-- overwrites it (docs/adr/0019). Additive only: existing rows default to false
-- and the running API ignores the column until it is redeployed.
ALTER TABLE cards ADD COLUMN curated boolean NOT NULL DEFAULT false;
ALTER TABLE expansion_sets ADD COLUMN curated boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE expansion_sets DROP COLUMN curated;
ALTER TABLE cards DROP COLUMN curated;
