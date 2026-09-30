-- +goose Up

-- Inventory Entries (ticket 07): how many copies of one Card sit in one
-- Collection. A Collection's summed quantity is capped by
-- collections.max_card_count (0 = no limit), enforced by the service on write.
CREATE TABLE inventory_entries (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    collection_id UUID NOT NULL REFERENCES collections (id) ON DELETE CASCADE,
    card_id UUID NOT NULL REFERENCES cards (id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (collection_id, card_id)
);

-- +goose Down
DROP TABLE IF EXISTS inventory_entries;
