-- +goose Up

-- Each seat's acceptance of the Trade as it stands (spec: two nullable
-- acceptance timestamps beside the status column). The second one moves the
-- Trade to locked; they never decide the state by themselves.
ALTER TABLE trades
    ADD COLUMN owner_accepted_at TIMESTAMPTZ,
    ADD COLUMN participant_accepted_at TIMESTAMPTZ;

-- +goose Down

ALTER TABLE trades
    DROP COLUMN participant_accepted_at,
    DROP COLUMN owner_accepted_at;
