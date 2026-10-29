-- +goose Up

-- The Slug in a Shared Collection's link (docs/adr/0025). NULL while the
-- Collection is private, so a NULL here is what makes it private. Unique per
-- owner; Postgres counts NULLs as distinct, so private Collections never clash.
ALTER TABLE collections ADD COLUMN slug TEXT;

CREATE UNIQUE INDEX idx_collections_profile_slug ON collections (profile_id, slug);

-- +goose Down

DROP INDEX IF EXISTS idx_collections_profile_slug;

ALTER TABLE collections DROP COLUMN IF EXISTS slug;
