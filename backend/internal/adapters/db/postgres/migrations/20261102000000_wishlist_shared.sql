-- +goose Up
-- The Wishlist itself (GLOSSARY.md: Wishlist; ADR-0025): one row per profile, holding the
-- Wishlist's own state. Its items stay in wishlist_items.
CREATE TABLE wishlists (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    profile_id UUID NOT NULL UNIQUE REFERENCES user_profiles (id) ON DELETE CASCADE,
    shared BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS wishlists;
