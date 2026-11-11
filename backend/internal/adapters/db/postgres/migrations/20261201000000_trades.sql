-- +goose Up

-- The Trade (GLOSSARY.md: Trade): two seats, each a user_profiles row. The
-- creator holds one side from the start; the second seat is a nullable
-- participant claimed atomically by the first join (spec: a third opener is
-- told the Trade is full). slug is the link's address and is permanent for
-- the Trade's life; a new Trade draws a new one.
CREATE TABLE trades (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    slug TEXT NOT NULL UNIQUE,
    owner_profile_id UUID NOT NULL REFERENCES user_profiles (id) ON DELETE CASCADE,
    participant_profile_id UUID REFERENCES user_profiles (id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'negotiating', 'locked', 'executed', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Trades by participant, either seat.
CREATE INDEX idx_trades_owner_profile_id ON trades (owner_profile_id);
CREATE INDEX idx_trades_participant_profile_id ON trades (participant_profile_id);

-- The Trade Item (GLOSSARY.md: Trade Item): one entry on a Trade side — a
-- Card, a quantity, and the Collection it is meant to leave (source). side
-- is 0 for the owner's offer, 1 for the participant's. (trade_id, side,
-- card_id, source_collection_id) is unique: the same Card may appear twice on
-- one side from two Collections, but not twice from the same one.
CREATE TABLE trade_items (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    trade_id UUID NOT NULL REFERENCES trades (id) ON DELETE CASCADE,
    side SMALLINT NOT NULL CHECK (side IN (0, 1)),
    card_id UUID NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    source_collection_id UUID NOT NULL REFERENCES collections (id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL CHECK (quantity >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT trade_items_trade_side_card_source_key UNIQUE (trade_id, side, card_id, source_collection_id)
);

CREATE INDEX idx_trade_items_trade_id ON trade_items (trade_id);

-- +goose Down

DROP TABLE IF EXISTS trade_items;
DROP TABLE IF EXISTS trades;
