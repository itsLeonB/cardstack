-- E2E test fixture for the frontend's Playwright suite. Deterministic and
-- synthetic (not real scraped data): kept independent of ticket 04's live
-- scraper output so e2e assertions never silently break if that output
-- changes, and self-contained under its own Game/Locale so it never
-- collides with real Pokémon TCG data seeded by the ingester, or with
-- other tests' leftover rows in a shared database.
--
-- Idempotent (ON CONFLICT DO NOTHING keyed by each table's natural key), so
-- it's safe to run more than once against the same database.
--
-- Covers every catalog browse/search facet (ticket 05) plus the
-- series-less Expansion Set fix: two Rarities, two Categories, a Tag,
-- exact name/set+number lookups, and one ungrouped Expansion Set.
--
-- Run after migrations, before starting the API server, e.g.:
--   psql "$DATABASE_URL" -f internal/adapters/db/postgres/testdata/e2e_seed.sql

INSERT INTO games (slug, name) VALUES ('e2e-test-game', 'E2E Test Game')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO locales (code) VALUES ('e2e')
ON CONFLICT (code) DO NOTHING;

INSERT INTO series (game_id, code, name)
SELECT g.id, 'test-series', 'Test Series Alpha'
FROM games g
WHERE g.slug = 'e2e-test-game'
ON CONFLICT (game_id, code) DO NOTHING;

INSERT INTO rarities (game_id, code, name)
SELECT g.id, 'C', 'Common'
FROM games g
WHERE g.slug = 'e2e-test-game'
ON CONFLICT (game_id, code) DO NOTHING;

INSERT INTO rarities (game_id, code, name)
SELECT g.id, 'R', 'Rare'
FROM games g
WHERE g.slug = 'e2e-test-game'
ON CONFLICT (game_id, code) DO NOTHING;

-- Grouped Expansion Set, under Test Series Alpha.
INSERT INTO expansion_sets (game_id, code, name, locale_id, series_id, release_date)
SELECT g.id, 'TSA', 'Test Set Alpha', l.id, s.id, DATE '2020-01-01'
FROM games g
JOIN locales l ON l.code = 'e2e'
JOIN series s ON s.game_id = g.id AND s.code = 'test-series'
WHERE g.slug = 'e2e-test-game'
ON CONFLICT (game_id, code) DO NOTHING;

-- Ungrouped Expansion Set (no Series) — exercises the fix that surfaces
-- these through GET /catalog/series's ungroupedExpansionSets.
INSERT INTO expansion_sets (game_id, code, name, locale_id, series_id, release_date)
SELECT g.id, 'TSU', 'Ungrouped Test Set', l.id, NULL, NULL
FROM games g
JOIN locales l ON l.code = 'e2e'
WHERE g.slug = 'e2e-test-game'
ON CONFLICT (game_id, code) DO NOTHING;

-- Cards in Test Set Alpha (TSA):
--   001 E2E Sparky        / Creature / Common / tags: [Basic]
--   002 E2E Boulder       / Creature / Rare   / tags: [Evolved]
--   003 E2E Trainer Card  / Support  / Common / tags: []
-- image_url is left empty on every fixture Card so the UI renders its
-- name-placeholder fallback deterministically, with no network dependency.
INSERT INTO cards (expansion_set_id, local_id, name, category, illustrator, tags, rarity_id, image_url)
SELECT es.id, '001', 'E2E Sparky', 'Creature', 'E2E Illustrator', '["Basic"]'::jsonb, r.id, ''
FROM expansion_sets es
JOIN games g ON g.id = es.game_id
JOIN rarities r ON r.game_id = g.id AND r.code = 'C'
WHERE g.slug = 'e2e-test-game' AND es.code = 'TSA'
ON CONFLICT (expansion_set_id, local_id) DO NOTHING;

INSERT INTO cards (expansion_set_id, local_id, name, category, illustrator, tags, rarity_id, image_url)
SELECT es.id, '002', 'E2E Boulder', 'Creature', 'E2E Illustrator', '["Evolved"]'::jsonb, r.id, ''
FROM expansion_sets es
JOIN games g ON g.id = es.game_id
JOIN rarities r ON r.game_id = g.id AND r.code = 'R'
WHERE g.slug = 'e2e-test-game' AND es.code = 'TSA'
ON CONFLICT (expansion_set_id, local_id) DO NOTHING;

INSERT INTO cards (expansion_set_id, local_id, name, category, illustrator, tags, rarity_id, image_url)
SELECT es.id, '003', 'E2E Trainer Card', 'Support', 'E2E Illustrator', '[]'::jsonb, r.id, ''
FROM expansion_sets es
JOIN games g ON g.id = es.game_id
JOIN rarities r ON r.game_id = g.id AND r.code = 'C'
WHERE g.slug = 'e2e-test-game' AND es.code = 'TSA'
ON CONFLICT (expansion_set_id, local_id) DO NOTHING;

-- Card in the ungrouped set (TSU):
--   001 E2E Orphan Card / Creature / Common / tags: [Basic]
INSERT INTO cards (expansion_set_id, local_id, name, category, illustrator, tags, rarity_id, image_url)
SELECT es.id, '001', 'E2E Orphan Card', 'Creature', 'E2E Illustrator', '["Basic"]'::jsonb, r.id, ''
FROM expansion_sets es
JOIN games g ON g.id = es.game_id
JOIN rarities r ON r.game_id = g.id AND r.code = 'C'
WHERE g.slug = 'e2e-test-game' AND es.code = 'TSU'
ON CONFLICT (expansion_set_id, local_id) DO NOTHING;
