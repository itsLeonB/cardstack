-- +goose Up

-- The Slug in a profile's shared link (docs/adr/0025): the name lower-cased and
-- hyphenated, then the last 5 hex characters of the id (the leading ones are a
-- uuidv7 timestamp, near-constant across rows). It must match mapper.SlugOf,
-- which makes new profiles' Slugs; a name with no a-z or 0-9 becomes 'user'.
-- The name is cut to its first 60 characters before that, as SlugOf cuts it to
-- maxSlugTextRunes; change both together.
-- Two profiles can share a stem and those 5 characters. Every member of such a
-- group takes the full id, without hyphens, as its suffix: 32 characters, so it
-- cannot equal a 5-character suffix that mapper.SlugOf makes later.
ALTER TABLE user_profiles ADD COLUMN slug TEXT;

WITH stems AS (
    SELECT id, coalesce(nullif(trim(BOTH '-' FROM regexp_replace(lower(left(name, 60)), '[^a-z0-9]+', '-', 'g')), ''), 'user') AS stem
    FROM user_profiles
), candidates AS (
    SELECT id, stem, stem || '-' || right(id::text, 5) AS slug
    FROM stems
), counted AS (
    SELECT id, stem, slug, count(*) OVER (PARTITION BY slug) AS clashes
    FROM candidates
)
UPDATE user_profiles p
SET slug = CASE WHEN c.clashes > 1 THEN c.stem || '-' || replace(p.id::text, '-', '') ELSE c.slug END
FROM counted c
WHERE c.id = p.id;

ALTER TABLE user_profiles ALTER COLUMN slug SET NOT NULL;

CREATE UNIQUE INDEX idx_user_profiles_slug ON user_profiles (slug);

-- +goose Down

DROP INDEX IF EXISTS idx_user_profiles_slug;

ALTER TABLE user_profiles DROP COLUMN IF EXISTS slug;
