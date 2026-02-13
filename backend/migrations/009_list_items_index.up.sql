-- Index on list_items.media_item_id speeds up cascading deletes when a
-- media_item is removed, and "which lists contain this item?" queries.
CREATE INDEX ON list_items (media_item_id);
