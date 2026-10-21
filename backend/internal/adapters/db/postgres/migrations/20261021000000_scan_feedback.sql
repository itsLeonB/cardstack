-- +goose Up

-- One row per confirmed Scan or Page Review cell whose Detection and final choice
-- are both kept, so the operator can see which Cards the matcher is least sure of
-- and which photos it handles badly (docs/adr/0022). Rows and their photos have no
-- expiry and no deletion path in the product. collection_id has no foreign key, so
-- deleting a Collection keeps its rows. profile_id does cascade, as Collections do.
CREATE TABLE scan_feedback (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    profile_id UUID NOT NULL REFERENCES user_profiles (id) ON DELETE CASCADE,
    collection_id UUID NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('card_scan', 'binder_page')),
    kind TEXT NOT NULL CHECK (kind IN ('confirmed', 'switched', 'reprint', 'empty', 'unidentified', 'placed')),
    -- The Detection exactly as /scan/match returned it, so the server does not
    -- need a schema change when the response gains a field.
    detection JSONB,
    card_id UUID REFERENCES cards (id),
    card_rank INTEGER CHECK (card_rank >= 1),
    card_rank_source TEXT CHECK (card_rank_source IN ('top', 'reprint')),
    image_key TEXT,
    -- A binder cell's Page, and its row and column on that Page.
    page INTEGER CHECK (page >= 1),
    slot_row INTEGER CHECK (slot_row >= 1),
    slot_column INTEGER CHECK (slot_column >= 1),
    grid_rows INTEGER CHECK (grid_rows >= 1),
    grid_columns INTEGER CHECK (grid_columns >= 1),
    corners JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (card_id IS NULL OR (card_rank IS NOT NULL AND card_rank_source IS NOT NULL)),
    CHECK (card_id IS NOT NULL OR (card_rank IS NULL AND card_rank_source IS NULL)),
    CHECK (
        (source = 'card_scan' AND page IS NULL AND slot_row IS NULL AND slot_column IS NULL
            AND grid_rows IS NULL AND grid_columns IS NULL AND corners IS NULL)
        OR (source = 'binder_page' AND page IS NOT NULL AND slot_row IS NOT NULL AND slot_column IS NOT NULL
            AND grid_rows IS NOT NULL AND grid_columns IS NOT NULL AND corners IS NOT NULL)
    )
);

-- Correction rate per chosen Card: a Card's rows grouped by kind.
CREATE INDEX idx_scan_feedback_card_kind ON scan_feedback (card_id, kind);

-- Corrections of one kind over time.
CREATE INDEX idx_scan_feedback_kind_created_at ON scan_feedback (kind, created_at);

-- A user's rows, for the operator's manual deletion of their data.
CREATE INDEX idx_scan_feedback_profile_id ON scan_feedback (profile_id);

-- +goose Down
DROP TABLE IF EXISTS scan_feedback;
