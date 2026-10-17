-- +goose Up
-- source_logo_url is the scraped address of an Expansion Set's logo and logo_key
-- its hosted copy (docs/adr/0016), filled by host-images. Logos are hosted as is,
-- not pre-sized (ADR-0017 covers only covers). Additive only: existing rows default
-- to empty, and the running API serves the cover until it is redeployed.
ALTER TABLE expansion_sets ADD COLUMN source_logo_url text NOT NULL DEFAULT '';
ALTER TABLE expansion_sets ADD COLUMN logo_key text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE expansion_sets DROP COLUMN logo_key;
ALTER TABLE expansion_sets DROP COLUMN source_logo_url;
