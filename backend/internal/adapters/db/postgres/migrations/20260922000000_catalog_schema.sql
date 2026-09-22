-- +goose Up
CREATE TABLE games (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_games_slug ON games (slug);

-- locale is descriptive source metadata, not part of the natural key: see
-- docs/adr/0001 and CONTEXT.md's Expansion Set entry. (game_id, code) is
-- already unique per confirmed source behavior — a set code like "SV1V"
-- only ever exists under one locale for a given upstream source.
CREATE TABLE expansion_sets (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    game_id UUID NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    locale TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_expansion_sets_game_id_code ON expansion_sets (game_id, code);

-- names is a per-locale display-name map (e.g. {"id": "...", "ja": "..."})
-- since no single upstream call returns every locale at once. attributes
-- holds game-specific data (HP, types, attacks, ...) per docs/adr/0001;
-- rarity/image_url stay first-class columns since they're cross-game
-- concerns.
CREATE TABLE cards (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    expansion_set_id UUID NOT NULL REFERENCES expansion_sets (id) ON DELETE CASCADE,
    local_id TEXT NOT NULL,
    names JSONB NOT NULL DEFAULT '{}',
    rarity TEXT NOT NULL DEFAULT '',
    image_url TEXT NOT NULL DEFAULT '',
    attributes JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_cards_expansion_set_id_local_id ON cards (expansion_set_id, local_id);

-- finish is free text with a CHECK rather than a Postgres ENUM, so adding a
-- new finish later is a one-line migration instead of an ALTER TYPE.
CREATE TABLE card_variants (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    card_id UUID NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    finish TEXT NOT NULL CHECK (finish IN ('normal', 'reverse', 'holo', 'first_edition', 'w_promo')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_card_variants_card_id_finish ON card_variants (card_id, finish);

-- +goose Down
DROP TABLE IF EXISTS card_variants;
DROP TABLE IF EXISTS cards;
DROP TABLE IF EXISTS expansion_sets;
DROP TABLE IF EXISTS games;
