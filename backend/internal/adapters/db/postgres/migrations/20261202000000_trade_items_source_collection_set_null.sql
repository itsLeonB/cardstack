-- +goose Up

-- A Trade Item is part of the agreed deal and outlives its source Collection:
-- deleting the Collection must not silently drop the item, locked Trades
-- included. The spec's execution already copes with a gone source by falling
-- back to the participant's other Collections. SET NULL keeps the row and
-- records "collection deleted"; the (trade, side, card, source) unique index
-- treats NULLs as distinct, so uniqueness only applies while a source exists.
ALTER TABLE trade_items ALTER COLUMN source_collection_id DROP NOT NULL;

ALTER TABLE trade_items DROP CONSTRAINT trade_items_source_collection_id_fkey;
ALTER TABLE trade_items ADD CONSTRAINT trade_items_source_collection_id_fkey
    FOREIGN KEY (source_collection_id) REFERENCES collections (id) ON DELETE SET NULL;

-- +goose Down

ALTER TABLE trade_items DROP CONSTRAINT trade_items_source_collection_id_fkey;

-- Items whose source is already gone cannot satisfy the old NOT NULL; they
-- are dropped on the way back, as the old cascade would have done anyway.
DELETE FROM trade_items WHERE source_collection_id IS NULL;

ALTER TABLE trade_items ALTER COLUMN source_collection_id SET NOT NULL;
ALTER TABLE trade_items ADD CONSTRAINT trade_items_source_collection_id_fkey
    FOREIGN KEY (source_collection_id) REFERENCES collections (id) ON DELETE CASCADE;
