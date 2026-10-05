-- +goose Up
-- image_key is the object-store key of the hosted Series logo (ADR-0017); empty
-- means "no logo yet". Additive only: the running API is unaffected. Populated
-- by cmd/host-series-images, never by the ingester.
ALTER TABLE series ADD COLUMN image_key text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE series DROP COLUMN image_key;
