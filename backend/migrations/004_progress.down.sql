ALTER TABLE media_items
  DROP COLUMN IF EXISTS current_progress,
  DROP COLUMN IF EXISTS total_progress;
