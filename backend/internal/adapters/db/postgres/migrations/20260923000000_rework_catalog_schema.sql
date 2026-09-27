-- +goose Up

-- ticket 03's TCGDex-sourced cards/sets are wholly superseded by this
-- ticket's pokemonasia scraper (TCGDex data was already flagged outdated
-- when this ticket was written); cleared here rather than column-migrated
-- so no stale TCGDex rows survive alongside fresh pokemonasia rows.
-- `games`/`locales` rows are kept — the ingester finds the same
-- `pokemon-tcg`/`id` rows by natural key rather than duplicating them.
DELETE FROM cards;
DELETE FROM expansion_sets;

DROP TABLE card_variants;
DROP TABLE finishes;

CREATE TABLE series (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    game_id UUID NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_series_game_id_code ON series (game_id, code);

-- rarities: per-Game, not seeded — the ingester find-or-creates a row per
-- source rarity code it encounters. See ADR-0009: scoped by Game because
-- the codes are Pokémon-specific (unlike locale/finish, which were shared
-- reference data); not scoped finer than Game because the roster rotates
-- per print era (presence/absence) without any code changing meaning.
CREATE TABLE rarities (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    game_id UUID NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_rarities_game_id_code ON rarities (game_id, code);

ALTER TABLE expansion_sets
    ADD COLUMN series_id UUID REFERENCES series (id) ON DELETE SET NULL,
    ADD COLUMN release_date DATE;
CREATE INDEX idx_expansion_sets_series_id ON expansion_sets (series_id);

ALTER TABLE cards
    DROP COLUMN names,
    DROP COLUMN rarity,
    ADD COLUMN name TEXT NOT NULL,
    ADD COLUMN category TEXT NOT NULL,
    ADD COLUMN illustrator TEXT NOT NULL DEFAULT '',
    ADD COLUMN tags JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN rarity_id UUID NOT NULL REFERENCES rarities (id);
CREATE INDEX idx_cards_category ON cards (category);
CREATE INDEX idx_cards_rarity_id ON cards (rarity_id);
CREATE INDEX idx_cards_tags ON cards USING GIN (tags jsonb_path_ops);

-- +goose Down
ALTER TABLE cards
    DROP COLUMN rarity_id,
    DROP COLUMN tags,
    DROP COLUMN illustrator,
    DROP COLUMN category,
    DROP COLUMN name,
    ADD COLUMN names JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN rarity TEXT NOT NULL DEFAULT '';

ALTER TABLE expansion_sets
    DROP COLUMN release_date,
    DROP COLUMN series_id;

DROP TABLE rarities;
DROP TABLE series;

CREATE TABLE finishes (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    code TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_finishes_code ON finishes (code);

CREATE TABLE card_variants (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    card_id UUID NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    finish_id UUID NOT NULL REFERENCES finishes (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_card_variants_card_id_finish_id ON card_variants (card_id, finish_id);
