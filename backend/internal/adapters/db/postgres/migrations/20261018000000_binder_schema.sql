-- +goose Up

-- Binders (phase 2 of card scanning; see GLOSSARY.md's Layout, Leaf, Page and
-- Slot, and docs/adr/0020). A Collection with a row here is a binder. Its
-- Pages are derived from the Leaves: Leaf n holds Pages 2n-1 (front) and 2n
-- (back).
CREATE TABLE collection_layouts (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    collection_id UUID NOT NULL UNIQUE REFERENCES collections (id) ON DELETE CASCADE,
    rows_per_page INTEGER NOT NULL CHECK (rows_per_page BETWEEN 1 AND 6),
    columns_per_page INTEGER NOT NULL CHECK (columns_per_page BETWEEN 1 AND 6),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- position is the Leaf's number in turning order. It is deferrable so that
-- inserting or removing a Leaf can shift every later position in one statement.
CREATE TABLE binder_leaves (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    collection_id UUID NOT NULL REFERENCES collections (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT binder_leaves_collection_position_key UNIQUE (collection_id, position) DEFERRABLE INITIALLY DEFERRED
);

-- Only occupied Slots have a row: an empty Slot is the absence of one. A row
-- holds either a Card or Unidentified, never both and never neither.
CREATE TABLE binder_slots (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    collection_id UUID NOT NULL REFERENCES collections (id) ON DELETE CASCADE,
    leaf_id UUID NOT NULL REFERENCES binder_leaves (id) ON DELETE CASCADE,
    side TEXT NOT NULL CHECK (side IN ('front', 'back')),
    slot_row INTEGER NOT NULL CHECK (slot_row >= 1),
    slot_column INTEGER NOT NULL CHECK (slot_column >= 1),
    card_id UUID REFERENCES cards (id),
    unidentified BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT binder_slots_card_or_unidentified CHECK ((card_id IS NULL) = unidentified),
    CONSTRAINT binder_slots_leaf_side_row_column_key UNIQUE (leaf_id, side, slot_row, slot_column)
);

CREATE INDEX binder_slots_collection_id_idx ON binder_slots (collection_id);
CREATE INDEX binder_slots_card_id_idx ON binder_slots (card_id);

-- +goose Down
DROP TABLE IF EXISTS binder_slots;
DROP TABLE IF EXISTS binder_leaves;
DROP TABLE IF EXISTS collection_layouts;
