-- +goose Up
-- image_key is the object-store key of the hosted copy of the image (ADR-0016);
-- empty means "not hosted yet". source_image_url keeps the scraped address so a
-- lost upload can be re-fetched. The key starts empty, so the next ingest run
-- hosts every image.
ALTER TABLE cards ADD COLUMN source_image_url text NOT NULL DEFAULT '';
ALTER TABLE cards ADD COLUMN image_key text NOT NULL DEFAULT '';
UPDATE cards SET source_image_url = image_url;
ALTER TABLE cards DROP COLUMN image_url;

ALTER TABLE expansion_sets ADD COLUMN source_image_url text NOT NULL DEFAULT '';
ALTER TABLE expansion_sets ADD COLUMN image_key text NOT NULL DEFAULT '';
UPDATE expansion_sets SET source_image_url = image_url;
ALTER TABLE expansion_sets DROP COLUMN image_url;

-- +goose Down
ALTER TABLE expansion_sets ADD COLUMN image_url text NOT NULL DEFAULT '';
UPDATE expansion_sets SET image_url = source_image_url;
ALTER TABLE expansion_sets DROP COLUMN image_key;
ALTER TABLE expansion_sets DROP COLUMN source_image_url;

ALTER TABLE cards ADD COLUMN image_url text NOT NULL DEFAULT '';
UPDATE cards SET image_url = source_image_url;
ALTER TABLE cards DROP COLUMN image_key;
ALTER TABLE cards DROP COLUMN source_image_url;
