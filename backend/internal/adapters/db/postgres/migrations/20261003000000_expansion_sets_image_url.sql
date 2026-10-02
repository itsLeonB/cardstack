-- +goose Up
ALTER TABLE expansion_sets ADD COLUMN image_url text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE expansion_sets DROP COLUMN image_url;
