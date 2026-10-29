-- +goose Up

-- The Slug in a profile's shared link (docs/adr/0025): the name lower-cased and
-- hyphenated, then the last 5 hex characters of the id (the leading ones are a
-- uuidv7 timestamp, near-constant across rows). It must match mapper.SlugOf,
-- which makes new profiles' Slugs; a name with no a-z or 0-9 becomes 'user'.
ALTER TABLE user_profiles ADD COLUMN slug TEXT;

UPDATE user_profiles
SET slug = coalesce(nullif(trim(BOTH '-' FROM regexp_replace(lower(name), '[^a-z0-9]+', '-', 'g')), ''), 'user')
    || '-' || right(id::text, 5);

ALTER TABLE user_profiles ALTER COLUMN slug SET NOT NULL;

CREATE UNIQUE INDEX idx_user_profiles_slug ON user_profiles (slug);

-- +goose Down

DROP INDEX IF EXISTS idx_user_profiles_slug;

ALTER TABLE user_profiles DROP COLUMN IF EXISTS slug;
