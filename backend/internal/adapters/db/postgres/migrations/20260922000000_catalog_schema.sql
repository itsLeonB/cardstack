-- +goose Up
CREATE TABLE games (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_games_slug ON games (slug);

-- locales is shared reference data (an upstream source's locale codes, e.g.
-- "id", "ja"), not owned by any one game/set.
CREATE TABLE locales (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    code TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_locales_code ON locales (code);

-- locale_id is descriptive source metadata, not part of the natural key: see
-- docs/adr/0001 and CONTEXT.md's Expansion Set entry. (game_id, code) is
-- already unique per confirmed source behavior — a set code like "SV1V"
-- only ever exists under one locale for a given upstream source. No
-- ON DELETE CASCADE on locale_id: locales is shared reference data, so
-- deleting a referenced locale should error, not silently cascade-delete
-- expansion sets.
CREATE TABLE expansion_sets (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    game_id UUID NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    locale_id UUID NOT NULL REFERENCES locales (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_expansion_sets_game_id_code ON expansion_sets (game_id, code);

-- names is a per-locale display-name map (e.g. {"id": "...", "ja": "..."})
-- since no single upstream call returns every locale at once. attributes
-- holds game-specific data (HP, types, attacks, ...) per docs/adr/0001;
-- rarity/image_url stay first-class columns since they're cross-game
-- concerns. raw is the full upstream response as-is, kept so a future need
-- for another field doesn't require re-ingesting historical cards.
CREATE TABLE cards (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    expansion_set_id UUID NOT NULL REFERENCES expansion_sets (id) ON DELETE CASCADE,
    local_id TEXT NOT NULL,
    names JSONB NOT NULL DEFAULT '{}',
    rarity TEXT NOT NULL DEFAULT '',
    image_url TEXT NOT NULL DEFAULT '',
    attributes JSONB NOT NULL DEFAULT '{}',
    raw JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_cards_expansion_set_id_local_id ON cards (expansion_set_id, local_id);

-- finishes is shared reference data (the closed set of print finishes, e.g.
-- "holo", "reverse"), normalized out of a CHECK-constrained free-text
-- column so a new finish is a row insert, not a migration.
CREATE TABLE finishes (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    code TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_finishes_code ON finishes (code);

-- No ON DELETE CASCADE on finish_id: finishes is shared reference data, same
-- reasoning as locale_id above.
CREATE TABLE card_variants (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    card_id UUID NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    finish_id UUID NOT NULL REFERENCES finishes (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_card_variants_card_id_finish_id ON card_variants (card_id, finish_id);

-- +goose Down
DROP TABLE IF EXISTS card_variants;
DROP TABLE IF EXISTS finishes;
DROP TABLE IF EXISTS cards;
DROP TABLE IF EXISTS expansion_sets;
DROP TABLE IF EXISTS locales;
DROP TABLE IF EXISTS games;
