ALTER TABLE media_items
  ADD COLUMN IF NOT EXISTS current_progress INT,
  ADD COLUMN IF NOT EXISTS total_progress INT;
