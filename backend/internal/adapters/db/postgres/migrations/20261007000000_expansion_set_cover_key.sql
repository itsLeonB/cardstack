-- +goose Up
-- cover_key is the object-store key of the Expansion Set cover pre-sized to
-- 128px at host time (ADR-0017); empty means "no cover yet". Additive only:
-- the running API keeps reading image_key until it is redeployed. Populated by
-- cmd/presize-images, never by the ingester.
ALTER TABLE expansion_sets ADD COLUMN cover_key text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE expansion_sets DROP COLUMN cover_key;
