-- +goose Up

-- Collections (ticket 06): a profile-owned, named grouping of Cards with
-- quantities (a binder/box/deck; see CONTEXT.md's Collection entry).
-- max_card_count is an optional hard cap on the collection's summed
-- Inventory Entry quantities, enforced when Inventory Entries are written
-- (a later ticket) - nothing in this schema tracks quantities yet.
CREATE TABLE collections (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    profile_id UUID NOT NULL REFERENCES user_profiles (id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    max_card_count INTEGER CHECK (max_card_count > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_collections_profile_id ON collections (profile_id);

-- +goose Down
DROP TABLE IF EXISTS collections;
