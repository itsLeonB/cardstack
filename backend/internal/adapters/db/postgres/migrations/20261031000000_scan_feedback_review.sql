-- +goose Up

-- The Admin's decision on a Correction's photo (docs/adr/0027). NULL review_status
-- means pending, so every row recorded before this migration starts pending and
-- nothing already recorded changes. reviewed_by is the Admin's profile; it is
-- cleared, not cascaded, if that profile is deleted, so the decision outlives it.
ALTER TABLE scan_feedback
    ADD COLUMN review_status TEXT CHECK (review_status IN ('approved', 'rejected')),
    ADD COLUMN reviewed_by UUID REFERENCES user_profiles (id) ON DELETE SET NULL,
    ADD COLUMN reviewed_at TIMESTAMPTZ,
    ADD CONSTRAINT scan_feedback_review_status_set_at CHECK ((review_status IS NULL) = (reviewed_at IS NULL));

-- The review queue: photos still pending, oldest first.
CREATE INDEX idx_scan_feedback_review_queue ON scan_feedback (created_at, id)
    WHERE review_status IS NULL AND image_key IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_scan_feedback_review_queue;

ALTER TABLE scan_feedback
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS review_status;
