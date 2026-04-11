-- Composite index for the most common list query: filter by type + status
CREATE INDEX ON media_items (user_id, media_type, status);

-- Sort/filter by updated_at (used by AutoMarkInactive and date-based sorts)
CREATE INDEX ON media_items (user_id, updated_at DESC);

-- Rating queries (stats distribution, sort by rating)
CREATE INDEX ON media_items (user_id, rating) WHERE rating IS NOT NULL;

-- Episode logs queried by watched_at (streak and monthly activity in stats)
CREATE INDEX ON tv_episode_logs (media_item_id, watched_at) WHERE watched_at IS NOT NULL;

-- Chapter logs queried by completed_at (streak and monthly activity in stats)
CREATE INDEX ON book_chapter_logs (media_item_id, completed_at) WHERE completed_at IS NOT NULL;
