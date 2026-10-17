-- +goose Up
-- created_by and updated_by name the Source that created a row and the Source
-- that last updated it, replacing curated (docs/adr/0021). A Source is a writer,
-- not a user. Existing rows came from the Pokémon Asia ingester, or were curated
-- by an Admin.
ALTER TABLE cards
    ADD COLUMN created_by text,
    ADD COLUMN updated_by text;
ALTER TABLE expansion_sets
    ADD COLUMN created_by text,
    ADD COLUMN updated_by text;

UPDATE cards SET created_by = 'pokemon-asia', updated_by = 'pokemon-asia';
UPDATE expansion_sets SET created_by = 'pokemon-asia', updated_by = 'pokemon-asia';
UPDATE cards SET created_by = 'admin', updated_by = 'admin' WHERE curated;
UPDATE expansion_sets SET created_by = 'admin', updated_by = 'admin' WHERE curated;

ALTER TABLE cards
    ALTER COLUMN created_by SET NOT NULL,
    ALTER COLUMN updated_by SET NOT NULL,
    ADD CONSTRAINT cards_created_by_source CHECK (created_by IN ('pokemon-asia', 'tcgcollector', 'admin')),
    ADD CONSTRAINT cards_updated_by_source CHECK (updated_by IN ('pokemon-asia', 'tcgcollector', 'admin'));
ALTER TABLE expansion_sets
    ALTER COLUMN created_by SET NOT NULL,
    ALTER COLUMN updated_by SET NOT NULL,
    ADD CONSTRAINT expansion_sets_created_by_source CHECK (created_by IN ('pokemon-asia', 'tcgcollector', 'admin')),
    ADD CONSTRAINT expansion_sets_updated_by_source CHECK (updated_by IN ('pokemon-asia', 'tcgcollector', 'admin'));

COMMENT ON COLUMN cards.created_by IS 'Source that created the row: pokemon-asia, tcgcollector or admin (docs/adr/0021). A Source, not a user.';
COMMENT ON COLUMN cards.updated_by IS 'Source that last updated the row: pokemon-asia, tcgcollector or admin (docs/adr/0021). A Source, not a user.';
COMMENT ON COLUMN expansion_sets.created_by IS 'Source that created the row: pokemon-asia, tcgcollector or admin (docs/adr/0021). A Source, not a user.';
COMMENT ON COLUMN expansion_sets.updated_by IS 'Source that last updated the row: pokemon-asia, tcgcollector or admin (docs/adr/0021). A Source, not a user.';

ALTER TABLE expansion_sets DROP COLUMN curated;
ALTER TABLE cards DROP COLUMN curated;

-- +goose Down
ALTER TABLE cards ADD COLUMN curated boolean NOT NULL DEFAULT false;
ALTER TABLE expansion_sets ADD COLUMN curated boolean NOT NULL DEFAULT false;
UPDATE cards SET curated = true WHERE updated_by = 'admin';
UPDATE expansion_sets SET curated = true WHERE updated_by = 'admin';

ALTER TABLE expansion_sets DROP COLUMN created_by, DROP COLUMN updated_by;
ALTER TABLE cards DROP COLUMN created_by, DROP COLUMN updated_by;
