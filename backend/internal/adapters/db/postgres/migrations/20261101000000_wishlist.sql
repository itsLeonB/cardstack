-- +goose Up

-- The Wishlist (GLOSSARY.md: Wishlist): one item per Card a profile wants, with
-- its wanted quantity. Items never count toward Master Inventory, which reads
-- inventory_entries only.
-- shortcut: a profile's Wishlist has no row of its own yet, only its items; add
-- one when sharing needs a Slug (ticket 10).
CREATE TABLE wishlist_items (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    profile_id UUID NOT NULL REFERENCES user_profiles (id) ON DELETE CASCADE,
    card_id UUID NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
    wanted_quantity INTEGER NOT NULL CHECK (wanted_quantity >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT wishlist_items_profile_card_key UNIQUE (profile_id, card_id)
);

-- +goose Down

DROP TABLE IF EXISTS wishlist_items;
