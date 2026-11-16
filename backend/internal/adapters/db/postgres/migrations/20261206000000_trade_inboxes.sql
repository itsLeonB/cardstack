-- +goose Up

ALTER TABLE trades
    ADD COLUMN owner_confirmed_at TIMESTAMPTZ,
    ADD COLUMN participant_confirmed_at TIMESTAMPTZ;

ALTER TABLE collections
    ADD COLUMN trade_id UUID REFERENCES trades (id) ON DELETE SET NULL;

CREATE UNIQUE INDEX idx_collections_profile_trade_id
    ON collections (profile_id, trade_id)
    WHERE trade_id IS NOT NULL;

-- +goose Down

ALTER TABLE trades
    DROP COLUMN participant_confirmed_at,
    DROP COLUMN owner_confirmed_at;

ALTER TABLE collections
    DROP COLUMN trade_id;
